package vmware

import (
	"context"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/vmware/govmomi/cns"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vslm"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
)

var _ blockstorage.Provider = (*fcdProvider)(nil)

const (
	noDescription   = ""
	defaultWaitTime = 10 * time.Minute
)

type fcdProvider struct {
	gom *vslm.GlobalObjectManager
	cns *cns.Client
}

// NewProvider creates new VMWare FCD provider with the config.
// URL taken from config helps to establish connection.
func NewProvider(config map[string]string) (blockstorage.Provider, error) {
	u, err := soap.ParseURL(config["url"])
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get config")
	}
	soapCli := soap.NewClient(u, true)
	ctx := context.Background()
	cli, err := vim25.NewClient(ctx, soapCli)
	req := types.Login{
		This: *cli.ServiceContent.SessionManager,
	}
	req.UserName = u.User.Username()
	req.Password, _ = u.User.Password()
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
	return &fcdProvider{
		cns: cnsCli,
		gom: gom,
	}, nil
}

func (p *fcdProvider) Type() blockstorage.Type {
	return blockstorage.TypeFCD
}

func (p *fcdProvider) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (p *fcdProvider) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	volID, snapshotID, err := splitSnapshotFullID(snapshot.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to split snapshot full ID")
	}
	task, err := p.gom.CreateDiskFromSnapshot(ctx, vimID(volID), vimID(snapshotID), uuid.NewV1().String(), nil, nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create disk from snapshot")
	}
	res, err := task.Wait(ctx, defaultWaitTime)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to wait on task")
	}
	obj, ok := res.(types.VStorageObject)
	if !ok {
		return nil, errors.New("Wrong type returned")
	}
	vol, err := p.VolumeGet(ctx, obj.Config.Id.Id, "")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get volume")
	}
	tagsCNS := make(map[string]string)
	tagsCNS["cns.tag"] = "1"
	tags = ktags.Union(tags, tagsCNS)
	err = p.SetTags(ctx, vol, tags)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to set tags")
	}
	return p.VolumeGet(ctx, vol.ID, "")
}

func (p *fcdProvider) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	return errors.New("Not implemented")
}

func (p *fcdProvider) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	obj, err := p.gom.Retrieve(ctx, vimID(id))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query the disk")
	}
	kvs, err := p.gom.RetrieveMetadata(ctx, vimID(id), nil, "")
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

func (p *fcdProvider) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	task, err := p.gom.CreateSnapshot(ctx, vimID(volume.ID), noDescription)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create snapshot")
	}
	res, err := task.Wait(ctx, defaultWaitTime)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to wait on task")
	}
	id, ok := res.(types.ID)
	if !ok {
		return nil, errors.New("Unexpected type")
	}
	return p.SnapshotGet(ctx, snapshotFullID(volume.ID, id.Id))
}

func (p *fcdProvider) SnapshotCreateWaitForCompletion(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	return nil
}

func (p *fcdProvider) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	return errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	volID, snapshotID, err := splitSnapshotFullID(id)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot infer volume ID from full snapshot ID")
	}
	results, err := p.gom.RetrieveSnapshotInfo(ctx, vimID(volID))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get snapshot info")
	}
	for _, result := range results {
		if result.Id.Id == snapshotID {
			snapshot, err := convertFromObjectToSnapshot(&result, volID)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to convert object to snapshot")
			}
			snapID := vimID(snapshotID)
			kvs, err := p.gom.RetrieveMetadata(ctx, vimID(volID), &snapID, "")
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get snapshot metadata")
			}
			snapshot.Tags = convertKeyValueToTags(kvs)
			return snapshot, nil
		}
	}
	return nil, errors.New("Failed to find snapshot")
}

func (p *fcdProvider) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch r := resource.(type) {
	case *blockstorage.Volume:
		return p.setTagsVolume(ctx, r, tags)
	case *blockstorage.Snapshot:
		return nil
	default:
		return errors.New("Unsupported type for resource")
	}
}

func (p *fcdProvider) setTagsVolume(ctx context.Context, volume *blockstorage.Volume, tags map[string]string) error {
	task, err := p.gom.UpdateMetadata(ctx, vimID(volume.ID), convertTagsToKeyValue(tags), nil)
	if err != nil {
		return errors.Wrap(err, "Failed to update metadata")
	}
	_, err = task.Wait(ctx, defaultWaitTime)
	if err != nil {
		return errors.Wrap(err, "Failed to wait on task")
	}
	return nil
}

func (p *fcdProvider) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}
