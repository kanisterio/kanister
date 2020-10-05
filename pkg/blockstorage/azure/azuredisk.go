package azure

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/storage"
	azto "github.com/Azure/go-autorest/autorest/to"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	"github.com/kanisterio/kanister/pkg/blockstorage/zone"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

var _ blockstorage.Provider = (*AdStorage)(nil)

var _ zone.Mapper = (*AdStorage)(nil)

const (
	volumeNameFmt     = "vol-%s"
	snapshotNameFmt   = "snap-%s"
	copyContainerName = "vhdscontainer"
	copyBlobName      = "copy-blob-%s.vhd"
)

type AdStorage struct {
	azCli *Client
}

func (s *AdStorage) Type() blockstorage.Type {
	return blockstorage.TypeAD
}

// NewProvider returns a provider for the Azure blockstorage type
func NewProvider(ctx context.Context, config map[string]string) (blockstorage.Provider, error) {
	azCli, err := NewClient(ctx, config)
	if err != nil {
		return nil, err
	}
	return &AdStorage{azCli: azCli}, nil
}

func (s *AdStorage) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	_, rg, name, err := parseDiskID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get info for volume with ID %s", id)
	}
	disk, err := s.azCli.DisksClient.Get(ctx, rg, name)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get volume, volumeID: %s", id)
	}
	return s.VolumeParse(ctx, disk)
}

func (s *AdStorage) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	tags := blockstorage.SanitizeTags(blockstorage.KeyValueToMap(volume.Tags))
	diskName := fmt.Sprintf(volumeNameFmt, uuid.NewV1().String())
	diskProperties := &azcompute.DiskProperties{
		DiskSizeGB: azto.Int32Ptr(int32(volume.Size)),
		CreationData: &azcompute.CreationData{
			CreateOption: azcompute.DiskCreateOption(azcompute.DiskCreateOptionTypesEmpty),
		},
	}
	region, id, err := getLocationInfo(volume.Az)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get region from zone %s", volume.Az)
	}
	// TODO(ilya): figure out how to create SKUed disks
	createDisk := azcompute.Disk{
		Name:           azto.StringPtr(diskName),
		Tags:           *azto.StringMapPtr(tags),
		Location:       azto.StringPtr(region),
		DiskProperties: diskProperties,
	}
	if id != "" {
		createDisk.Zones = azto.StringSlicePtr([]string{id})
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

func (s *AdStorage) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	_, rg, name, err := parseDiskID(volume.ID)
	if err != nil {
		return errors.Wrapf(err, "Error in deleting Volume with ID %s", volume.ID)
	}
	result, err := s.azCli.DisksClient.Delete(ctx, rg, name)
	if err != nil {
		return errors.Wrapf(err, "Error in deleting Volume with ID %s", volume.ID)
	}
	err = result.WaitForCompletionRef(ctx, s.azCli.DisksClient.Client)
	return errors.Wrapf(err, "Error in deleting Volume with ID %s", volume.ID)
}

func (s *AdStorage) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Copy Snapshot not implemented")
}

// SnapshotCopyWithArgs func: args map should contain non-empty StorageAccountName(AZURE_MIGRATE_STORAGE_ACCOUNT_NAME)
// and StorageKey(AZURE_MIGRATE_STORAGE_ACCOUNT_KEY)
func (s *AdStorage) SnapshotCopyWithArgs(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot, args map[string]string) (*blockstorage.Snapshot, error) {
	migrateStorageAccount := args[blockstorage.AzureMigrateStorageAccount]
	migrateStorageKey := args[blockstorage.AzureMigrateStorageKey]
	if migrateStorageAccount == "" || migrateStorageKey == "" {
		return nil, errors.Errorf("Required args %s and %s  for snapshot copy not available", blockstorage.AzureMigrateStorageAccount, blockstorage.AzureMigrateStorageKey)
	}

	storageCli, err := storage.NewBasicClient(migrateStorageAccount, migrateStorageKey)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get storage service client")
	}
	storageAccountID := "/subscriptions/" + s.azCli.SubscriptionID + "/resourceGroups/" + s.azCli.ResourceGroup + "/providers/Microsoft.Storage/storageAccounts/" + migrateStorageAccount

	_, rg, name, err := parseSnapshotID(from.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "SnapshotsClient.Copy: Failure in parsing snapshot ID %s", from.ID)
	}
	_, err = s.azCli.SnapshotsClient.Get(ctx, rg, name)
	if err != nil {
		return nil, errors.Wrapf(err, "SnapshotsClient.Copy: Failed to get snapshot with ID %s", from.ID)
	}

	duration := int32(3600)
	gad := azcompute.GrantAccessData{
		Access:            azcompute.Read,
		DurationInSeconds: &duration,
	}

	snapshotsGrantAccessFuture, err := s.azCli.SnapshotsClient.GrantAccess(ctx, rg, name, gad)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to grant read access to snapshot: %s", from.ID)
	}
	defer s.revokeAccess(ctx, rg, name, from.ID)

	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		_, err := snapshotsGrantAccessFuture.Result(*s.azCli.SnapshotsClient)
		if err != nil {
			if strings.Contains(err.Error(), "asynchronous operation has not completed") {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "SnapshotsClient.Copy failure to grant snapshot access")
	}

	accessURI, err := snapshotsGrantAccessFuture.Result(*s.azCli.SnapshotsClient)
	if err != nil {
		return nil, errors.Wrap(err, "SnapshotsClient.Copy failure to grant snapshot access")
	}
	blobStorageClient := storageCli.GetBlobService()
	container := blobStorageClient.GetContainerReference(copyContainerName)
	_, err = container.CreateIfNotExists(nil)
	if err != nil {
		return nil, err
	}
	blobName := fmt.Sprintf(copyBlobName, name)
	blob := container.GetBlobReference(blobName)
	defer deleteBlob(blob, blobName)

	var copyOptions *storage.CopyOptions
	if t, ok := ctx.Deadline(); ok {
		time := time.Until(t).Seconds()
		if time <= 0 {
			return nil, errors.New("Context deadline exceeded, cannot copy snapshot")
		}
		copyOptions = &storage.CopyOptions{
			Timeout: uint(time),
		}
	}
	err = blob.Copy(*accessURI.AccessSAS, copyOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to copy disk to blob")
	}
	blobURI := blob.GetURL()

	snapName := fmt.Sprintf(snapshotNameFmt, uuid.NewV1().String())
	var tags = make(map[string]string)
	for _, tag := range from.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}
	tags = blockstorage.SanitizeTags(ktags.GetTags(tags))

	createSnap := azcompute.Snapshot{
		Name:     azto.StringPtr(snapName),
		Location: azto.StringPtr(to.Region),
		Tags:     *azto.StringMapPtr(tags),
		SnapshotProperties: &azcompute.SnapshotProperties{
			CreationData: &azcompute.CreationData{
				CreateOption:     azcompute.Import,
				StorageAccountID: azto.StringPtr(storageAccountID),
				SourceURI:        azto.StringPtr(blobURI),
			},
		},
	}

	result, err := s.azCli.SnapshotsClient.CreateOrUpdate(ctx, s.azCli.ResourceGroup, snapName, createSnap)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to copy snapshot from source snapshot %v", from)
	}
	err = result.WaitForCompletionRef(ctx, s.azCli.SnapshotsClient.Client)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to copy snapshot from source snapshot %v", from)
	}
	rs, err := result.Result(*s.azCli.SnapshotsClient)
	if err != nil {
		return nil, errors.Wrapf(err, "Error in getting result of Snapshot copy operation, snaphotName %s", snapName)
	}

	snap, err := s.SnapshotGet(ctx, azto.String(rs.ID))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to Get Snapshot after create, snaphotName %s", snapName)
	}
	*snap.Volume = *from.Volume
	return snap, nil
}

func (s *AdStorage) revokeAccess(ctx context.Context, rg, name, ID string) {
	_, err := s.azCli.SnapshotsClient.RevokeAccess(ctx, rg, name)
	if err != nil {
		log.Print("Failed to revoke access from snapshot", field.M{"snapshot": ID})
	}
}

func deleteBlob(blob *storage.Blob, blobName string) {
	_, err := blob.DeleteIfExists(nil)
	if err != nil {
		log.Print("Failed to delete blob", field.M{"blob": blobName})
	}
}

func (s *AdStorage) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	snapName := fmt.Sprintf(snapshotNameFmt, uuid.NewV1().String())
	tags = blockstorage.SanitizeTags(ktags.GetTags(tags))
	region, _, err := getLocationInfo(volume.Az)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get region from zone %s", volume.Az)
	}
	createSnap := azcompute.Snapshot{
		Name:     azto.StringPtr(snapName),
		Location: azto.StringPtr(region),
		Tags:     *azto.StringMapPtr(tags),
		SnapshotProperties: &azcompute.SnapshotProperties{
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
		return nil, errors.Wrapf(err, "Error in getting result of Snapshot create operation, snaphotName %s", snapName)
	}

	snap, err := s.SnapshotGet(ctx, azto.String(rs.ID))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to Get Snapshot after create, snaphotName %s", snapName)
	}
	snap.Volume = &volume
	return snap, nil
}

func (s *AdStorage) SnapshotCreateWaitForCompletion(ctx context.Context, snap *blockstorage.Snapshot) error {
	//return errors.New("SnapshotCreateWaitForCompletion not implemented")
	return nil
}

const (
	snapshotIDRegEx = `/subscriptions/(.*)/resourceGroups/(.*)/providers/Microsoft.Compute/snapshots/(.*)`
	diskIDRegEx     = `/subscriptions/(.*)/resourceGroups/(.*)/providers/Microsoft.Compute/disks/(.*)`
)

var diskIDRe = regexp.MustCompile(diskIDRegEx)
var snapIDRe = regexp.MustCompile(snapshotIDRegEx)

// nolint:unparam
func parseDiskID(id string) (subscription string, resourceGroup string, name string, err error) {
	comps := diskIDRe.FindStringSubmatch(id)
	if len(comps) != 4 {
		return "", "", "", errors.New("Failed to parse Disk ID" + id)
	}
	return comps[1], comps[2], comps[3], nil
}

// nolint:unparam
func parseSnapshotID(id string) (subscription string, resourceGroup string, name string, err error) {
	comps := snapIDRe.FindStringSubmatch(id)
	if len(comps) != 4 {
		return "", "", "", errors.New("Failed to parse Snapshot ID" + id)
	}
	return comps[1], comps[2], comps[3], nil
}

func (s *AdStorage) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	_, rg, name, err := parseSnapshotID(snapshot.ID)
	if err != nil {
		return errors.Wrapf(err, "SnapshotClient.Delete: Failure in parsing snapshot ID %s", snapshot.ID)
	}
	result, err := s.azCli.SnapshotsClient.Delete(ctx, rg, name)
	if err != nil {
		return errors.Wrapf(err, "SnapshotClient.Delete: Failed to delete snapshot with ID %s", snapshot.ID)
	}
	err = result.WaitForCompletionRef(ctx, s.azCli.SnapshotsClient.Client)

	return errors.Wrapf(err, "SnapshotClient.Delete: Error while waiting for snapshot with ID %s to get deleted", snapshot.ID)
}

func (s *AdStorage) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	_, rg, name, err := parseSnapshotID(id)
	if err != nil {
		return nil, errors.Wrapf(err, "SnapshotsClient.Get: Failure in parsing snapshot ID %s", id)
	}
	snap, err := s.azCli.SnapshotsClient.Get(ctx, rg, name)
	if err != nil {
		return nil, errors.Wrapf(err, "SnapshotsClient.Get: Failed to get snapshot with ID %s", id)
	}

	return s.snapshotParse(ctx, snap), nil
}

func (s *AdStorage) VolumeParse(ctx context.Context, volume interface{}) (*blockstorage.Volume, error) {
	vol, ok := volume.(azcompute.Disk)
	if !ok {
		return nil, errors.New(fmt.Sprintf("Volume is not of type *azcompute.Disk, volume: %v", volume))
	}
	encrypted := false
	if vol.DiskProperties.EncryptionSettingsCollection != nil &&
		vol.DiskProperties.EncryptionSettingsCollection.Enabled != nil {
		encrypted = *vol.DiskProperties.EncryptionSettingsCollection.Enabled
	}
	tags := map[string]string{"": ""}
	if vol.Tags != nil {
		tags = azto.StringMap(vol.Tags)
	}
	az := azto.String(vol.Location)
	if z := azto.StringSlice(vol.Zones); len(z) > 0 {
		az = az + "-" + z[0]
	}

	return &blockstorage.Volume{
		Type:         s.Type(),
		ID:           azto.String(vol.ID),
		Encrypted:    encrypted,
		Size:         int64(azto.Int32(vol.DiskSizeGB)),
		Az:           az,
		Tags:         blockstorage.MapToKeyValue(tags),
		VolumeType:   string(vol.Sku.Name),
		CreationTime: blockstorage.TimeStamp(vol.DiskProperties.TimeCreated.ToTime()),
		Attributes:   map[string]string{"Users": azto.String(vol.ManagedBy)},
	}, nil
}

func (s *AdStorage) SnapshotParse(ctx context.Context, snapshot interface{}) (*blockstorage.Snapshot, error) {
	if snap, ok := snapshot.(azcompute.Snapshot); ok {
		return s.snapshotParse(ctx, snap), nil
	}
	return nil, errors.New(fmt.Sprintf("Snapshot is not of type *azcompute.Snapshot, snapshot: %v", snapshot))
}

func (s *AdStorage) snapshotParse(ctx context.Context, snap azcompute.Snapshot) *blockstorage.Snapshot {
	vol := &blockstorage.Volume{
		Type: s.Type(),
		ID:   azto.String(snap.SnapshotProperties.CreationData.SourceResourceID),
	}

	snapCreationTime := *snap.TimeCreated
	encrypted := false
	if snap.SnapshotProperties.EncryptionSettingsCollection != nil &&
		snap.SnapshotProperties.EncryptionSettingsCollection.Enabled != nil {
		encrypted = *snap.SnapshotProperties.EncryptionSettingsCollection.Enabled
	}
	tags := map[string]string{"": ""}
	if snap.Tags != nil {
		tags = azto.StringMap(snap.Tags)
	}
	return &blockstorage.Snapshot{
		Encrypted:    encrypted,
		ID:           azto.String(snap.ID),
		Region:       azto.String(snap.Location),
		Size:         int64(azto.Int32(snap.SnapshotProperties.DiskSizeGB)),
		Tags:         blockstorage.MapToKeyValue(tags),
		Type:         s.Type(),
		Volume:       vol,
		CreationTime: blockstorage.TimeStamp(snapCreationTime.ToTime()),
	}
}

func (s *AdStorage) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	// (ilya): It looks like azure doesn't support search by tags
	// List does listing per Subscription
	for diskList, err := s.azCli.DisksClient.ListComplete(ctx); diskList.NotDone(); err = diskList.Next() {
		if err != nil {
			return nil, errors.Wrap(err, "DisksClient.List in VolumesList")
		}
		disk := diskList.Value()
		vol, err := s.VolumeParse(ctx, disk)
		if err != nil {
			return nil, errors.Wrap(err, "DisksClient.List in VolumesList, failure in parsing Volume")
		}
		vols = append(vols, vol)
	}
	return vols, nil
}

func (s *AdStorage) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	// (ilya): It looks like azure doesn't support search by tags
	// List does listing per Subscription
	for snapList, err := s.azCli.SnapshotsClient.ListComplete(ctx); snapList.NotDone(); err = snapList.Next() {
		if err != nil {
			return nil, errors.Wrap(err, "SnapshotsClient.List in SnapshotsList")
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

func (s *AdStorage) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	// Incorporate pre-existing tags if overrides don't already exist
	// in provided tags
	for _, tag := range snapshot.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}

	region, id, err := s.getRegionAndZoneID(ctx, snapshot.Region, snapshot.Volume.Az)
	if err != nil {
		return nil, err
	}

	diskName := fmt.Sprintf(volumeNameFmt, uuid.NewV1().String())
	tags = blockstorage.SanitizeTags(tags)
	createDisk := azcompute.Disk{
		Name:     azto.StringPtr(diskName),
		Tags:     *azto.StringMapPtr(tags),
		Location: azto.StringPtr(region),
		DiskProperties: &azcompute.DiskProperties{
			CreationData: &azcompute.CreationData{
				CreateOption:     azcompute.Copy,
				SourceResourceID: azto.StringPtr(snapshot.ID),
			},
		},
	}
	if id != "" {
		createDisk.Zones = azto.StringSlicePtr([]string{id})
	}
	for _, saType := range azcompute.PossibleDiskStorageAccountTypesValues() {
		if string(saType) == snapshot.Volume.VolumeType {
			createDisk.Sku = &azcompute.DiskSku{
				Name: saType,
			}
		}
	}
	result, err := s.azCli.DisksClient.CreateOrUpdate(ctx, s.azCli.ResourceGroup, diskName, createDisk)
	if err != nil {
		return nil, errors.Wrapf(err, "DiskCLient.CreateOrUpdate in VolumeCreateFromSnapshot, diskName: %s, snapshotID: %s", diskName, snapshot.ID)
	}
	if err = result.WaitForCompletionRef(ctx, s.azCli.DisksClient.Client); err != nil {
		return nil, errors.Wrapf(err, "DiskCLient.CreateOrUpdate in VolumeCreateFromSnapshot, diskName: %s, snapshotID: %s", diskName, snapshot.ID)
	}
	disk, err := result.Result(*s.azCli.DisksClient)
	if err != nil {
		return nil, err
	}
	return s.VolumeGet(ctx, azto.String(disk.ID), snapshot.Volume.Az)
}

func (s *AdStorage) getRegionAndZoneID(ctx context.Context, sourceRegion, volAz string) (string, string, error) {
	//check if current node region is zoned or not
	kubeCli, err := kube.NewClient()
	if err != nil {
		return "", "", err
	}
	zs, region, err := zone.NodeZonesAndRegion(ctx, kubeCli)
	if err != nil {
		return "", "", err
	}
	if len(zs) == 0 {
		return region, "", nil
	}

	zones, err := zone.FromSourceRegionZone(ctx, s, kubeCli, sourceRegion, volAz)
	if err != nil {
		return "", "", err
	}
	if len(zones) != 1 {
		return "", "", errors.Errorf("Length of zone slice should be 1, got %d", len(zones))
	}

	region, id, err := getLocationInfo(zones[0])
	return region, id, errors.Wrapf(err, "Could not get region from zone %s", zones[0])
}

func getLocationInfo(az string) (string, string, error) {
	if az == "" {
		return "", "", errors.New("zone value is empty")
	}

	s := strings.Split(az, "-")
	var region, zoneID string
	if len(s) == 2 {
		region = s[0]
		zoneID = s[1]
	} else {
		region = s[0]
	}
	return region, zoneID, nil
}

func (s *AdStorage) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch res := resource.(type) {
	case *blockstorage.Snapshot:
		{
			_, rg, name, err := parseSnapshotID(res.ID)
			if err != nil {
				return err
			}
			snap, err := s.azCli.SnapshotsClient.Get(ctx, rg, name)
			if err != nil {
				return errors.Wrapf(err, "SnapshotsClient.Get in SetTags, snapshotID: %s", res.ID)
			}
			tags = ktags.AddMissingTags(azto.StringMap(snap.Tags), ktags.GetTags(tags))
			snapProperties := azcompute.SnapshotUpdate{
				Tags: *azto.StringMapPtr(blockstorage.SanitizeTags(tags)),
			}
			result, err := s.azCli.SnapshotsClient.Update(ctx, rg, name, snapProperties)
			if err != nil {
				return errors.Wrapf(err, "SnapshotsClient.Update in SetTags, snapshotID: %s", name)
			}
			err = result.WaitForCompletionRef(ctx, s.azCli.SnapshotsClient.Client)
			return errors.Wrapf(err, "SnapshotsClient.Update in SetTags, snapshotID: %s", name)
		}
	case *blockstorage.Volume:
		{
			_, rg, volID, err := parseDiskID(res.ID)
			if err != nil {
				return err
			}
			vol, err := s.azCli.DisksClient.Get(ctx, rg, volID)
			if err != nil {
				return errors.Wrapf(err, "DiskClient.Get in SetTags, volumeID: %s", volID)
			}
			tags = ktags.AddMissingTags(azto.StringMap(vol.Tags), ktags.GetTags(tags))

			diskProperties := azcompute.DiskUpdate{
				Tags: *azto.StringMapPtr(blockstorage.SanitizeTags(tags)),
			}
			result, err := s.azCli.DisksClient.Update(ctx, rg, volID, diskProperties)
			if err != nil {
				return errors.Wrapf(err, "DiskClient.Update in SetTags, volumeID: %s", volID)
			}
			err = result.WaitForCompletionRef(ctx, s.azCli.DisksClient.Client)
			return errors.Wrapf(err, "DiskClient.Update in SetTags, volumeID: %s", volID)
		}
	default:
		return errors.New(fmt.Sprintf("Unknown resource type %v", res))
	}
}

func (s *AdStorage) FromRegion(ctx context.Context, region string) ([]string, error) {
	return staticRegionToZones(region)
}

func (s *AdStorage) SnapshotRestoreTargets(ctx context.Context, snapshot *blockstorage.Snapshot) (global bool, regionsAndZones map[string][]string, err error) {
	// A few checks from VolumeCreateFromSnapshot
	if snapshot.Volume == nil {
		return false, nil, errors.New("Snapshot volume information not available")
	}
	if snapshot.Volume.VolumeType == "" {
		return false, nil, errors.Errorf("Required VolumeType not set")
	}

	zl, err := staticRegionToZones(snapshot.Region)
	if err != nil {
		return false, nil, err
	}
	return false, map[string][]string{snapshot.Region: zl}, nil
}

func staticRegionToZones(region string) ([]string, error) {
	switch region {
	case "australiasoutheast", "brazilsouth", "canadaeast", "centralindia", "eastasia", "japanwest", "northcentralus", "southcentralus", "southindia", "ukwest", "westcentralus", "westindia", "westus":
		return nil, nil
	case "centralus":
		return []string{
			"centralus-1",
			"centralus-2",
			"centralus-3",
		}, nil
	case "eastus":
		return []string{
			"eastus-1",
			"eastus-2",
			"eastus-3",
		}, nil
	case "eastus2":
		return []string{
			"eastus2-1",
			"eastus2-2",
			"eastus2-3",
		}, nil
	case "japaneast":
		return []string{
			"japaneast-1",
			"japaneast-2",
			"japaneast-3",
		}, nil
	case "northeurope":
		return []string{
			"northeurope-1",
			"northeurope-2",
			"northeurope-3",
		}, nil
	case "southeastasia":
		return []string{
			"southeastasia-1",
			"southeastasia-2",
			"southeastasia-3",
		}, nil
	case "uksouth":
		return []string{
			"uksouth-1",
			"uksouth-2",
			"uksouth-3",
		}, nil
	case "westeurope":
		return []string{
			"westeurope-1",
			"westeurope-2",
			"westeurope-3",
		}, nil
	case "westus2":
		return []string{
			"westus2-1",
			"westus2-2",
			"westus2-3",
		}, nil
	case "canadacentral":
		return []string{
			"canadacentral-1",
			"canadacentral-2",
			"canadacentral-3",
		}, nil
	case "francecentral":
		return []string{
			"francecentral-1",
			"francecentral-2",
			"francecentral-3",
		}, nil
	case "australiaeast":
		return []string{
			"australiaeast-1",
			"australiaeast-2",
			"australiaeast-3",
		}, nil
	}
	return nil, errors.New(fmt.Sprintf("cannot get availability zones for region %s", region))
}
