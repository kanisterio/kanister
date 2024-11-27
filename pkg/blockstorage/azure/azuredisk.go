// TODO: Switch to using the latest azure sdk and remove nolint.
// Related Ticket- https://github.com/kanisterio/kanister/issues/1684
package azure

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azto "github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/gofrs/uuid"
	"github.com/kanisterio/errkit"

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

type LocationZoneMap map[string]struct{}

// AdStorage describes the azure storage client
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
		return nil, errkit.Wrap(err, "Failed to get info for volume with ID", "id", id)
	}

	diskResponse, err := s.azCli.DisksClient.Get(ctx, rg, name, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get volume", "volumeID", id)
	}
	return s.VolumeParse(ctx, diskResponse.Disk)
}

func (s *AdStorage) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	tags := blockstorage.SanitizeTags(blockstorage.KeyValueToMap(volume.Tags))
	diskID, err := uuid.NewV1()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create UUID")
	}
	diskName := fmt.Sprintf(volumeNameFmt, diskID.String())

	diskProperties := &armcompute.DiskProperties{
		CreationData: &armcompute.CreationData{
			CreateOption: azto.Ptr(armcompute.DiskCreateOptionEmpty),
		},
		DiskSizeGB: blockstorage.Int32Ptr(int32(blockstorage.SizeInGi(volume.SizeInBytes))),
	}
	region, id, err := getLocationInfo(volume.Az)
	if err != nil {
		return nil, errkit.Wrap(err, "Could not get region from zone", "zone", volume.Az)
	}
	// TODO(ilya): figure out how to create SKUed disks
	createdDisk := armcompute.Disk{
		Name:       blockstorage.StringPtr(diskName),
		Tags:       *blockstorage.StringMapPtr(tags),
		Location:   blockstorage.StringPtr(region),
		Properties: diskProperties,
		SKU: &armcompute.DiskSKU{
			Name: azto.Ptr(armcompute.DiskStorageAccountTypesStandardLRS),
		},
	}
	if id != "" {
		createdDisk.Zones = blockstorage.SliceStringPtr([]string{id})
	}

	pollerResp, err := s.azCli.DisksClient.BeginCreateOrUpdate(ctx, s.azCli.ResourceGroup, diskName, createdDisk, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Could not create volume", "volume", diskName)
	}
	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Volume create polling error", "volume", diskName)
	}
	return s.VolumeParse(ctx, resp.Disk)
}

func (s *AdStorage) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	_, rg, name, err := parseDiskID(volume.ID)
	if err != nil {
		return errkit.Wrap(err, "Error in deleting Volume with ID", "volumeID", volume.ID)
	}
	poller, err := s.azCli.DisksClient.BeginDelete(ctx, rg, name, nil)
	if err != nil {
		return errkit.Wrap(err, "Error in deleting Volume with ID", "volumeID", volume.ID)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return errkit.Wrap(err, "Error in deleting Volume with ID", "volumeID", volume.ID)
}

func (s *AdStorage) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errkit.New("Copy Snapshot not implemented")
}

// SnapshotCopyWithArgs func: args map should contain non-empty StorageAccountName(AZURE_MIGRATE_STORAGE_ACCOUNT_NAME)
// and StorageKey(AZURE_MIGRATE_STORAGE_ACCOUNT_KEY)
// A destination ResourceGroup (AZURE_MIGRATE_RESOURCE_GROUP) can be provided. The created snapshot will belong to this.
func (s *AdStorage) SnapshotCopyWithArgs(ctx context.Context, from blockstorage.Snapshot,
	to blockstorage.Snapshot, args map[string]string) (*blockstorage.Snapshot, error) {
	migrateStorageAccount := args[blockstorage.AzureMigrateStorageAccount]
	migrateStorageKey := args[blockstorage.AzureMigrateStorageKey]
	if isMigrateStorageAccountorKey(migrateStorageAccount, migrateStorageKey) {
		return nil, errkit.New(fmt.Sprintf("Required args %s and %s for snapshot copy not available", blockstorage.AzureMigrateStorageAccount, blockstorage.AzureMigrateStorageKey))
	}

	storageCli, err := storage.NewBasicClient(migrateStorageAccount, migrateStorageKey)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot get storage service client")
	}
	storageAccountID := "/subscriptions/" + s.azCli.SubscriptionID + "/resourceGroups/" + s.azCli.ResourceGroup + "/providers/Microsoft.Storage/storageAccounts/" + migrateStorageAccount

	_, rg, name, err := parseSnapshotID(from.ID)
	if err != nil {
		return nil, errkit.Wrap(err, "SnapshotsClient.Copy: Failure in parsing snapshot ID", "snapshotID", from.ID)
	}
	_, err = s.azCli.SnapshotsClient.Get(ctx, rg, name, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "SnapshotsClient.Copy: Failed to get snapshot with ID", "snapshotID", from.ID)
	}

	duration := int32(3600)
	gad := armcompute.GrantAccessData{
		Access:            azto.Ptr(armcompute.AccessLevelRead),
		DurationInSeconds: &duration,
	}

	snapshotsGrantAccessPoller, err := s.azCli.SnapshotsClient.BeginGrantAccess(ctx, rg, name, gad, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to grant read access to", "snapshot", from.ID)
	}
	defer s.revokeAccess(ctx, rg, name, from.ID)
	snapshotGrantRes, err := snapshotsGrantAccessPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "SnapshotsClient.Copy failure to grant snapshot access. Snapshot grant access poller failed to pull the result")
	}

	if err != nil {
		return nil, errkit.Wrap(err, "SnapshotsClient.Copy failure to grant snapshot access")
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

	copyOptions, err := getCopyOptions(ctx)
	if err != nil {
		return nil, err
	}
	err = blob.Copy(*snapshotGrantRes.AccessSAS, copyOptions)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to copy disk to blob")
	}
	snapID, err := uuid.NewV1()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create UUID")
	}
	snapName := fmt.Sprintf(snapshotNameFmt, snapID.String())
	createSnap := getSnapshotObject(blob, from, to, snapName, storageAccountID)

	migrateResourceGroup := s.azCli.ResourceGroup
	if val, ok := args[blockstorage.AzureMigrateResourceGroup]; ok && val != "" {
		migrateResourceGroup = val
	}
	createSnapshotPoller, err := s.azCli.SnapshotsClient.BeginCreateOrUpdate(ctx, migrateResourceGroup, snapName, createSnap, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to copy snapshot from source snapshot", "snapshot", from)
	}
	createSnapRes, err := createSnapshotPoller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Poller failed to retrieve snapshot")
	}
	snap, err := s.SnapshotGet(ctx, blockstorage.StringFromPtr(createSnapRes.ID))
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to Get Snapshot after create, snaphotName", "snaphotName", snapName)
	}
	*snap.Volume = *from.Volume
	return snap, nil
}

func getCopyOptions(ctx context.Context) (*storage.CopyOptions, error) {
	var copyOptions *storage.CopyOptions
	if t, ok := ctx.Deadline(); ok {
		time := time.Until(t).Seconds()
		if time <= 0 {
			return nil, errkit.New("Context deadline exceeded, cannot copy snapshot")
		}
		copyOptions = &storage.CopyOptions{
			Timeout: uint(time),
		}
	}
	return copyOptions, nil
}

func getSnapshotObject(
	blob *storage.Blob,
	from,
	to blockstorage.Snapshot,
	snapName,
	storageAccountID string,
) armcompute.Snapshot {
	blobURI := blob.GetURL()

	var tags = make(map[string]string)
	for _, tag := range from.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}
	tags = blockstorage.SanitizeTags(ktags.GetTags(tags))

	createSnap := armcompute.Snapshot{
		Name:     blockstorage.StringPtr(snapName),
		Location: blockstorage.StringPtr(to.Region),
		Tags:     *blockstorage.StringMapPtr(tags),
		Properties: &armcompute.SnapshotProperties{
			CreationData: &armcompute.CreationData{
				CreateOption:     azto.Ptr(armcompute.DiskCreateOptionImport),
				StorageAccountID: blockstorage.StringPtr(storageAccountID),
				SourceURI:        blockstorage.StringPtr(blobURI),
			},
		},
	}
	return createSnap
}

func isMigrateStorageAccountorKey(migrateStorageAccount, migrateStorageKey string) bool {
	return migrateStorageAccount == "" || migrateStorageKey == ""
}

func (s *AdStorage) revokeAccess(ctx context.Context, rg, name, id string) {
	poller, err := s.azCli.SnapshotsClient.BeginRevokeAccess(ctx, rg, name, nil)
	if err != nil {
		log.Print("Failed to finish the revoke request", field.M{"error": err.Error()})
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Print("failed to pull the result", field.M{"error": err.Error()})
	}

	if err != nil {
		log.Print("Failed to revoke access from snapshot", field.M{"snapshot": id})
	}
}

func deleteBlob(blob *storage.Blob, blobName string) {
	_, err := blob.DeleteIfExists(nil)
	if err != nil {
		log.Print("Failed to delete blob", field.M{"blob": blobName})
	}
}

func (s *AdStorage) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	snapID, err := uuid.NewV1()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create UUID")
	}
	snapName := fmt.Sprintf(snapshotNameFmt, snapID.String())
	tags = blockstorage.SanitizeTags(ktags.GetTags(tags))
	region, _, err := getLocationInfo(volume.Az)
	if err != nil {
		return nil, errkit.Wrap(err, "Could not get region from zone", "zone", volume.Az)
	}
	createSnap := armcompute.Snapshot{
		Name:     blockstorage.StringPtr(snapName),
		Location: blockstorage.StringPtr(region),
		Tags:     *blockstorage.StringMapPtr(tags),
		Properties: &armcompute.SnapshotProperties{
			CreationData: &armcompute.CreationData{
				CreateOption:     azto.Ptr(armcompute.DiskCreateOptionCopy),
				SourceResourceID: blockstorage.StringPtr(volume.ID),
			},
		},
	}
	pollerResp, err := s.azCli.SnapshotsClient.BeginCreateOrUpdate(ctx, s.azCli.ResourceGroup, snapName, createSnap, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create snapshot for volume", "volume", volume)
	}
	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to Get Snapshot after create, snaphotName", "snaphotName", snapName)
	}
	blockSnapshot, err := s.snapshotParse(ctx, resp.Snapshot)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to Parse Snapshot, snaphotName", "snaphotName", snapName)
	}

	blockSnapshot.Volume = &volume
	return blockSnapshot, nil
}

func (s *AdStorage) SnapshotCreateWaitForCompletion(ctx context.Context, snap *blockstorage.Snapshot) error {
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		snapshot, err := s.SnapshotGet(ctx, snap.ID)
		if err == nil && snapshot.ProvisioningState == string(armcompute.GalleryProvisioningStateSucceeded) {
			return true, nil
		}

		return false, nil
	})
	return err
}

const (
	snapshotIDRegEx = `/subscriptions/(.*)/resourceGroups/(.*)/providers/Microsoft.Compute/snapshots/(.*)`
	diskIDRegEx     = `/subscriptions/(.*)/resourceGroups/(.*)/providers/Microsoft.Compute/disks/(.*)`
)

var diskIDRe = regexp.MustCompile(diskIDRegEx)
var snapIDRe = regexp.MustCompile(snapshotIDRegEx)

//nolint:unparam
func parseDiskID(id string) (subscription string, resourceGroup string, name string, err error) {
	comps := diskIDRe.FindStringSubmatch(id)
	if len(comps) != 4 {
		return "", "", "", errkit.New("Failed to parse Disk ID" + id)
	}
	return comps[1], comps[2], comps[3], nil
}

//nolint:unparam
func parseSnapshotID(id string) (subscription string, resourceGroup string, name string, err error) {
	comps := snapIDRe.FindStringSubmatch(id)
	if len(comps) != 4 {
		return "", "", "", errkit.New("Failed to parse Snapshot ID" + id)
	}
	return comps[1], comps[2], comps[3], nil
}

func (s *AdStorage) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	_, rg, name, err := parseSnapshotID(snapshot.ID)
	if err != nil {
		return errkit.Wrap(err, "SnapshotClient.Delete: Failure in parsing snapshot ID", "snapshotID", snapshot.ID)
	}
	poller, err := s.azCli.SnapshotsClient.BeginDelete(ctx, rg, name, nil)
	if err != nil {
		return errkit.Wrap(err, "SnapshotClient.Delete: Failed to delete snapshot with ID", "snapshotID", snapshot.ID)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return errkit.Wrap(err, "SnapshotClient.Delete: Error while waiting for snapshot with ID to get deleted", "snapshotID", snapshot.ID)
}

func (s *AdStorage) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	_, rg, name, err := parseSnapshotID(id)
	if err != nil {
		return nil, errkit.Wrap(err, "SnapshotsClient.Get: Failure in parsing snapshot ID", "id", id)
	}
	snapRes, err := s.azCli.SnapshotsClient.Get(ctx, rg, name, nil)
	if err != nil {
		if isNotFoundError(err) {
			err = errkit.Wrap(err, blockstorage.SnapshotDoesNotExistError)
		}
		return nil, errkit.Wrap(err, "SnapshotsClient.Get: Failed to get snapshot with ID", "id", id)
	}

	return s.snapshotParse(ctx, snapRes.Snapshot)
}

func (s *AdStorage) VolumeParse(ctx context.Context, volume interface{}) (*blockstorage.Volume, error) {
	vol, ok := volume.(armcompute.Disk)
	if !ok {
		return nil, errkit.New(fmt.Sprintf("Volume is not of type *armcompute.Disk, volume: %v", volume)) // TODO: Fixme
	}
	encrypted := false
	if vol.Properties.EncryptionSettingsCollection != nil &&
		vol.Properties.EncryptionSettingsCollection.Enabled != nil {
		encrypted = *vol.Properties.EncryptionSettingsCollection.Enabled
	}
	tags := map[string]string{"": ""}
	if vol.Tags != nil {
		tags = blockstorage.StringMap(vol.Tags)
	}
	az := blockstorage.StringFromPtr(vol.Location)
	if z := vol.Zones; len(z) > 0 {
		az = az + "-" + *(z[0])
	}
	volumeType := ""
	if vol.SKU != nil &&
		vol.SKU.Name != nil {
		volumeType = string(*vol.SKU.Name)
	} else {
		return nil, errkit.New("Volume type is not available")
	}

	volID := ""
	if vol.ID != nil {
		volID = blockstorage.StringFromPtr(vol.ID)
	} else {
		return nil, errkit.New("Volume Id is not available")
	}
	diskSize := int64(0)
	if vol.Properties != nil &&
		vol.Properties.DiskSizeBytes != nil {
		diskSize = blockstorage.Int64(vol.Properties.DiskSizeBytes)
	}

	var creationTime = time.Now()
	if vol.Properties != nil && vol.Properties.TimeCreated != nil {
		creationTime = *vol.Properties.TimeCreated
	}

	var managedBy = "N.A."
	if vol.ManagedBy != nil {
		managedBy = blockstorage.StringFromPtr(vol.ManagedBy)
	}

	return &blockstorage.Volume{
		Type:         s.Type(),
		ID:           volID,
		Encrypted:    encrypted,
		SizeInBytes:  diskSize,
		Az:           az,
		Tags:         blockstorage.MapToKeyValue(tags),
		VolumeType:   volumeType,
		CreationTime: blockstorage.TimeStamp(creationTime),
		Attributes:   map[string]string{"Users": managedBy},
	}, nil
}

func (s *AdStorage) SnapshotParse(ctx context.Context, snapshot interface{}) (*blockstorage.Snapshot, error) {
	if snap, ok := snapshot.(armcompute.Snapshot); ok {
		return s.snapshotParse(ctx, snap)
	}
	return nil, errkit.New(fmt.Sprintf("Snapshot is not of type *armcompute.Snapshot, snapshot: %v", snapshot)) // TODO: Fixme
}

func (s *AdStorage) snapshotParse(ctx context.Context, snap armcompute.Snapshot) (*blockstorage.Snapshot, error) {
	snapID := ""
	if snap.ID != nil {
		snapID = *snap.ID
	} else {
		return nil, errkit.New("Snapshot ID is missing")
	}
	vol := &blockstorage.Volume{
		Type: s.Type(),
		ID:   snapID,
	}

	snapCreationTime := time.Now()
	if snap.Properties != nil && snap.Properties.TimeCreated != nil {
		snapCreationTime = *snap.Properties.TimeCreated
	}

	encrypted := false
	if snap.Properties.EncryptionSettingsCollection != nil &&
		snap.Properties.EncryptionSettingsCollection.Enabled != nil {
		encrypted = *snap.Properties.EncryptionSettingsCollection.Enabled
	}
	tags := map[string]string{}
	if snap.Tags != nil {
		tags = blockstorage.StringMap(snap.Tags)
	}

	diskSize := azto.Ptr(int64(0))
	if snap.Properties != nil &&
		snap.Properties.DiskSizeBytes != nil {
		diskSize = snap.Properties.DiskSizeBytes
	}

	region := ""
	if snap.Location != nil {
		region = *snap.Location
	}
	return &blockstorage.Snapshot{
		Encrypted:         encrypted,
		ID:                snapID,
		Region:            region,
		SizeInBytes:       blockstorage.Int64(diskSize),
		Tags:              blockstorage.MapToKeyValue(tags),
		Type:              s.Type(),
		Volume:            vol,
		CreationTime:      blockstorage.TimeStamp(snapCreationTime),
		ProvisioningState: *snap.Properties.ProvisioningState,
	}, nil
}

func (s *AdStorage) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	// (ilya): It looks like azure doesn't support search by tags
	// List does listing per Subscription
	pager := s.azCli.DisksClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, errkit.Wrap(err, "DisksClient.List in VolumesList")
		}
		for _, disk := range page.Value {
			vol, err := s.VolumeParse(ctx, *disk)
			if err != nil {
				return nil, errkit.Wrap(err, "DisksClient.List in VolumesList, failure in parsing Volume")
			}
			vols = append(vols, vol)
		}
	}
	return vols, nil
}

func (s *AdStorage) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	// (ilya): It looks like azure doesn't support search by tags
	// List does listing per Subscription
	pager := s.azCli.SnapshotsClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, errkit.Wrap(err, "SnapshotsClient.List in SnapshotsList")
		}
		for _, snap := range page.Value {
			parsedSnap, err := s.SnapshotParse(ctx, *snap)
			if err != nil {
				log.WithError(err).Print("Incorrect Snaphost type", field.M{"SnapshotID": snap.ID})
				continue
			}
			snaps = append(snaps, parsedSnap)
		}
	}
	snaps = blockstorage.FilterSnapshotsWithTags(snaps, blockstorage.SanitizeTags(tags))
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

	diskID, err := uuid.NewV1()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create UUID")
	}
	diskName := fmt.Sprintf(volumeNameFmt, diskID.String())
	tags = blockstorage.SanitizeTags(tags)
	createDisk := armcompute.Disk{
		Name:     blockstorage.StringPtr(diskName),
		Tags:     *blockstorage.StringMapPtr(tags),
		Location: blockstorage.StringPtr(region),
		Properties: &armcompute.DiskProperties{
			CreationData: &armcompute.CreationData{
				CreateOption:     azto.Ptr(armcompute.DiskCreateOptionCopy),
				SourceResourceID: blockstorage.StringPtr(snapshot.ID),
			},
		},
	}
	if id != "" {
		createDisk.Zones = blockstorage.SliceStringPtr([]string{id})
	}
	for _, saType := range armcompute.PossibleDiskStorageAccountTypesValues() {
		if string(saType) == snapshot.Volume.VolumeType {
			createDisk.SKU = &armcompute.DiskSKU{
				Name: azto.Ptr(saType),
			}
		}
	}
	poller, err := s.azCli.DisksClient.BeginCreateOrUpdate(ctx, s.azCli.ResourceGroup, diskName, createDisk, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "DiskCLient.CreateOrUpdate in VolumeCreateFromSnapshot", "diskName", diskName, "snapshotID", snapshot.ID)
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "DiskCLient.CreateOrUpdate in VolumeCreateFromSnapshot", "diskName", diskName, "snapshotID", snapshot.ID)
	}
	return s.VolumeParse(ctx, resp.Disk)
}

func (s *AdStorage) getRegionAndZoneID(ctx context.Context, sourceRegion, volAz string) (string, string, error) {
	// check if current node region is zoned or not
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
		return "", "", errkit.New(fmt.Sprintf("Length of zone slice should be 1, got %d", len(zones)))
	}

	region, id, err := getLocationInfo(zones[0])
	return region, id, errkit.Wrap(err, "Could not get region from zone", "zone", zones[0])
}

func getLocationInfo(az string) (string, string, error) {
	if az == "" {
		return "", "", errkit.New("zone value is empty")
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
			snap, err := s.azCli.SnapshotsClient.Get(ctx, rg, name, nil)
			if err != nil {
				return errkit.Wrap(err, "SnapshotsClient.Get in SetTags", "snapshotID", res.ID)
			}
			tags = ktags.AddMissingTags(blockstorage.StringMap(snap.Tags), ktags.GetTags(tags))
			snapProperties := armcompute.SnapshotUpdate{
				Tags: *blockstorage.StringMapPtr(blockstorage.SanitizeTags(tags)),
			}
			poller, err := s.azCli.SnapshotsClient.BeginUpdate(ctx, rg, name, snapProperties, nil)
			if err != nil {
				return errkit.Wrap(err, "SnapshotsClient.Update in SetTags", "snapshotID", name)
			}
			_, err = poller.PollUntilDone(ctx, nil)
			return errkit.Wrap(err, "SnapshotsClient.Update in SetTags", "snapshotID", name)
		}
	case *blockstorage.Volume:
		{
			_, rg, volID, err := parseDiskID(res.ID)
			if err != nil {
				return err
			}
			vol, err := s.azCli.DisksClient.Get(ctx, rg, volID, nil)
			if err != nil {
				return errkit.Wrap(err, "DiskClient.Get in SetTags", "volumeID", volID)
			}
			tags = ktags.AddMissingTags(blockstorage.StringMap(vol.Tags), ktags.GetTags(tags))

			diskProperties := armcompute.DiskUpdate{
				Tags: *blockstorage.StringMapPtr(blockstorage.SanitizeTags(tags)),
			}
			poller, err := s.azCli.DisksClient.BeginUpdate(ctx, rg, volID, diskProperties, nil)
			if err != nil {
				return errkit.Wrap(err, "DiskClient.Update in SetTags", "volumeID", volID)
			}
			_, err = poller.PollUntilDone(ctx, nil)
			return errkit.Wrap(err, "DiskClient.Update in SetTags", "volumeID", volID)
		}
	default:
		return errkit.New(fmt.Sprintf("Unknown resource type %v", res)) // TODO: Fixme
	}
}

func (s *AdStorage) FromRegion(ctx context.Context, region string) ([]string, error) {
	regionMap, err := s.dynamicRegionMapAzure(ctx)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to fetch dynamic region map for region", "region", region)
	}
	zones, ok := regionMap[region]
	if !ok {
		return nil, errkit.New(fmt.Sprintf("Zones for region %s not found", region))
	}
	return zones, nil
}

func (s *AdStorage) GetRegions(ctx context.Context) ([]string, error) {
	regionMap, err := s.dynamicRegionMapAzure(ctx)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to fetch dynamic region map")
	}
	regions := []string{}
	for region := range regionMap {
		regions = append(regions, region)
	}
	return regions, nil
}

func (s *AdStorage) SnapshotRestoreTargets(ctx context.Context, snapshot *blockstorage.Snapshot) (global bool, regionsAndZones map[string][]string, err error) {
	// A few checks from VolumeCreateFromSnapshot
	if snapshot.Volume == nil {
		return false, nil, errkit.New("Snapshot volume information not available")
	}
	if snapshot.Volume.VolumeType == "" {
		return false, nil, errkit.New("Required VolumeType not set")
	}

	zl, err := s.FromRegion(ctx, snapshot.Region)
	if err != nil {
		return false, nil, err
	}
	return false, map[string][]string{snapshot.Region: zl}, nil
}

// dynamicRegionMapAzure derives a mapping from Regions to zones for Azure. Depends on subscriptionID
func (s *AdStorage) dynamicRegionMapAzure(ctx context.Context) (map[string][]string, error) {
	subscriptionsClient := s.azCli.SubscriptionsClient
	regionMap := make(map[string]LocationZoneMap)

	locationsPager := subscriptionsClient.NewListLocationsPager(s.azCli.SubscriptionID, &armsubscriptions.ClientListLocationsOptions{IncludeExtendedLocations: nil})
	for locationsPager.More() {
		page, err := locationsPager.NextPage(ctx)
		if err != nil {
			return nil, errkit.Wrap(err, "failed to advance page")
		}
		for _, location := range page.Value {
			if location != nil && location.Name != nil {
				regionMap[*location.Name] = make(LocationZoneMap)
			} else {
				continue
			}
		}
	}

	skusClient := s.azCli.SKUsClient
	skusPager := skusClient.NewListPager(&armcompute.ResourceSKUsClientListOptions{Filter: nil,
		IncludeExtendedLocations: nil})
	for skusPager.More() {
		skuResults, err := skusPager.NextPage(ctx)
		if err != nil {
			return nil, errkit.Wrap(err, "failed to advance page")
		}
		for _, skuResult := range skuResults.Value {
			if skuResult.Name != nil && skuResult.ResourceType != nil && *skuResult.ResourceType == "disks" {
				s.mapLocationToZone(skuResult, &regionMap)
			}
		}
	}
	// convert to map of []string
	regionMapResult := make(map[string][]string)
	for region, zoneSet := range regionMap {
		var zoneArray []string
		for zone := range zoneSet {
			zoneArray = append(zoneArray, region+"-"+zone)
		}
		regionMapResult[region] = zoneArray
	}
	return regionMapResult, nil
}

func (s *AdStorage) mapLocationToZone(skuResult *armcompute.ResourceSKU, regionMap *map[string]LocationZoneMap) {
	var rm = *regionMap
	for _, locationInfo := range skuResult.LocationInfo {
		location := ""
		if locationInfo.Location != nil {
			location = *locationInfo.Location
		} else {
			continue
		}
		if val, ok := rm[location]; ok {
			for _, zone := range locationInfo.Zones {
				val[*zone] = struct{}{}
			}
			rm[location] = val
		}
	}
}

func isNotFoundError(err error) bool {
	var azerr azcore.ResponseError
	if errkit.As(err, azerr) {
		return azerr.StatusCode == http.StatusNotFound
	}
	return false
}
