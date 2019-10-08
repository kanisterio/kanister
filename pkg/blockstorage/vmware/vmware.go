package vmware

import (
	"context"

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
	cli, err := vim25.NewClient(context.TODO(), soapCli)
	req := types.Login{
		This: *cli.ServiceContent.SessionManager,
	}
	req.UserName = u.User.Username()
	req.Password, _ = u.User.Password()

	_, err = methods.Login(context.Background(), cli, &req)
	if err != nil {
		return nil, err
	}
	cnsCli, err := cns.NewClient(context.Background(), cli)
	if err != nil {
		return nil, err
	}
	vslmCli, err := vslm.NewClient(context.TODO(), cli)
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
	return nil, errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotCreateWaitForCompletion(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	return errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	return errors.New("Not implemented")
}

func (p *fcdProvider) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
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
