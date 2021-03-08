package vmware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/vmware/govmomi/cns"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vslm"
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
	defaultWaitTime   = 10 * time.Minute
	defaultRetryLimit = 30 * time.Minute
	invalidStateError = "The operation is not allowed in the current state"
)

// FcdProvider provides blockstorage.Provider
type FcdProvider struct {
	Gom *vslm.GlobalObjectManager
	Cns *cns.Client
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
	gom := vslm.NewGlobalObjectManager(vslmCli)
	return &FcdProvider{
		Cns: cnsCli,
		Gom: gom,
	}, nil
}

// Type is part of blockstorage.Provider
func (p *FcdProvider) Type() blockstorage.Type {
	return blockstorage.TypeFCD
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
	task, err := p.Gom.CreateDiskFromSnapshot(ctx, vimID(volID), vimID(snapshotID), uuid.NewV1().String(), nil, nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create disk from snapshot")
	}
	log.Debug().Print("Started CreateDiskFromSnapshot task", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
	res, err := task.Wait(ctx, defaultWaitTime)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to wait on task")
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
	_, err = task.Wait(ctx, defaultWaitTime)
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
	encodedTags, err := json.Marshal(tags)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to JSON encode tags")
	}
	err = wait.PollImmediate(time.Second, defaultRetryLimit, func() (bool, error) {
		log.Debug().Print("CreateSnapshot", field.M{"VolumeID": volume.ID})
		task, lerr := p.Gom.CreateSnapshot(ctx, vimID(volume.ID), string(encodedTags))
		if lerr != nil {
			return false, errors.Wrap(lerr, "Failed to create snapshot")
		}
		log.Debug().Print("Started CreateSnapshot task", field.M{"VolumeID": volume.ID})
		res, lerr = task.Wait(ctx, defaultWaitTime)
		if lerr != nil {
			if strings.Contains(lerr.Error(), invalidStateError) {
				log.Debug().Print("Retrying CreateSnapshot task", field.M{"VolumeID": volume.ID})
				// retry operation
				return false, nil
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
			return false, errors.Wrap(lerr, "Failed to delete snapshot")
		}
		log.Debug().Print("Started SnapshotDelete task", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
		_, lerr = task.Wait(ctx, defaultWaitTime)
		if lerr != nil {
			if strings.Contains(lerr.Error(), invalidStateError) {
				log.Debug().Print("Retrying SnapshotDelete task", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
				// retry operation
				return false, nil
			}
			return false, errors.Wrap(lerr, "Failed to wait on task")
		}
		log.Debug().Print("SnapshotDelete task complete", field.M{"VolumeID": volID, "SnapshotID": snapshotID})
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
			// TBD retrieve labels from tags
			kvs, err := p.Gom.RetrieveMetadata(ctx, vimID(volID), &snapID, "")
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get snapshot metadata")
			}
			log.Debug().Print("RetrieveMetadata done: " + volID + "," + snapshotID)
			snapshot.Tags = convertKeyValueToTags(kvs)
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
		return nil
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
	_, err = task.Wait(ctx, defaultWaitTime)
	if err != nil {
		return errors.Wrap(err, "Failed to wait on task")
	}
	return nil
}

// VolumesList is part of blockstorage.Provider
func (p *FcdProvider) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

// SnapshotsList is part of blockstorage.Provider
func (p *FcdProvider) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}
