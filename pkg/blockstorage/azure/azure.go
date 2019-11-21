package azure

import (
	"context"
	"fmt"
	"regexp"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	azto "github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"

	"github.com/pkg/errors"
)

var _ blockstorage.Provider = (*adStorage)(nil)

const volumeNameFmt = "vol-%s"
const snapshotNameFmt = "snap-%s"

type adStorage struct {
	azCli *Client
}

func (s *adStorage) Type() blockstorage.Type {
	return blockstorage.TypeAD
}

// NewProvider returns a provider for the Azure storage type
func NewProvider(config map[string]string) (blockstorage.Provider, error) {
	azCli, err := NewClient(context.Background())
	if err != nil {
		return nil, err
	}
	return &adStorage{azCli: azCli}, nil
}

func (s *adStorage) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	_, rg, name, err := parseDiskID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get info for volume %v", id)
	}
	disk, err := s.azCli.DisksClient.Get(ctx, rg, name)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get info for volume %v", id)
	}
	mv := s.VolumeParse(ctx, disk)
	return mv, nil
}

func (s *adStorage) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	tags := blockstorage.SanitizeTags(blockstorage.KeyValueToMap(volume.Tags))
	diskName := fmt.Sprintf(volumeNameFmt, uuid.NewV1().String())
	diskProperties := &azcompute.DiskProperties{
		DiskSizeGB: azto.Int32Ptr(int32(volume.Size)),
		CreationData: &azcompute.CreationData{
			CreateOption: azcompute.DiskCreateOption(azcompute.DiskCreateOptionTypesEmpty),
		},
	}
	// TODO(ilya): figure out how to create SKUed disks
	createDisk := azcompute.Disk{
		Name:           azto.StringPtr(diskName),
		Tags:           *azto.StringMapPtr(tags),
		Location:       azto.StringPtr(volume.Az),
		DiskProperties: diskProperties,
	}
	result, err := s.azCli.DisksClient.CreateOrUpdate(ctx, s.azCli.ResourceGroup, diskName, createDisk)
	if err != nil {
		return nil, err
	}
	err = result.WaitForCompletionRef(ctx, s.azCli.DisksClient.Client)
	if err != nil {
		return nil, err
	}
	disk, err := result.Result(*s.azCli.DisksClient)
	if err != nil {
		return nil, err
	}

	// Even though the 'CreateOrUpdate' call above returns a 'Disk' model, this is incomplete and
	// requires a GET to populate correctly.
	// See https://github.com/Azure/azure-sdk-for-go/issues/326 for the explanation why
	return s.VolumeGet(ctx, azto.String(disk.ID), volume.Az)
}

func (s *adStorage) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	_, rg, name, err := parseDiskID(volume.ID)
	if err != nil {
		return errors.Wrapf(err, "After parseDiskID, failed to delete volume %v", volume.ID)
	}
	result, err := s.azCli.DisksClient.Delete(ctx, rg, name)
	if err != nil {
		return errors.Wrapf(err, "Failed to delete volume %v", volume.ID)
	}
	err = result.WaitForCompletionRef(ctx, s.azCli.DisksClient.Client)
	if err != nil {
		return errors.Wrapf(err, "After WaitForCompletion, failed to delete volume %v", volume.ID)
	}
	return nil
}

func (s *adStorage) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.Wrap(nil, "SnapshotCopy not implemented")
}

func (s *adStorage) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	snapName := fmt.Sprintf(snapshotNameFmt, uuid.NewV1().String())
	tags = blockstorage.SanitizeTags(ktags.GetTags(tags))
	createSnap := azcompute.Snapshot{
		Name:     azto.StringPtr(snapName),
		Location: azto.StringPtr(volume.Az),
		Tags:     *azto.StringMapPtr(tags),
		DiskProperties: &azcompute.DiskProperties{
			CreationData: &azcompute.CreationData{
				CreateOption:     azcompute.Copy,
				SourceResourceID: azto.StringPtr(volume.ID),
			},
		},
	}
	result, err := s.azCli.SnapshotsClient.CreateOrUpdate(ctx, s.azCli.ResourceGroup, snapName, createSnap)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create snapshot for volume %v", volume)
	}
	err = result.WaitForCompletionRef(ctx, s.azCli.SnapshotsClient.Client)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create snapshot for volume %v", volume)
	}
	rs, err := result.Result(*s.azCli.SnapshotsClient)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create snapshot for volume %v", volume)
	}

	snap, err := s.SnapshotGet(ctx, azto.String(rs.ID))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create snapshot for volume %v", volume)
	}
	snap.Volume = &volume
	return snap, nil
}

func (s *adStorage) SnapshotCreateWaitForCompletion(ctx context.Context, snap *blockstorage.Snapshot) error {
	log.Error().Print("SnapshotCreateWaitForCompletion not implemented")
	return nil
}

const (
	snapshotIDRegEx = `/subscriptions/(.*)/resourceGroups/(.*)/providers/Microsoft.Compute/snapshots/(.*)`
	diskIDRegEx     = `/subscriptions/(.*)/resourceGroups/(.*)/providers/Microsoft.Compute/disks/(.*)`
)

func parseDiskID(id string) (subscription string, resourceGroup string, name string, err error) {
	comps := regexp.MustCompile(diskIDRegEx).FindStringSubmatch(id)
	if len(comps) != 4 {
		return "", "", "", errors.New("Failed to parse Disk ID" + id)
	}
	return comps[1], comps[2], comps[3], nil
}

func parseSnapshotID(id string) (subscription string, resourceGroup string, name string, err error) {
	comps := regexp.MustCompile(snapshotIDRegEx).FindStringSubmatch(id)
	if len(comps) != 4 {
		return "", "", "", errors.New("Failed to parse Snapshot ID" + id)
	}
	return comps[1], comps[2], comps[3], nil
}

func (s *adStorage) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	_, rg, name, err := parseSnapshotID(snapshot.ID)
	if err != nil {
		return errors.Wrapf(err, "Failed to delete snapshot for snapshot ID %v", snapshot.ID)
	}
	result, err := s.azCli.SnapshotsClient.Delete(ctx, rg, name)
	if err != nil {
		return errors.Wrapf(err, "Failed to delete snapshot for snapshot ID %v", snapshot.ID)
	}
	err = result.WaitForCompletionRef(ctx, s.azCli.SnapshotsClient.Client)
	if err != nil {
		return errors.Wrapf(err, "Failed to delete snapshot for snapshot ID %v", snapshot.ID)
	}
	return nil
}

func (s *adStorage) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	_, rg, name, err := parseSnapshotID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get snapshot for snapshot ID %v", id)
	}
	snap, err := s.azCli.SnapshotsClient.Get(ctx, rg, name)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get snapshot for snapshot ID %v", id)
	}

	return s.snapshotParse(ctx, snap), nil
}

func (s *adStorage) VolumeParse(ctx context.Context, volume interface{}) *blockstorage.Volume {
	vol := volume.(azcompute.Disk)
	encrypted := false
	if vol.DiskProperties.EncryptionSettings != nil {
		encrypted = true
	}
	tags := map[string]string{"": ""}
	if vol.Tags != nil {
		tags = azto.StringMap(vol.Tags)
	}

	return &blockstorage.Volume{
		Type:       s.Type(),
		ID:         azto.String(vol.ID),
		Encrypted:  encrypted,
		Size:       int64(azto.Int32(vol.DiskSizeGB)),
		Az:         azto.String(vol.Location),
		Tags:       blockstorage.MapToKeyValue(tags),
		VolumeType: azto.String(vol.Sku.Tier),
		// Users:        []string{azto.String(vol.ManagedBy)},
		CreationTime: blockstorage.TimeStamp(vol.DiskProperties.TimeCreated.ToTime()),
	}
}

func (s *adStorage) SnapshotParse(ctx context.Context, snapshot interface{}) (*blockstorage.Snapshot, error) {
	if snap, ok := snapshot.(azcompute.Snapshot); ok {
		return s.snapshotParse(ctx, snap), nil
	}
	return nil, errors.Wrapf(nil, "snapshot is not of type *azcompute.Snapshot: %v", snapshot)
}

func (s *adStorage) snapshotParse(ctx context.Context, snap azcompute.Snapshot) *blockstorage.Snapshot {
	vol := &blockstorage.Volume{
		Type: s.Type(),
		ID:   azto.String(snap.DiskProperties.CreationData.SourceResourceID),
	}

	snapCreationTime := *snap.TimeCreated
	encrypted := false
	if snap.DiskProperties.EncryptionSettings != nil {
		encrypted = true
	}
	tags := map[string]string{"": ""}
	if snap.Tags != nil {
		tags = azto.StringMap(snap.Tags)
	}

	return &blockstorage.Snapshot{
		Encrypted:    encrypted,
		ID:           azto.String(snap.ID),
		Region:       azto.String(snap.Location),
		Size:         int64(azto.Int32(snap.DiskProperties.DiskSizeGB)),
		Tags:         blockstorage.MapToKeyValue(tags),
		Type:         s.Type(),
		Volume:       vol,
		CreationTime: blockstorage.TimeStamp(snapCreationTime.ToTime()),
	}
}

func (s *adStorage) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	// (ilya): It looks like azure doesn't support search by tags
	// List does listing per Subscription
	for diskList, err := s.azCli.DisksClient.ListComplete(ctx); diskList.NotDone(); err = diskList.Next() {
		if err != nil {
			return nil, errors.Wrapf(err, "DisksClient.List in VolumesList")
		}
		disk := diskList.Value()
		vol := s.VolumeParse(ctx, disk)
		vols = append(vols, vol)
	}
	return vols, nil
}

func (s *adStorage) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	// (ilya): It looks like azure doesn't support search by tags
	// List does listing per Subscription
	for snapList, err := s.azCli.SnapshotsClient.ListComplete(ctx); snapList.NotDone(); err = snapList.Next() {
		if err != nil {
			return nil, errors.Wrapf(err, "SnapshotsClient.List in SnapshotsList")
		}
		snap := snapList.Value()
		k10Snap, err := s.SnapshotParse(ctx, snap)
		if err != nil {
			log.WithError(err).Print("Incorrect Snaphost type", field.M{"SnapshotID": snap.ID})
			continue
		}
		snaps = append(snaps, k10Snap)
	}
	return snaps, nil
}

func (s *adStorage) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	// Incorporate pre-existing tags if overrides don't already exist
	// in provided tags
	for _, tag := range snapshot.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}

	diskName := fmt.Sprintf(volumeNameFmt, uuid.NewV1().String())
	tags = blockstorage.SanitizeTags(tags)
	createDisk := azcompute.Disk{
		Name:     azto.StringPtr(diskName),
		Tags:     *azto.StringMapPtr(tags),
		Location: azto.StringPtr(snapshot.Region),
		DiskProperties: &azcompute.DiskProperties{
			CreationData: &azcompute.CreationData{
				CreateOption:     azcompute.Copy,
				SourceResourceID: azto.StringPtr(snapshot.ID),
			},
		},
	}
	result, err := s.azCli.DisksClient.CreateOrUpdate(ctx, s.azCli.ResourceGroup, diskName, createDisk)
	if err != nil {
		return nil, errors.Wrapf(err, "DiskCLient CreateOrUpdate failed: diskName=%v, snapshotID=%v", diskName, snapshot.ID)
	}
	err = result.WaitForCompletionRef(ctx, s.azCli.DisksClient.Client)
	if err != nil {
		return nil, errors.Wrapf(err, "DiskCLient CreateOrUpdate failed: diskName=%v, snapshotID=%v", diskName, snapshot.ID)
	}
	disk, err := result.Result(*s.azCli.DisksClient)
	if err != nil {
		return nil, err
	}
	return s.VolumeGet(ctx, azto.String(disk.ID), snapshot.Volume.Az)
}

func (s *adStorage) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch res := resource.(type) {
	case *blockstorage.Snapshot:
		{
			_, rg, name, err := parseSnapshotID(res.ID)
			if err != nil {
				return err
			}
			snap, err := s.azCli.SnapshotsClient.Get(ctx, rg, name)
			if err != nil {
				return errors.Wrapf(err, "SnapshotsClient.Get failed: snapshotID=%v", res.ID)
			}
			tags = ktags.AddMissingTags(azto.StringMap(snap.Tags), ktags.GetTags(tags))
			snapProperties := azcompute.SnapshotUpdate{
				Tags: *azto.StringMapPtr(blockstorage.SanitizeTags(tags)),
			}
			result, err := s.azCli.SnapshotsClient.Update(ctx, rg, name, snapProperties)
			if err != nil {
				return errors.Wrapf(err, "SnapshotsClient.Update failed: snapshotID=%v", name)
			}
			err = result.WaitForCompletionRef(ctx, s.azCli.SnapshotsClient.Client)
			if err != nil {
				return errors.Wrapf(err, "SnapshotsClient.Update failed: snapshotID=%v", name)
			}

			return nil
		}
	case *blockstorage.Volume:
		{
			_, rg, volID, err := parseDiskID(res.ID)
			if err != nil {
				return err
			}
			vol, err := s.azCli.DisksClient.Get(ctx, rg, volID)
			if err != nil {
				return errors.Wrapf(err, "DiskClient.Get failed: volumeID=%v", volID)
			}
			tags = ktags.AddMissingTags(azto.StringMap(vol.Tags), ktags.GetTags(tags))

			diskProperties := azcompute.DiskUpdate{
				Tags: *azto.StringMapPtr(blockstorage.SanitizeTags(tags)),
			}
			result, err := s.azCli.DisksClient.Update(ctx, rg, volID, diskProperties)
			if err != nil {
				return errors.Wrapf(err, "DiskClient.Update failed: volumeID=%v", volID)
			}
			err = result.WaitForCompletionRef(ctx, s.azCli.DisksClient.Client)
			if err != nil {
				return errors.Wrapf(err, "DiskClient.Update failed: volumeID=%v", volID)
			}
			return nil
		}
	default:
		return errors.Wrapf(nil, "Unknown resource type %v", res)
	}

}
