// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcepd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	"github.com/kanisterio/kanister/pkg/blockstorage/zone"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/volume"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

var _ blockstorage.Provider = (*GpdStorage)(nil)
var _ zone.Mapper = (*GpdStorage)(nil)

// GpdStorage implements blockstorage.Provider
type GpdStorage struct {
	service *compute.Service
	project string
}

const (
	operationDone    = "DONE"
	operationRunning = "RUNNING"
	operationPending = "PENDING"
	volumeNameFmt    = "vol-%s"
	snapshotNameFmt  = "snap-%s"
)

// Type is part of blockstorage.Provider
func (s *GpdStorage) Type() blockstorage.Type {
	return blockstorage.TypeGPD
}

// NewProvider returns a provider for the GCP storage type
func NewProvider(config map[string]string) (blockstorage.Provider, error) {
	serviceKey := config[blockstorage.GoogleServiceKey]
	gCli, err := NewClient(context.Background(), serviceKey)
	if err != nil {
		return nil, err
	}
	if projectID, ok := config[blockstorage.GoogleProjectID]; ok {
		gCli.ProjectID = projectID
	}
	return &GpdStorage{
		service: gCli.Service,
		project: gCli.ProjectID}, nil
}

// VolumeGet is part of blockstorage.Provider
func (s *GpdStorage) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	var err error
	var disk *compute.Disk

	if isMultiZone(zone) {
		region, err := getRegionFromZones(zone)
		if err != nil {
			return nil, err
		}
		disk, err = s.service.RegionDisks.Get(s.project, region, id).Context(ctx).Do()
		if err != nil {
			return nil, err
		}
	} else {
		disk, err = s.service.Disks.Get(s.project, zone, id).Context(ctx).Do()
		if err != nil {
			return nil, err
		}
	}
	mv := s.volumeParse(ctx, disk, zone)
	return mv, nil
}

// VolumeCreate is part of blockstorage.Provider
func (s *GpdStorage) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	var resp *compute.Operation
	var err error
	tags := make(map[string]string, len(volume.Tags))
	for _, tag := range volume.Tags {
		tags[tag.Key] = tag.Value
	}
	tags = blockstorage.SanitizeTags(ktags.GetTags(tags))

	id, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create UUID")
	}
	createDisk := &compute.Disk{
		Name:   fmt.Sprintf(volumeNameFmt, id.String()),
		SizeGb: blockstorage.SizeInGi(volume.SizeInBytes),
		Type:   volume.VolumeType,
		Labels: tags,
	}
	if isMultiZone(volume.Az) {
		region, err := getRegionFromZones(volume.Az)
		if err != nil {
			return nil, err
		}
		replicaZones, err := s.getSelfLinks(ctx, splitZones(volume.Az))
		if err != nil {
			return nil, err
		}
		createDisk.ReplicaZones = replicaZones
		if resp, err = s.service.RegionDisks.Insert(s.project, region, createDisk).Context(ctx).Do(); err != nil {
			return nil, err
		}
	} else {
		if resp, err = s.service.Disks.Insert(s.project, volume.Az, createDisk).Context(ctx).Do(); err != nil {
			return nil, err
		}
	}
	if err := s.waitOnOperation(ctx, resp, volume.Az); err != nil {
		return nil, err
	}
	return s.VolumeGet(ctx, createDisk.Name, volume.Az)
}

// VolumeDelete is part of blockstorage.Provider
func (s *GpdStorage) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	var op *compute.Operation
	var err error
	var region string

	if isMultiZone(volume.Az) {
		region, err = getRegionFromZones(volume.Az)
		if err != nil {
			return err
		}
		op, err = s.service.RegionDisks.Delete(s.project, region, volume.ID).Context(ctx).Do()
	} else {
		op, err = s.service.Disks.Delete(s.project, volume.Az, volume.ID).Context(ctx).Do()
	}
	if isNotFoundError(err) {
		log.Debug().Print("Cannot delete volume.", field.M{"VolumeID": volume.ID, "reason": "Volume not found"})
		return nil
	}
	if err != nil {
		return err
	}
	// For Regional Disks, op = nil if we try to delete an already deleted volume. Hence, the following check!
	if op == nil {
		return nil
	}
	return s.waitOnOperation(ctx, op, volume.Az)
}

// SnapshotCopy is part of blockstorage.Provider
func (s *GpdStorage) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.Errorf("Not implemented")
}

// SnapshotCopyWithArgs is part of blockstorage.Provider
func (s *GpdStorage) SnapshotCopyWithArgs(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot, args map[string]string) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Copy Snapshot with Args not implemented")
}

// SnapshotCreate is part of blockstorage.Provider
func (s *GpdStorage) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	rbID, uerr := uuid.NewV1()
	if uerr != nil {
		return nil, errors.Wrap(uerr, "Failed to create UUID")
	}
	rb := &compute.Snapshot{
		Name:   fmt.Sprintf(snapshotNameFmt, rbID.String()),
		Labels: blockstorage.SanitizeTags(ktags.GetTags(tags)),
	}
	var err error
	if isMultiZone(volume.Az) {
		var region string
		region, err = getRegionFromZones(volume.Az)
		if err != nil {
			return nil, err
		}
		_, err = s.service.RegionDisks.CreateSnapshot(s.project, region, volume.ID, rb).Context(ctx).Do()
	} else {
		_, err = s.service.Disks.CreateSnapshot(s.project, volume.Az, volume.ID, rb).Context(ctx).Do()
	}
	if err != nil {
		return nil, err
	}

	var snap *compute.Snapshot
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		snap, err = s.service.Snapshots.Get(s.project, rb.Name).Context(ctx).Do()
		if err != nil {
			if strings.Contains(err.Error(), "notFound") {
				return false, nil
			}
			return false, err
		}
		if snap.StorageBytesStatus != "UP_TO_DATE" {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	rs := s.snapshotParse(ctx, snap)
	rs.Volume = &volume
	return rs, nil
}

// SnapshotCreateWaitForCompletion is part of blockstorage.Provider
func (s *GpdStorage) SnapshotCreateWaitForCompletion(ctx context.Context, snap *blockstorage.Snapshot) error {
	if err := s.waitOnSnapshotID(ctx, snap.ID); err != nil {
		return errors.Wrapf(err, "Waiting on snapshot %v", snap)
	}
	return nil
}

// SnapshotDelete is part of blockstorage.Provider
func (s *GpdStorage) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	op, err := s.service.Snapshots.Delete(s.project, snapshot.ID).Context(ctx).Do()
	if isNotFoundError(err) {
		log.Debug().Print("Cannot delete snapshot", field.M{"SnapshotID": snapshot.ID, "reason": "Snapshot not found"})
		return nil
	}
	if err != nil {
		return err
	}
	return s.waitOnOperation(ctx, op, "")
}

// SnapshotGet is part of blockstorage.Provider
func (s *GpdStorage) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	snap, err := s.service.Snapshots.Get(s.project, id).Context(ctx).Do()
	if err != nil {
		if isNotFoundError(err) {
			return nil, errors.Wrap(err, blockstorage.SnapshotDoesNotExistError)
		}
		return nil, err
	}
	return s.snapshotParse(ctx, snap), nil
}

func (s *GpdStorage) volumeParse(ctx context.Context, volume interface{}, zone string) *blockstorage.Volume {
	vol := volume.(*compute.Disk)
	volCreationTime, err := time.Parse(time.RFC3339, vol.CreationTimestamp)
	if err != nil {
		log.Error().Print("Cannot parse GCP Disk timestamp")
	}

	return &blockstorage.Volume{
		Type:         s.Type(),
		ID:           vol.Name,
		Encrypted:    false,
		SizeInBytes:  vol.SizeGb * blockstorage.BytesInGi,
		Az:           filepath.Base(zone),
		Tags:         blockstorage.MapToKeyValue(vol.Labels),
		VolumeType:   vol.Type,
		CreationTime: blockstorage.TimeStamp(volCreationTime),
		Attributes:   map[string]string{"Users": strings.Join(vol.Users, ",")},
	}
}

func (s *GpdStorage) snapshotParse(ctx context.Context, snap *compute.Snapshot) *blockstorage.Snapshot {
	var encrypted bool
	if snap.SnapshotEncryptionKey == nil {
		encrypted = false
	} else {
		encrypted = true
	}
	vol := &blockstorage.Volume{
		Type: s.Type(),
		ID:   snap.SourceDisk,
	}
	snapCreationTIme, err := time.Parse(time.RFC3339, snap.CreationTimestamp)
	if err != nil {
		log.Error().Print("Cannot parse GCP Snapshot timestamp")
	}
	// TODO: fix getting region from zone
	return &blockstorage.Snapshot{
		Encrypted:    encrypted,
		ID:           snap.Name,
		Region:       "",
		SizeInBytes:  snap.StorageBytes,
		Tags:         blockstorage.MapToKeyValue(snap.Labels),
		Type:         s.Type(),
		Volume:       vol,
		CreationTime: blockstorage.TimeStamp(snapCreationTIme),
	}
}

// VolumesList is part of blockstorage.Provider
func (s *GpdStorage) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	fltrs := blockstorage.MapToString(blockstorage.SanitizeTags(tags), " AND ", ":", "labels.")
	if isMultiZone(zone) {
		region, err := getRegionFromZones(zone)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get region from zones %s", zone)
		}
		req := s.service.RegionDisks.List(s.project, region).Filter(fltrs)
		if err := req.Pages(ctx, func(page *compute.DiskList) error {
			for _, disk := range page.Items {
				vol := s.volumeParse(ctx, disk, zone)
				vols = append(vols, vol)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		req := s.service.Disks.List(s.project, zone).Filter(fltrs)
		if err := req.Pages(ctx, func(page *compute.DiskList) error {
			for _, disk := range page.Items {
				vol := s.volumeParse(ctx, disk, zone)
				vols = append(vols, vol)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return vols, nil
}

// SnapshotsList is part of blockstorage.Provider. It filters on tags.
func (s *GpdStorage) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	fltrs := blockstorage.MapToString(blockstorage.SanitizeTags(tags), " AND ", ":", "labels.")
	req := s.service.Snapshots.List(s.project).Filter(fltrs)
	if err := req.Pages(ctx, func(page *compute.SnapshotList) error {
		for _, snapshot := range page.Items {
			snap := s.snapshotParse(ctx, snapshot)
			snaps = append(snaps, snap)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return snaps, nil
}

// VolumeCreateFromSnapshot is part of blockstorage.Provider
func (s *GpdStorage) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	snap, err := s.service.Snapshots.Get(s.project, snapshot.ID).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	if snapshot.Volume.VolumeType == "" || snapshot.Volume.Az == "" {
		return nil, errors.Errorf("Required volume fields not available, volumeType: %s, Az: %s", snapshot.Volume.VolumeType, snapshot.Volume.Az)
	}

	// Incorporate pre-existing tags if overrides don't already exist
	// in provided tags
	for _, tag := range snapshot.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}
	createDiskID, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create UUID")
	}
	createDisk := &compute.Disk{
		Name:           fmt.Sprintf(volumeNameFmt, createDiskID.String()),
		SizeGb:         blockstorage.SizeInGi(snapshot.Volume.SizeInBytes),
		Type:           snapshot.Volume.VolumeType,
		Labels:         blockstorage.SanitizeTags(ktags.GetTags(tags)),
		SourceSnapshot: snap.SelfLink,
	}

	var resp *compute.Operation
	var zones []string
	var region string
	// Validate Zones
	if region, err = getRegionFromZones(snapshot.Volume.Az); err != nil {
		return nil, errors.Wrapf(err, "Could not validate zones: %s", snapshot.Volume.Az)
	}
	kubeCli, err := kube.NewClient()
	if err != nil {
		// TODO: Pull KubeCli creation out of kanister
		log.WithError(err).Print("Failed to initialize kubernetes client")
	}
	zones = splitZones(snapshot.Volume.Az)
	zones, err = zone.FromSourceRegionZone(ctx, s, kubeCli, region, zones...)
	if err != nil {
		return nil, err
	}
	volZone := strings.Join(zones, volume.RegionZoneSeparator)
	// Validates new Zones
	region, err = getRegionFromZones(volZone)
	if err != nil {
		return nil, err
	}
	newZones := splitZones(volZone)

	if len(newZones) == 1 {
		resp, err = s.service.Disks.Insert(s.project, volZone, createDisk).Context(ctx).Do()
	} else {
		zones, err = s.getSelfLinks(ctx, newZones)
		if err != nil {
			return nil, err
		}
		createDisk.ReplicaZones = zones
		resp, err = s.service.RegionDisks.Insert(s.project, region, createDisk).Context(ctx).Do()
	}
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create volume from snapshot")
	}

	if err = s.waitOnOperation(ctx, resp, volZone); err != nil {
		return nil, err
	}

	return s.VolumeGet(ctx, createDisk.Name, volZone)
}

// SetTags is part of blockstorage.Provider
func (s *GpdStorage) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch res := resource.(type) {
	case *blockstorage.Snapshot:
		{
			snap, err := s.service.Snapshots.Get(s.project, res.ID).Context(ctx).Do()
			if err != nil {
				return err
			}
			tags = ktags.AddMissingTags(snap.Labels, ktags.GetTags(tags))
			slr := &compute.GlobalSetLabelsRequest{
				LabelFingerprint: snap.LabelFingerprint,
				Labels:           blockstorage.SanitizeTags(tags),
			}
			op, err := s.service.Snapshots.SetLabels(s.project, snap.Name, slr).Do()
			if err != nil {
				return err
			}
			return s.waitOnOperation(ctx, op, "")
		}
	case *blockstorage.Volume:
		{
			var op *compute.Operation
			if isMultiZone(res.Az) {
				region, err := getRegionFromZones(res.Az)
				if err != nil {
					return err
				}
				vol, err := s.service.RegionDisks.Get(s.project, region, res.ID).Context(ctx).Do()
				if err != nil {
					return err
				}
				tags = ktags.AddMissingTags(vol.Labels, ktags.GetTags(tags))
				slr := &compute.RegionSetLabelsRequest{
					LabelFingerprint: vol.LabelFingerprint,
					Labels:           blockstorage.SanitizeTags(tags),
				}
				op, err = s.service.RegionDisks.SetLabels(s.project, region, vol.Name, slr).Do()
				if err != nil {
					return err
				}
				return s.waitOnOperation(ctx, op, res.Az)
			}
			vol, err := s.service.Disks.Get(s.project, res.Az, res.ID).Context(ctx).Do()
			if err != nil {
				return err
			}
			tags = ktags.AddMissingTags(vol.Labels, ktags.GetTags(tags))
			slr := &compute.ZoneSetLabelsRequest{
				LabelFingerprint: vol.LabelFingerprint,
				Labels:           blockstorage.SanitizeTags(tags),
			}
			op, err = s.service.Disks.SetLabels(s.project, res.Az, vol.Name, slr).Do()
			if err != nil {
				return err
			}
			return s.waitOnOperation(ctx, op, res.Az)
		}
	default:
		return errors.Errorf("Unknown resource type %v (%T)", res, res)
	}
}

// waitOnOperation waits for the operation to be done
func (s *GpdStorage) waitOnOperation(ctx context.Context, op *compute.Operation, zone string) error {
	waitBackoff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    1 * time.Second,
		Max:    10 * time.Second,
	}

	return poll.WaitWithBackoff(ctx, waitBackoff, func(ctx context.Context) (bool, error) {
		var err error
		var region string
		switch {
		case zone == "":
			op, err = s.service.GlobalOperations.Get(s.project, op.Name).Context(ctx).Do()
		case isMultiZone(zone):
			region, err = getRegionFromZones(zone)
			if err != nil {
				return false, err
			}
			op, err = s.service.RegionOperations.Get(s.project, region, op.Name).Context(ctx).Do()
		default:
			op, err = s.service.ZoneOperations.Get(s.project, zone, op.Name).Context(ctx).Do()
		}
		if err != nil {
			return false, err
		}
		switch op.Status {
		case operationDone:
			if op.Error != nil {
				errJSON, merr := op.Error.MarshalJSON()
				if merr != nil {
					return false, errors.Errorf("Operation %s failed. Failed to marshal error string with error %s", op.OperationType, merr)
				}
				return false, errors.Errorf("%s", errJSON)
			}
			log.Print("Operation done", field.M{"OperationType": op.OperationType})
			return true, nil
		case operationPending, operationRunning:
			log.Debug().Print("Operation status update", field.M{"Operation": op.OperationType, "Status": op.Status, "Status message": op.StatusMessage, "Progress": op.Progress})
			return false, nil
		default:
			return false, errors.Errorf("Unknown operation status")
		}
	})
}

// waitOnSnapshotID waits for the snapshot to be created
func (s *GpdStorage) waitOnSnapshotID(ctx context.Context, id string) error {
	snapWaitBackoff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    1 * time.Second,
		Max:    10 * time.Second,
	}
	return poll.WaitWithBackoff(ctx, snapWaitBackoff, func(ctx context.Context) (bool, error) {
		snap, err := s.service.Snapshots.Get(s.project, id).Context(ctx).Do()
		if err != nil {
			return false, errors.Wrapf(err, "Snapshot not found")
		}
		if snap.Status == "FAILED" {
			return false, errors.New("Snapshot GCP volume failed")
		}
		if snap.Status == "READY" {
			log.Print("Snapshot completed", field.M{"SnapshotID": id})
			return true, nil
		}
		log.Debug().Print("Snapshot status", field.M{"snapshot_id": id, "status": snap.Status})
		return false, nil
	})
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	ae, ok := err.(*googleapi.Error)
	return ok && ae.Code == http.StatusNotFound
}

// FromRegion is part of zone.Mapper
func (s *GpdStorage) FromRegion(ctx context.Context, region string) ([]string, error) {
	rtzMap, err := s.dynamicRegionToZoneMap(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get region to zone map for region (%s)", region)
	}
	zones, ok := rtzMap[region]
	if !ok {
		return nil, fmt.Errorf("cannot get availability zones for region %s", region)
	}
	return zones, nil
}

func (s *GpdStorage) GetRegions(ctx context.Context) ([]string, error) {
	regionMap, err := s.dynamicRegionToZoneMap(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch dynamic region map")
	}
	regions := []string{}
	for region := range regionMap {
		regions = append(regions, region)
	}
	return regions, nil
}

// dynamicRegionToZoneMap returns the region to zone map fetched from the provider
func (s *GpdStorage) dynamicRegionToZoneMap(ctx context.Context) (map[string][]string, error) {
	regionMap := make(map[string][]string)
	req := s.service.Zones.List(s.project)
	if err := req.Pages(ctx, func(page *compute.ZoneList) error {
		for _, zone := range page.Items {
			regionURL, err := url.Parse(zone.Region)
			if err != nil {
				return err
			}
			region := path.Base(regionURL.Path)
			regionMap[region] = append(regionMap[region], zone.Name)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return regionMap, nil
}

func isMultiZone(az string) bool {
	return strings.Contains(az, volume.RegionZoneSeparator)
}

// getRegionFromZones function is used from the link below
// https://github.com/kubernetes-sigs/gcp-compute-persistent-disk-csi-driver/blob/master/pkg/common/utils.go#L103

func getRegionFromZones(az string) (string, error) {
	zones := splitZones(az)
	regions := sets.Set[string]{}
	if len(zones) < 1 {
		return "", errors.Errorf("no zones specified, zone: %s", az)
	}
	for _, zone := range zones {
		// Expected format of zone: {locale}-{region}-{zone}
		splitZone := strings.Split(zone, "-")
		if len(splitZone) != 3 {
			return "", errors.Errorf("zone in unexpected format, expected: {locale}-{region}-{zone}, got: %v", zone)
		}
		regions.Insert(strings.Join(splitZone[0:2], "-"))
	}
	if regions.Len() != 1 {
		return "", errors.Errorf("multiple or no regions gotten from zones, got: %v", regions)
	}
	return regions.UnsortedList()[0], nil
}

func (s *GpdStorage) getSelfLinks(ctx context.Context, zones []string) ([]string, error) {
	selfLinks := make([]string, len(zones))
	for i, zone := range zones {
		replicaZone, err := s.service.Zones.Get(s.project, zone).Context(ctx).Do()
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get Zone %s", zone)
		}
		selfLinks[i] = replicaZone.SelfLink
	}
	return selfLinks, nil
}

func splitZones(az string) []string {
	return strings.Split(az, volume.RegionZoneSeparator)
}
