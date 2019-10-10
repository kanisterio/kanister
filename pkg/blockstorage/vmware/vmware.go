package vmware

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi/cns"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vslm"

	"github.com/kanisterio/kanister/pkg/blockstorage"
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
	return nil, errors.New("Not implemented")
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
	vol := convertFromObjectToVolume(obj)
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
	return errors.New("Not implemented")
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
			snapshot := convertFromObjectToSnapshot(&result, volID)
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
	return errors.New("Not implemented")
}

func (p *fcdProvider) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}
