package vmware

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/vmware/govmomi/cns"
	govmomitask "github.com/vmware/govmomi/task"
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

	// VSphereIsParaVirtualizedKey is the key for the para-virtualized indicator.
	// A value of "true" or "1" implies use of a para-virtualized CSI driver in the
	// cluster. When this is true then some operations will fail.
	// Note: it is up to the creator of the provider to determine if a para-virtualized environment exists.
	VSphereIsParaVirtualizedKey = "VSphereIsParaVirtualizedKey"

	defaultWaitTime   = 60 * time.Minute
	defaultRetryLimit = 30 * time.Minute

	vmWareTimeoutMinEnv = "VMWARE_GOM_TIMEOUT_MIN"

	// DescriptionTag is the prefix of the tags that should be placed in the snapshot description.
	// This constant must be used by clients, so changing this field may make already created snapshots inaccessible.
	DescriptionTag = "kanister.fcd.description"
	// VolumeIDListTag is the predefined name of the tag which contains volume ids separated by comma
	VolumeIDListTag = "kanister.fcd.volume-id"
)

var (
	vmWareTimeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute

	ErrNotSupportedWithParaVirtualizedVolumes = stderrors.New("operation not supported with para-virtualized volumes")
)

// FcdProvider provides blockstorage.Provider
type FcdProvider struct {
	Gom               *vslm.GlobalObjectManager
	Cns               *cns.Client
	TagsSvc           *vapitags.Manager
	tagManager        tagManager
	categoryID        string
	isParaVirtualized bool
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
		Cns:               cnsCli,
		Gom:               gom,
		TagsSvc:           tm,
		tagManager:        tm,
		isParaVirtualized: configIsParaVirtualized(config),
	}, nil
}

func configIsParaVirtualized(config map[string]string) bool {
	if isParaVirtualizedVal, ok := config[VSphereIsParaVirtualizedKey]; ok {
		if strings.ToLower(isParaVirtualizedVal) == "true" || isParaVirtualizedVal == "1" {
			return true
		}
	}
	return false
}

// IsParaVirtualized is not part of blockstorage.Provider.
func (p *FcdProvider) IsParaVirtualized() bool {
	return p.isParaVirtualized
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
	if p.IsParaVirtualized() {
		return nil, errors.WithStack(ErrNotSupportedWithParaVirtualizedVolumes)
	}
	volID, snapshotID, err := SplitSnapshotFullID(snapshot.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to split snapshot full ID")
	}
	log.Debug().Print("CreateDiskFromSnapshot foo", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
	uid, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create UUID")
	}
	task, err := p.Gom.CreateDiskFromSnapshot(ctx, vimID(volID), vimID(snapshotID), uid.String(), nil, nil, "")
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

var reVslmSyncFaultFatal = regexp.MustCompile("Change tracking invalid or disk in use")

// SnapshotCreate is part of blockstorage.Provider
func (p *FcdProvider) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	var res types.AnyType
	description := generateSnapshotDescription(tags)
	err := wait.PollImmediate(time.Second, defaultRetryLimit, func() (bool, error) {
		timeOfCreateSnapshotCall := time.Now()
		var createErr error
		res, createErr = p.createSnapshotAndWaitForCompletion(volume, ctx, description)
		if createErr == nil {
			return true, nil
		}

		// it's possible that snapshot was created despite of SOAP errors,
		// we're trying to find snapshot created in the current iteration(using timeOfCreateSnapshotCall)
		// so we won't reuse snapshots created in previous runs
		foundSnapId := p.getCreatedSnapshotID(ctx, description, volume.ID, timeOfCreateSnapshotCall)
		if foundSnapId != nil {
			res = *foundSnapId
			log.Error().WithError(createErr).Print("snapshot created with errors")
			return true, nil
		}

		if !soap.IsVimFault(createErr) {
			return false, errors.Wrap(createErr, "Failed to wait on task")
		}

		// snapshot wasn't created, handle the different SOAP errors then retry
		switch t := soap.ToVimFault(createErr).(type) {
		case *types.InvalidState:
			log.Error().WithError(createErr).Print("There is some operation, other than this CreateSnapshot invocation, on the VM attached still being protected by its VM state. Will retry", field.M{"VolumeID": volume.ID})
			return false, nil
		case *vslmtypes.VslmSyncFault: // potentially can leak snapshots
			log.Error().Print(fmt.Sprintf("VslmSyncFault: %#v", t))
			if !(govmomiError{createErr}).Matches(reVslmSyncFaultFatal) {
				log.Error().Print(fmt.Sprintf("CreateSnapshot failed with VslmSyncFault. Will retry: %s", (govmomiError{createErr}).Format()), field.M{"VolumeID": volume.ID})
				return false, nil
			}
			return false, errors.Wrap(createErr, "CreateSnapshot failed with VslmSyncFault. A snapshot may have been created by this failed operation")
		case *types.NotFound:
			log.Error().WithError(createErr).Print("CreateSnapshot failed with NotFound error. Will retry", field.M{"VolumeID": volume.ID})
			return false, nil
		default:
			return false, errors.Wrap(createErr, "Failed to wait on task")
		}
	})
	if err != nil {
		log.Error().WithError(err).Print(fmt.Sprintf("Failed to create snapshot for FCD %s: %s", volume.ID, govmomiError{err}.Format()))
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to create snapshot for FCD %s", volume.ID))
	}
	id, ok := res.(types.ID)
	if !ok {
		return nil, errors.New(fmt.Sprintf("Unexpected type returned for FCD %s", volume.ID))
	}
	snap, err := p.SnapshotGet(ctx, SnapshotFullID(volume.ID, id.Id))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to get snapshot %s:%s", volume.ID, id.Id))
	}
	log.Debug().Print("SnapshotCreate complete", field.M{"VolumeID": volume.ID, "SnapshotID": snap.ID})
	// We don't get size information from `SnapshotGet` - so set this to the volume size for now
	if snap.SizeInBytes == 0 {
		snap.SizeInBytes = volume.SizeInBytes
	}
	snap.Volume = &volume

	if err = p.SetTags(ctx, snap, getTagsWithoutDescription(tags)); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to set tags for snapshot %s:%s", volume.ID, snap.ID))
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
			snapID := vimID(snapshotID)
			log.Debug().Print("RetrieveMetadata: " + volID + "," + snapshotID)
			kvs, err := p.Gom.RetrieveMetadata(ctx, vimID(volID), &snapID, "")
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get snapshot metadata")
			}
			log.Debug().Print("RetrieveMetadata done: " + volID + "," + snapshotID)
			tags := convertKeyValueToTags(kvs)
			additionalTags, err := p.getSnapshotTags(ctx, id)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get snapshot tags")
			}
			tags = append(tags, additionalTags...)
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

// SnapshotsList is part of blockstorage.Provider
func (p *FcdProvider) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	if filterStr := generateSnapshotDescription(tags); filterStr != "" {
		volumeIDs := strings.Split(tags[VolumeIDListTag], ",")
		if len(volumeIDs) == 0 {
			return nil, errors.New("vSphere can't list by description without list of volumes. Cannot list snapshots")
		}
		return p.snapshotsListByDescription(ctx, volumeIDs, filterStr)
	}
	return p.snapshotsListByTag(ctx, tags)
}

func (p *FcdProvider) createSnapshotAndWaitForCompletion(volume blockstorage.Volume, ctx context.Context, description string) (types.AnyType, error) {
	log.Debug().Print("CreateSnapshot", field.M{"VolumeID": volume.ID})
	task, err := p.Gom.CreateSnapshot(ctx, vimID(volume.ID), description)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSnapshot task creation failure")
	}
	log.Debug().Print("Started CreateSnapshot task", field.M{"VolumeID": volume.ID})
	return task.Wait(ctx, vmWareTimeout)
}

func (p *FcdProvider) getCreatedSnapshotID(ctx context.Context, description string, volID string, notEarlierThan time.Time) *types.ID {
	var filteredSns []*blockstorage.Snapshot
	sns, err := p.snapshotsListByDescription(ctx, []string{volID}, description)
	if err != nil {
		log.Error().WithError(err).Print("Failed to list when checking failed creation")
		return nil
	}

	for _, sn := range sns {
		if notEarlierThan.Before((time.Time)(sn.CreationTime)) {
			filteredSns = append(filteredSns, sn)
		}
	}

	if len(filteredSns) == 1 {
		_, snapID, err := SplitSnapshotFullID(filteredSns[0].ID)
		if err != nil {
			log.Error().WithError(err)
			return nil
		}
		return &types.ID{
			Id: snapID,
		}
	}

	if len(filteredSns) > 1 {
		log.Error().Print(fmt.Sprintf("More than one snapshot was found, IDs: %s", strings.Join(getSnapshotsIDs(filteredSns), ",")))
	}
	return nil
}

func generateSnapshotDescription(tags map[string]string) string {
	var tagsAsStr []string
	for name, value := range tags {
		if strings.HasPrefix(name, DescriptionTag) {
			tagsAsStr = append(tagsAsStr, fmt.Sprintf("%s:%s", name, value))
		}
	}
	return strings.Join(tagsAsStr, ",")
}

func getTagsWithoutDescription(tags map[string]string) map[string]string {
	result := make(map[string]string, len(tags))
	for name, value := range tags {
		if !strings.HasPrefix(name, DescriptionTag) {
			result[name] = value
		}
	}
	return result
}

func getSnapshotsIDs(snapshots []*blockstorage.Snapshot) []string {
	result := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		result = append(result, snapshot.ID)
	}
	return result
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

func (p *FcdProvider) getSnapshotTags(ctx context.Context, fullSnapshotID string) ([]*blockstorage.KeyValue, error) {
	if p.categoryID == "" {
		log.Debug().Print("vSphere snapshot tagging is disabled (categoryID not set). Cannot get snapshot tags")
		return nil, nil
	}
	categoryTags, err := p.tagManager.GetTagsForCategory(ctx, p.categoryID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list tags")
	}
	return p.getTagsFromSnapshotID(categoryTags, fullSnapshotID)
}

func (p *FcdProvider) snapshotsListByDescription(ctx context.Context, volumeIDs []string, filterStr string) ([]*blockstorage.Snapshot, error) {
	var result []*blockstorage.Snapshot
	for _, volID := range volumeIDs {
		snapshots, _ := p.Gom.RetrieveSnapshotInfo(ctx, vimID(volID))
		for _, snapshot := range snapshots {
			if snapshot.Description == filterStr {
				sn, err := convertFromObjectToSnapshot(&snapshot, volID)
				if err != nil {
					return nil, errors.Wrap(err, "Failed to convert object to snapshot")
				}
				result = append(result, sn)
			}
		}
	}

	return result, nil
}

func (p *FcdProvider) snapshotsListByTag(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
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

// Helper to parse an error code returned by the govmomi repo.
type govmomiError struct {
	err error
}

func (ge govmomiError) Format() string {
	msgs := ge.ExtractMessages()
	switch len(msgs) {
	case 0:
		return ""
	case 1:
		return msgs[0]
	}
	return fmt.Sprintf("[%s]", strings.Join(msgs, "; "))
}

func (ge govmomiError) ExtractMessages() []string {
	err := ge.err

	if err == nil {
		return nil
	}

	msgs := []string{}
	if reason := err.Error(); reason != "" {
		msgs = append(msgs, reason)
	}

	// unwrap to a type handled
	foundHandledErrorType := false
	for err != nil && !foundHandledErrorType {
		switch err.(type) {
		case govmomitask.Error:
			foundHandledErrorType = true
		default:
			if soap.IsSoapFault(err) {
				foundHandledErrorType = true
			} else if soap.IsVimFault(err) {
				foundHandledErrorType = true
			} else {
				err = errors.Unwrap(err)
			}
		}
	}

	if err != nil {
		var faultMsgs []types.LocalizableMessage
		switch e := err.(type) {
		case govmomitask.Error:
			if e.Description != nil {
				msgs = append(msgs, e.Description.Message)
			}
			faultMsgs = e.LocalizedMethodFault.Fault.GetMethodFault().FaultMessage
		default:
			if soap.IsSoapFault(err) {
				detail := soap.ToSoapFault(err).Detail.Fault
				if f, ok := detail.(types.BaseMethodFault); ok {
					faultMsgs = f.GetMethodFault().FaultMessage
				}
			} else if soap.IsVimFault(err) {
				f := soap.ToVimFault(err)
				faultMsgs = f.GetMethodFault().FaultMessage
			}
		}

		for _, m := range faultMsgs {
			if m.Message != "" && !strings.HasPrefix(m.Message, "[context]") {
				msgs = append(msgs, fmt.Sprintf("%s (%s)", m.Message, m.Key))
			}
			for _, a := range m.Arg {
				msgs = append(msgs, fmt.Sprintf("%s", a.Value))
			}
		}
	}

	return msgs
}

func (ge govmomiError) Matches(pat *regexp.Regexp) bool {
	for _, m := range ge.ExtractMessages() {
		if pat.MatchString(m) {
			return true
		}
	}

	return false
}
