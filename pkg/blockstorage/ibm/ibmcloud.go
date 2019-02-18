package ibm

// IBM Cloud Block storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	ibmprov "github.com/IBM/ibmcloud-storage-volume-lib/lib/provider"
	"github.com/jpillora/backoff"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
)

var _ blockstorage.Provider = (*ibmCloud)(nil)

type ibmCloud struct {
	cli *client
}

const (
	maxRetries     = 10
	defaultTimeout = time.Duration(time.Minute * 5)
)

func (s *ibmCloud) Type() blockstorage.Type {
	return blockstorage.TypeSoftlayerBlock
}

// NewProvider returns a provider for the IBM Cloud
func NewProvider(ctx context.Context, args map[string]string) (blockstorage.Provider, error) {
	ibmCli, err := newClient(ctx, args)
	return &ibmCloud{cli: ibmCli}, err
}

func (s *ibmCloud) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	newVol := ibmprov.Volume{}
	newVol.VolumeType = ibmprov.VolumeType(volume.VolumeType)
	newVol.ProviderType = ibmprov.VolumeProviderType(volume.Attributes[ProviderTypeAttName])
	if volume.Iops > 0 {
		iops := strconv.Itoa(int(volume.Iops))
		newVol.Iops = &iops
	}
	if tier, ok := volume.Attributes[TierAttName]; ok {
		newVol.Tier = &tier
	}
	size := int(volume.Size)
	newVol.Capacity = &size
	newVol.VolumeNotes = blockstorage.KeyValueToMap(volume.Tags)
	newVol.Az = s.cli.SLCfg.SoftlayerDataCenter
	log.Debugf("Creating new volume %+v", newVol)
	volR, err := s.cli.Service.VolumeCreate(newVol)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to create volume. %s", err.Error()))
	}
	log.Debugf("New volume created %+v", volR)
	return s.volumeParse(ctx, volR)
}

func (s *ibmCloud) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	vol, err := s.cli.Service.VolumeGet(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get volume with id %s", id)
	}
	log.Debugf("Got volume from cloud provider %+v", vol)
	return s.volumeParse(ctx, vol)
}

func (s *ibmCloud) volumeParse(ctx context.Context, vol *ibmprov.Volume) (*blockstorage.Volume, error) {
	tags := []*blockstorage.KeyValue(nil)
	for k, v := range vol.VolumeNotes {
		tags = append(tags, &blockstorage.KeyValue{Key: k, Value: v})
	}
	var iops int64
	if vol.Iops != nil {
		iops, _ = strconv.ParseInt(*vol.Iops, 10, 64)
	}

	attribs := map[string]string{
		ProviderAttName:      string(vol.Provider),
		ProviderTypeAttName:  string(vol.ProviderType),
		SnapshotSpaceAttName: "",
		TierAttName:          "",
		BillingTypeAttName:   vol.BillingType,
		RegionAttName:        vol.Region,
	}

	if vol.Tier != nil {
		attribs[TierAttName] = *vol.Tier
	}

	if vol.SnapshotSpace != nil {
		attribs[SnapshotSpaceAttName] = strconv.Itoa(*vol.SnapshotSpace)
	}

	if vol.LunID == "" {
		return nil, errors.New("LunID is missing from Volume info")
	}
	attribs[LunIDAttName] = vol.LunID

	if len(vol.TargetIPAddresses) < 0 {
		return nil, errors.New("TargetIPAddresses are missing from Volume info")
	}
	attribs[TargetIPsAttName] = strings.Join(vol.TargetIPAddresses, ",")

	return &blockstorage.Volume{
		Type:         s.Type(),
		ID:           vol.VolumeID,
		Az:           s.cli.SLCfg.SoftlayerDataCenter, // Due to the bug in IBM lib vol.Az,
		Encrypted:    false,
		VolumeType:   string(vol.VolumeType),
		Size:         int64(*vol.Capacity),
		Tags:         tags,
		Iops:         iops,
		CreationTime: blockstorage.TimeStamp(vol.CreationTime),
		Attributes:   attribs,
	}, nil
}

func (s *ibmCloud) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	ibmVols, err := s.cli.Service.VolumesList(tags)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to list volumes with tags %v", tags)
	}

	for _, v := range ibmVols {
		pvol, err := s.volumeParse(ctx, v)
		if err != nil {
			return nil, errors.Errorf("Failed to parse vol %s", err.Error())
		}
		vols = append(vols, pvol)
	}
	return vols, nil
}

func (s *ibmCloud) snapshotParse(ctx context.Context, snap *ibmprov.Snapshot) *blockstorage.Snapshot {
	tags := []*blockstorage.KeyValue(nil)
	for k, v := range snap.SnapshotTags {
		tags = append(tags, &blockstorage.KeyValue{Key: k, Value: v})
	}

	vol := &blockstorage.Volume{
		Type: s.Type(),
		ID:   snap.VolumeID,
	}

	snapSize := int64(0)
	if snap.SnapshotSize != nil {
		snapSize = int64(*snap.SnapshotSize)
	}

	return &blockstorage.Snapshot{
		ID:           snap.SnapshotID,
		Tags:         tags,
		Type:         s.Type(),
		Encrypted:    false,
		Size:         snapSize,
		Region:       snap.Region,
		Volume:       vol,
		CreationTime: blockstorage.TimeStamp(snap.SnapshotCreationTime),
	}
}

func (s *ibmCloud) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	// IBM doens't support tag for snapshot list.
	// Getting all of them
	ibmsnaps, err := s.cli.Service.SnapshotsList()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list Snapshots")
	}

	for _, snap := range ibmsnaps {
		snaps = append(snaps, s.snapshotParse(ctx, snap))
	}
	return snaps, nil
}

// SnapshotCopy copies snapshot 'from' to 'to'. Follows aws restrictions regarding encryption;
// i.e., copying unencrypted to encrypted snapshot is allowed but not vice versa.
func (s *ibmCloud) SnapshotCopy(ctx context.Context, from, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (s *ibmCloud) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	alltags := ktags.GetTags(tags)
	ibmvol, err := s.cli.Service.VolumeGet(volume.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Volume for Snapshot creation, volume_id :%s", volume.ID)
	}

	if ibmvol.SnapshotSpace == nil {
		log.Debugf("Ordering snapshot space for volume %+v", ibmvol)
		ibmvol.SnapshotSpace = ibmvol.Capacity
		err = s.cli.Service.SnapshotOrder(*ibmvol)
		if err != nil {
			if strings.Contains(err.Error(), "already has snapshot space") != true {
				return nil, errors.Wrapf(err, "Failed to order Snapshot space, volume_id :%s", volume.ID)
			}
		}
		wctx, wcancel := context.WithTimeout(ctx, defaultTimeout)
		defer wcancel()
		err = waitforSnapSpaceOrder(wctx, s.cli, volume.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "Wait is expired for order Snapshot space, volume_id :%s", volume.ID)
		}
	}
	log.Debugf("Creating snapshot for vol %+v", ibmvol)
	snap, err := s.cli.Service.SnapshotCreate(ibmvol, alltags)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create snapshot, volume_id: %s", volume.ID)
	}
	log.Debugf("New snapshot was created %+v", snap)

	return s.snapshotParse(ctx, snap), nil
}

func (s *ibmCloud) SnapshotCreateWaitForCompletion(ctx context.Context, snap *blockstorage.Snapshot) error {
	return nil
}

func (s *ibmCloud) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	snap, err := s.cli.Service.SnapshotGet(snapshot.ID)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Snapshot for deletion, snapshot_id %s", snapshot.ID)
	}
	return s.cli.Service.SnapshotDelete(snap)
}

func (s *ibmCloud) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	snap, err := s.cli.Service.SnapshotGet(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Snapshot, snapshot_id %s", id)
	}
	return s.snapshotParse(ctx, snap), nil
}

func (s *ibmCloud) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	ibmvol, err := s.cli.Service.VolumeGet(volume.ID)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Volume for deletion, volume_id :%s", volume.ID)
	}
	return s.cli.Service.VolumeDelete(ibmvol)
}

func (s *ibmCloud) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	log.Info("IBM storage lib does not support SetTags")
	return nil
}

func (s *ibmCloud) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	if snapshot.Volume == nil {
		return nil, errors.New("Snapshot volume information not available")
	}

	// Incorporate pre-existing tags.
	for _, tag := range snapshot.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}
	snap, err := s.cli.Service.SnapshotGet(snapshot.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Snapshot for Volume Restore, snapshot_id %s", snapshot.ID)
	}
	// Temporary fix for Softlayer backend bug
	// GetSnapshot doens't return correct Volume info
	snap.VolumeID = snapshot.Volume.ID
	snap.Volume.VolumeID = snapshot.Volume.ID
	log.Debugf("Snapshot with new volume ID %+v", snap)

	vol, err := s.cli.Service.VolumeCreateFromSnapshot(*snap, tags)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Volume from Snapshot, snapshot_id %s", snapshot.ID)
	}
	return s.VolumeGet(ctx, vol.VolumeID, snapshot.Volume.Az)
}

func waitforSnapSpaceOrder(ctx context.Context, cli *client, id string) error {
	snapWaitBackoff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    1 * time.Second,
		Max:    10 * time.Second,
	}
	return poll.WaitWithBackoff(ctx, snapWaitBackoff, func(ctx context.Context) (bool, error) {
		vol, err := cli.Service.VolumeGet(id)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to get volume, volume_id: %s", id)
		}

		if vol.SnapshotSpace != nil {
			log.Debugf("Volume has snapshot space now, volume_id %s", id)
			return true, nil
		}
		log.Debugf("Still waiting for Snapshor Space order volume_id: %s", id)
		return false, nil
	})

}
