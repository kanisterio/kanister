package vmware

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/vmware/govmomi/cns"
	"github.com/vmware/govmomi/vapi/rest"
	vapitags "github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vslm"
	vslmtypes "github.com/vmware/govmomi/vslm/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

var _ blockstorage.Provider = (*FcdProvider)(nil)

const (
	// VSphereLoginURLKey represents key in config to establish connection.
	// It should contain the username and the password.
	VSphereLoginURLKey = "VSphereLoginURL"

	// VSphereEndpointKey represents key for the login endpoint.
	VSphereEndpointKey = "VSphereEndpoint"
	// VSphereUsernameKey represents key for the username.
	VSphereUsernameKey = "VSphereUsername"
	// VSpherePasswordKey represents key for the password.
	VSpherePasswordKey = "VSpherePasswordKey"

	noDescription     = ""
	defaultWaitTime   = 60 * time.Minute
	defaultRetryLimit = 30 * time.Minute

	vmWareTimeoutMinEnv = "VMWARE_GOM_TIMEOUT_MIN"
)

var (
	vmWareTimeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
)

// FcdProvider provides blockstorage.Provider
type FcdProvider struct {
	Gom        *vslm.GlobalObjectManager
	Cns        *cns.Client
	TagsSvc    *vapitags.Manager
	tagManager tagManager
	categoryID string
}

// NewProvider creates new VMWare FCD provider with the config.
// URL taken from config helps to establish connection.
func NewProvider(config map[string]string) (blockstorage.Provider, error) {
	ep, ok := config[VSphereEndpointKey]
	if !ok {
		return nil, errors.New("Failed to find VSphere endpoint value")
	}
	username, ok := config[VSphereUsernameKey]
	if !ok {
		return nil, errors.New("Failed to find VSphere username value")
	}
	password, ok := config[VSpherePasswordKey]
	if !ok {
		return nil, errors.New("Failed to find VSphere password value")
	}

	u := &url.URL{Scheme: "https", Host: ep, Path: "/sdk"}
	soapCli := soap.NewClient(u, true)
	ctx := context.Background()
	cli, err := vim25.NewClient(ctx, soapCli)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create VIM client")
	}
	req := types.Login{
		This: *cli.ServiceContent.SessionManager,
	}
	req.UserName = username
	req.Password = password
	_, err = methods.Login(ctx, cli, &req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to login")
	}
	cnsCli, err := cns.NewClient(ctx, cli)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create CNS client")
	}
	vslmCli, err := vslm.NewClient(ctx, cli)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create VSLM client")
	}
	c := rest.NewClient(cli)
	err = c.Login(ctx, url.UserPassword(username, password))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to login to VAPI rest client")
	}
	tm := vapitags.NewManager(c)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create tag manager")
	}
	gom := vslm.NewGlobalObjectManager(vslmCli)
	return &FcdProvider{
		Cns:        cnsCli,
		Gom:        gom,
		TagsSvc:    tm,
		tagManager: tm,
	}, nil
}

// Type is part of blockstorage.Provider
func (p *FcdProvider) Type() blockstorage.Type {
	return blockstorage.TypeFCD
}

// Type is part of blockstorage.Provider
func (p *FcdProvider) SetCategoryID(categoryID string) {
	p.categoryID = categoryID
}

// VolumeCreate is part of blockstorage.Provider
func (p *FcdProvider) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

// VolumeCreateFromSnapshot is part of blockstorage.Provider
func (p *FcdProvider) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	volID, snapshotID, err := SplitSnapshotFullID(snapshot.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to split snapshot full ID")
	}
	log.Debug().Print("CreateDiskFromSnapshot foo", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
	uid := uuid.NewV1().String()
	task, err := p.Gom.CreateDiskFromSnapshot(ctx, vimID(volID), vimID(snapshotID), uid, nil, nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create disk from snapshot")
	}
	log.Debug().Print("Started CreateDiskFromSnapshot task", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
	res, err := task.Wait(ctx, vmWareTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to wait on task")
	}
	if res == nil {
		return nil, errors.Errorf("vSphere task did not complete. TaskRefType: %s, TaskRefValue: %s, VolID: %s, SnapshotID: %s, NewVolID: %s",
			task.ManagedObjectReference.Type, task.ManagedObjectReference.Value, volID, snapshotID, uid)
	}
	log.Debug().Print("CreateDiskFromSnapshot task complete", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
	obj, ok := res.(types.VStorageObject)
	if !ok {
		return nil, errors.New(fmt.Sprintf("Wrong type returned for vSphere. Type: %T, Value: %v", res, res))
	}
	vol, err := p.VolumeGet(ctx, obj.Config.Id.Id, "")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get volume")
	}
	tagsCNS := make(map[string]string)
	tagsCNS["cns.tag"] = "1"
	tags = ktags.Union(tags, tagsCNS)
	if err = p.SetTags(ctx, vol, tags); err != nil {
		return nil, errors.Wrap(err, "Failed to set tags")
	}
	log.Debug().Print("CreateDiskFromSnapshot complete", field.M{"SnapshotID": snapshotID, "NewVolumeID": vol.ID})
	return p.VolumeGet(ctx, vol.ID, "")
}

// VolumeDelete is part of blockstorage.Provider
func (p *FcdProvider) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	task, err := p.Gom.Delete(ctx, vimID(volume.ID))
	if err != nil {
		return errors.Wrap(err, "Failed to delete the disk")
	}
	_, err = task.Wait(ctx, vmWareTimeout)
	return err
}

// VolumeGet is part of blockstorage.Provider
func (p *FcdProvider) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	obj, err := p.Gom.Retrieve(ctx, vimID(id))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query the disk")
	}
	kvs, err := p.Gom.RetrieveMetadata(ctx, vimID(id), nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get volume metadata")
	}
	vol, err := convertFromObjectToVolume(obj)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert object to volume")
	}
	vol.Tags = convertKeyValueToTags(kvs)
	return vol, nil
}

// SnapshotCopy is part of blockstorage.Provider
func (p *FcdProvider) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

// SnapshotCopyWithArgs is part of blockstorage.Provider
func (p *FcdProvider) SnapshotCopyWithArgs(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot, args map[string]string) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Copy Snapshot with Args not implemented")
}

// SnapshotCreate is part of blockstorage.Provider
func (p *FcdProvider) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	var res types.AnyType
	err := wait.PollImmediate(time.Second, defaultRetryLimit, func() (bool, error) {
		log.Debug().Print("CreateSnapshot", field.M{"VolumeID": volume.ID})
		task, lerr := p.Gom.CreateSnapshot(ctx, vimID(volume.ID), noDescription)
		if lerr != nil {
			return false, errors.Wrap(lerr, "Failed to create snapshot")
		}
		log.Debug().Print("Started CreateSnapshot task", field.M{"VolumeID": volume.ID})
		res, lerr = task.Wait(ctx, vmWareTimeout)
		if lerr != nil {
			if soap.IsVimFault(lerr) {
				switch soap.ToVimFault(lerr).(type) {
				case *types.InvalidState:
					log.Error().WithError(lerr).Print("There is some operation, other than this CreateSnapshot invocation, on the VM attached still being protected by its VM state. Will retry")
					return false, nil
				case *vslmtypes.VslmSyncFault:
					log.Error().WithError(lerr).Print("CreateSnapshot failed with VslmSyncFault error possibly due to race between concurrent DeleteSnapshot invocation. Will retry")
					return false, nil
				case *types.NotFound:
					log.Error().WithError(lerr).Print("CreateSnapshot failed with NotFound error. Will retry")
					return false, nil
				}
			}
			return false, errors.Wrap(lerr, "Failed to wait on task")
		}
		log.Debug().Print("CreateSnapshot task complete", field.M{"VolumeID": volume.ID})
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create snapshot")
	}
	id, ok := res.(types.ID)
	if !ok {
		return nil, errors.New("Unexpected type")
	}
	snap, err := p.SnapshotGet(ctx, SnapshotFullID(volume.ID, id.Id))
	if err != nil {
		return nil, err
	}
	log.Debug().Print("SnapshotCreate complete", field.M{"VolumeID": volume.ID, "SnapshotID": snap.ID})
	// We don't get size information from `SnapshotGet` - so set this to the volume size for now
	if snap.SizeInBytes == 0 {
		snap.SizeInBytes = volume.SizeInBytes
	}
	snap.Volume = &volume

	if err = p.SetTags(ctx, snap, tags); err != nil {
		return nil, errors.Wrap(err, "Failed to set tags")
	}

	return snap, nil
}

// SnapshotCreateWaitForCompletion is part of blockstorage.Provider
func (p *FcdProvider) SnapshotCreateWaitForCompletion(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	return nil
}

// SnapshotDelete is part of blockstorage.Provider
func (p *FcdProvider) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	volID, snapshotID, err := SplitSnapshotFullID(snapshot.ID)
	if err != nil {
		return errors.Wrap(err, "Cannot infer volume ID from full snapshot ID")
	}
	return wait.PollImmediate(time.Second, defaultRetryLimit, func() (bool, error) {
		log.Debug().Print("SnapshotDelete", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
		task, lerr := p.Gom.DeleteSnapshot(ctx, vimID(volID), vimID(snapshotID))
		if lerr != nil {
			if soap.IsSoapFault(lerr) {
				soapFault := soap.ToSoapFault(lerr)
				receivedFault := soapFault.Detail.Fault
				_, ok := receivedFault.(types.NotFound)
				if ok {
					log.Debug().Print("The FCD id was not found in VC during deletion, assuming success", field.M{"err": lerr, "VolumeID": volID, "SnapshotID": snapshotID})
					return true, nil
				}
			}
			return false, errors.Wrap(lerr, "Failed to create a task for the DeleteSnapshot invocation on an IVD Protected Entity")
		}
		log.Debug().Print("Started SnapshotDelete task", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
		_, lerr = task.Wait(ctx, vmWareTimeout)
		if lerr != nil {
			// The following error handling was pulled from https://github.com/vmware-tanzu/astrolabe/blob/91eeed4dcf77edd1387a25e984174f159d66fedb/pkg/ivd/ivd_protected_entity.go#L433
			if soap.IsVimFault(lerr) {
				switch soap.ToVimFault(lerr).(type) {
				case *types.InvalidArgument:
					log.Error().WithError(lerr).Print("Disk doesn't have given snapshot due to the snapshot stamp being removed in the previous DeleteSnapshot operation which failed with an InvalidState fault. It will be resolved by the next snapshot operation on the same VM. Will NOT retry")
					return true, nil
				case *types.NotFound:
					log.Error().WithError(lerr).Print("There is a temporary catalog mismatch due to a race condition with one another concurrent DeleteSnapshot operation. It will be resolved by the next consolidateDisks operation on the same VM. Will NOT retry")
					return true, nil
				case *types.InvalidState:
					log.Error().WithError(lerr).Print("There is some operation, other than this DeleteSnapshot invocation, on the same VM still being protected by its VM state. Will retry")
					return false, nil
				case *types.TaskInProgress:
					log.Error().WithError(lerr).Print("There is some other InProgress operation on the same VM. Will retry")
					return false, nil
				case *types.FileLocked:
					log.Error().WithError(lerr).Print("An error occurred while consolidating disks: Failed to lock the file. Will retry")
					return false, nil
				}
			}
			return false, errors.Wrap(lerr, "Failed to wait on task")
		}
		log.Debug().Print("SnapshotDelete task complete", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
		err = p.deleteSnapshotTags(ctx, snapshot)
		if err != nil {
			return false, errors.Wrap(err, "Failed to delete snapshot tags")
		}
		return true, nil
	})
}

// SnapshotGet is part of blockstorage.Provider
func (p *FcdProvider) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	volID, snapshotID, err := SplitSnapshotFullID(id)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot infer volume ID from full snapshot ID")
	}
	log.Debug().Print("RetrieveSnapshotInfo:" + volID)
	results, err := p.Gom.RetrieveSnapshotInfo(ctx, vimID(volID))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get snapshot info")
	}
	log.Debug().Print("RetrieveSnapshotInfo done:" + volID)

	for _, result := range results {
		if result.Id.Id == snapshotID {
			snapshot, err := convertFromObjectToSnapshot(&result, volID)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to convert object to snapshot")
			}
			log.Debug().Print("RetrieveMetadata: " + volID + "," + snapshotID)
			tags, err := p.getSnapshotTags(ctx, id, volID)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get snapshot metadata")
			}
			log.Debug().Print("RetrieveMetadata done: " + volID + "," + snapshotID)
			snapshot.Tags = tags
			return snapshot, nil
		}
	}
	return nil, errors.New("Failed to find snapshot")
}

// SetTags is part of blockstorage.Provider
func (p *FcdProvider) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch r := resource.(type) {
	case *blockstorage.Volume:
		return p.setTagsVolume(ctx, r, tags)
	case *blockstorage.Snapshot:
		return p.setSnapshotTags(ctx, r, tags)
	default:
		return errors.New("Unsupported type for resource")
	}
}

func (p *FcdProvider) setTagsVolume(ctx context.Context, volume *blockstorage.Volume, tags map[string]string) error {
	if volume == nil {
		return errors.New("Empty volume")
	}
	task, err := p.Gom.UpdateMetadata(ctx, vimID(volume.ID), convertTagsToKeyValue(tags), nil)
	if err != nil {
		return errors.Wrap(err, "Failed to update metadata")
	}
	_, err = task.Wait(ctx, vmWareTimeout)
	if err != nil {
		return errors.Wrap(err, "Failed to wait on task")
	}
	return nil
}

// GetOrCreateCategory takes a category name and attempts to get or create the category
// it returns the category ID.
func (p *FcdProvider) GetOrCreateCategory(ctx context.Context, categoryName string) (string, error) {
	id, err := p.GetCategoryID(ctx, categoryName)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			id, err := p.tagManager.CreateCategory(ctx, &vapitags.Category{
				Name:        categoryName,
				Cardinality: "SINGLE",
			})
			if err != nil {
				return "", errors.Wrap(err, "Failed to create category")
			}
			return id, nil
		}
		return "", err
	}
	return id, nil
}

// GetCategoryID takes a category name and returns the category ID if it finds it.
func (p *FcdProvider) GetCategoryID(ctx context.Context, categoryName string) (string, error) {
	cat, err := p.tagManager.GetCategory(ctx, categoryName)
	if err != nil {
		return "", errors.Wrap(err, "Failed to find category")
	}
	return cat.ID, nil
}

// snapshotTag is the struct that will be used to create vmware tags
// the tags are of the form volid:snapid:tag:value
// these tags are assigned to a predefined category that is initialized by the FcdProvider
type snapshotTag struct {
	volid  string
	snapid string
	key    string
	value  string
}

func (t *snapshotTag) String() string {
	volid := strings.ReplaceAll(t.volid, ":", "-")
	snapid := strings.ReplaceAll(t.snapid, ":", "-")
	key := strings.ReplaceAll(t.key, ":", "-")
	value := strings.ReplaceAll(t.value, ":", "-")
	return fmt.Sprintf("%s:%s:%s:%s", volid, snapid, key, value)
}

func (t *snapshotTag) Parse(tag string) error {
	parts := strings.Split(tag, ":")
	if len(parts) != 4 {
		return errors.Errorf("Malformed tag (%s)", tag)
	}
	t.volid, t.snapid, t.key, t.value = parts[0], parts[1], parts[2], parts[3]
	return nil
}

// setSnapshotTags sets tags for a snapshot
func (p *FcdProvider) setSnapshotTags(ctx context.Context, snapshot *blockstorage.Snapshot, tags map[string]string) error {
	if p.categoryID == "" {
		log.Debug().Print("vSphere snapshot tagging is disabled")
		return nil
	}
	if snapshot == nil {
		return errors.New("Empty snapshot")
	}
	volID, snapID, err := SplitSnapshotFullID(snapshot.ID)
	if err != nil {
		return errors.Wrap(err, "Cannot infer volumeID and snapshotID from full snapshot ID")
	}

	for k, v := range tags {
		tag := &snapshotTag{volID, snapID, k, v}
		_, err = p.tagManager.CreateTag(ctx, &vapitags.Tag{
			CategoryID: p.categoryID,
			Name:       tag.String(),
		})
		if err != nil && !strings.Contains(err.Error(), "ALREADY_EXISTS") {
			return errors.Wrapf(err, "Failed to create tag (%s) for categoryID (%s) ", tag, p.categoryID)
		}
	}
	return nil
}

func (p *FcdProvider) deleteSnapshotTags(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	if p.categoryID == "" {
		log.Debug().Print("vSphere snapshot tagging is disabled (categoryID not set). Cannot list snapshots")
		return nil
	}
	if snapshot == nil {
		return errors.New("Empty snapshot")
	}
	volID, snapID, err := SplitSnapshotFullID(snapshot.ID)
	if err != nil {
		return errors.Wrap(err, "Cannot infer volumeID and snapshotID from full snapshot ID")
	}
	categoryTags, err := p.tagManager.GetTagsForCategory(ctx, p.categoryID)
	if err != nil {
		return errors.Wrap(err, "Failed to list tags")
	}
	for _, tag := range categoryTags {
		parsedTag := &snapshotTag{}
		err := parsedTag.Parse(tag.Name)
		if err != nil {
			return errors.Wrapf(err, "Failed to parse tag (%s)", tag.Name)
		}
		if parsedTag.snapid == snapID && parsedTag.volid == volID {
			err := p.tagManager.DeleteTag(ctx, &tag)
			if err != nil {
				return errors.Wrapf(err, "Failed to delete tag (%s)", tag.Name)
			}
		}
	}
	return nil
}

// VolumesList is part of blockstorage.Provider
func (p *FcdProvider) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (p *FcdProvider) getSnapshotTags(ctx context.Context, fullSnapshotID string, volid string) ([]*blockstorage.KeyValue, error) {
	if p.categoryID == "" {
		if p.Gom == nil {
			return nil, errors.New("GlobalObjectManager not initialized")
		}
		kvs, err := p.Gom.RetrieveMetadata(ctx, vimID(volid), nil, "")
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get volume metadata")
		}
		return convertKeyValueToTags(kvs), nil
	}
	categoryTags, err := p.tagManager.GetTagsForCategory(ctx, p.categoryID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list tags")
	}
	return p.getTagsFromSnapshotID(categoryTags, fullSnapshotID)
}

// SnapshotsList is part of blockstorage.Provider
func (p *FcdProvider) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	if p.categoryID == "" {
		log.Debug().Print("vSphere snapshot tagging is disabled (categoryID not set). Cannot list snapshots")
		return nil, nil
	}

	categoryTags, err := p.tagManager.GetTagsForCategory(ctx, p.categoryID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list tags")
	}

	snapshotIDs, err := p.getSnapshotIDsFromTags(categoryTags, tags)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get snapshotIDs from tags")
	}

	var snapshots []*blockstorage.Snapshot
	if len(snapshotIDs) > 0 {
		for _, snapshotID := range snapshotIDs {
			snapshot, err := p.SnapshotGet(ctx, snapshotID)
			if err != nil {
				return nil, err
			}
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots, nil
}

func (p *FcdProvider) getTagsFromSnapshotID(categoryTags []vapitags.Tag, fullSnapshotID string) ([]*blockstorage.KeyValue, error) {
	tags := map[string]string{}
	for _, catTag := range categoryTags {
		parsedTag := &snapshotTag{}
		if err := parsedTag.Parse(catTag.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed to parse tag")
		}
		snapId := SnapshotFullID(parsedTag.volid, parsedTag.snapid)
		if snapId == fullSnapshotID {
			tags[parsedTag.key] = parsedTag.value
		}
	}
	return blockstorage.MapToKeyValue(tags), nil
}

func (p *FcdProvider) getSnapshotIDsFromTags(categoryTags []vapitags.Tag, tags map[string]string) ([]string, error) {
	snapshotTagMap := map[string]map[string]string{}
	for _, catTag := range categoryTags {
		parsedTag := &snapshotTag{}
		if err := parsedTag.Parse(catTag.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed to parse tag")
		}
		snapId := SnapshotFullID(parsedTag.volid, parsedTag.snapid)
		if _, ok := snapshotTagMap[snapId]; !ok {
			snapshotTagMap[snapId] = map[string]string{}
		}
		snapshotTagMap[snapId][parsedTag.key] = parsedTag.value
	}

	snapshotIDs := []string{}
	for snapshotID, snapshotTags := range snapshotTagMap {
		tagsMatch := true
		for k, v := range tags {
			if val, ok := snapshotTags[k]; !ok || val != v {
				tagsMatch = false
				break
			}
		}
		if tagsMatch {
			snapshotIDs = append(snapshotIDs, snapshotID)
		}
	}
	return snapshotIDs, nil
}

func getEnvAsIntOrDefault(envKey string, def int) int {
	if v, ok := os.LookupEnv(envKey); ok {
		iv, err := strconv.Atoi(v)
		if err == nil && iv > 0 {
			return iv
		}
		log.Debug().Print("Using default timeout value for vSphere because of invalid environment variable", field.M{"envVar": v})
	}

	return def
}

type tagManager interface {
	GetCategory(ctx context.Context, id string) (*vapitags.Category, error)
	CreateCategory(ctx context.Context, category *vapitags.Category) (string, error)
	CreateTag(ctx context.Context, tag *vapitags.Tag) (string, error)
	GetTagsForCategory(ctx context.Context, id string) ([]vapitags.Tag, error)
	DeleteTag(ctx context.Context, tag *vapitags.Tag) error
}
