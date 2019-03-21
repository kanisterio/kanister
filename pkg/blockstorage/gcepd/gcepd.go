package gcepd

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	"github.com/kanisterio/kanister/pkg/blockstorage/zone"
	"github.com/kanisterio/kanister/pkg/poll"
)

var _ blockstorage.Provider = (*gpdStorage)(nil)
var _ zone.Mapper = (*gpdStorage)(nil)

type gpdStorage struct {
	service *compute.Service
	project string
}

const (
	operationDone          = "DONE"
	operationRunning       = "RUNNING"
	operationPending       = "PENDING"
	bytesInGiB       int64 = 1024 * 1024 * 1024
	volumeNameFmt          = "vol-%s"
	snapshotNameFmt        = "snap-%s"
)

func (s *gpdStorage) Type() blockstorage.Type {
	return blockstorage.TypeGPD
}

// NewProvider returns a provider for the GCP storage type
func NewProvider(config map[string]string) (blockstorage.Provider, error) {
	serviceKey := config[blockstorage.GoogleServiceKey]
	gCli, err := NewClient(context.Background(), serviceKey)
	if err != nil {
		return nil, err
	}
	return &gpdStorage{
		service: gCli.Service,
		project: gCli.ProjectID}, nil
}

func (s *gpdStorage) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	disk, err := s.service.Disks.Get(s.project, zone, id).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	mv := s.volumeParse(ctx, disk)
	return mv, nil
}

func (s *gpdStorage) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	tags := make(map[string]string, len(volume.Tags))
	for _, tag := range volume.Tags {
		tags[tag.Key] = tag.Value
	}
	tags = blockstorage.SanitizeTags(ktags.GetTags(tags))

	createDisk := &compute.Disk{
		Name:   fmt.Sprintf(volumeNameFmt, uuid.NewV1().String()),
		SizeGb: volume.Size,
		Type:   volume.VolumeType,
		Labels: tags,
	}

	resp, err := s.service.Disks.Insert(s.project, volume.Az, createDisk).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	if err := s.waitOnOperation(ctx, resp, volume.Az); err != nil {
		return nil, err
	}

	return s.VolumeGet(ctx, createDisk.Name, volume.Az)
}

func (s *gpdStorage) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	op, err := s.service.Disks.Delete(s.project, volume.Az, volume.ID).Context(ctx).Do()
	if isNotFoundError(err) {
		log.Debugf("Cannot delete volume with id:%s Volume not found. ", volume.ID)
		return nil
	}
	if err != nil {
		return err
	}
	return s.waitOnOperation(ctx, op, volume.Az)
}

func (s *gpdStorage) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (s *gpdStorage) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	rb := &compute.Snapshot{
		Name:   fmt.Sprintf(snapshotNameFmt, uuid.NewV1().String()),
		Labels: blockstorage.SanitizeTags(ktags.GetTags(tags)),
	}
	_, err := s.service.Disks.CreateSnapshot(s.project, volume.Az, volume.ID, rb).Context(ctx).Do()
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
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	rs := s.snapshotParse(ctx, snap)
	rs.Volume = &volume
	return rs, nil
}

func (s *gpdStorage) SnapshotCreateWaitForCompletion(ctx context.Context, snap *blockstorage.Snapshot) error {
	if err := s.waitOnSnapshotID(ctx, snap.ID); err != nil {
		return errors.Wrapf(err, "Waiting on snapshot %v", snap)
	}
	return nil
}

func (s *gpdStorage) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	op, err := s.service.Snapshots.Delete(s.project, snapshot.ID).Context(ctx).Do()
	if isNotFoundError(err) {
		log.Debugf("Cannot delete snapshot with id:%s Snapshot not found. ", snapshot.ID)
		return nil
	}
	if err != nil {
		return err
	}
	return s.waitOnOperation(ctx, op, "")
}

func (s *gpdStorage) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	snap, err := s.service.Snapshots.Get(s.project, id).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return s.snapshotParse(ctx, snap), nil
}

func (s *gpdStorage) volumeParse(ctx context.Context, volume interface{}) *blockstorage.Volume {

	vol := volume.(*compute.Disk)
	volCreationTime, err := time.Parse(time.RFC3339, vol.CreationTimestamp)
	if err != nil {
		log.Errorf("Cannot parse GCP Disk timestamp")

	}

	return &blockstorage.Volume{
		Type:         s.Type(),
		ID:           vol.Name,
		Encrypted:    false,
		Size:         vol.SizeGb,
		Az:           filepath.Base(vol.Zone),
		Tags:         blockstorage.MapToKeyValue(vol.Labels),
		VolumeType:   vol.Type,
		CreationTime: blockstorage.TimeStamp(volCreationTime),
		Attributes:   map[string]string{"Users": strings.Join(vol.Users, ",")},
	}
}

func (s *gpdStorage) snapshotParse(ctx context.Context, snap *compute.Snapshot) *blockstorage.Snapshot {
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
		log.Errorf("Cannot parse GCP Snapshot timestamp")
	}
	// TODO: fix getting region from zone
	return &blockstorage.Snapshot{
		Encrypted:    encrypted,
		ID:           snap.Name,
		Region:       "",
		Size:         snap.StorageBytes / bytesInGiB,
		Tags:         blockstorage.MapToKeyValue(snap.Labels),
		Type:         s.Type(),
		Volume:       vol,
		CreationTime: blockstorage.TimeStamp(snapCreationTIme),
	}
}

func (s *gpdStorage) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	fltrs := blockstorage.MapToString(tags, " AND ", ":")
	req := s.service.Disks.List(s.project, zone).Filter(fltrs)
	if err := req.Pages(ctx, func(page *compute.DiskList) error {
		for _, disk := range page.Items {
			vol := s.volumeParse(ctx, disk)
			vols = append(vols, vol)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return vols, nil
}

func (s *gpdStorage) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	fltrs := blockstorage.MapToString(tags, " AND ", ":")
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

func (s *gpdStorage) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	snap, err := s.service.Snapshots.Get(s.project, snapshot.ID).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	// Incorporate pre-existing tags if overrides don't already exist
	// in provided tags
	for _, tag := range snapshot.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}
	z, err := zone.FromSourceRegionZone(ctx, s, snapshot.Region, snapshot.Volume.Az)
	if err != nil {
		return nil, err
	}
	createDisk := &compute.Disk{
		Name:           fmt.Sprintf(volumeNameFmt, uuid.NewV1().String()),
		SizeGb:         snapshot.Volume.Size,
		Type:           snapshot.Volume.VolumeType,
		Labels:         blockstorage.SanitizeTags(ktags.GetTags(tags)),
		SourceSnapshot: snap.SelfLink,
	}

	resp, err := s.service.Disks.Insert(s.project, z, createDisk).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	if err := s.waitOnOperation(ctx, resp, z); err != nil {
		return nil, err
	}

	return s.VolumeGet(ctx, createDisk.Name, z)
}

func (s *gpdStorage) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
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
			vol, err := s.service.Disks.Get(s.project, res.Az, res.ID).Context(ctx).Do()
			if err != nil {
				return err
			}
			tags = ktags.AddMissingTags(vol.Labels, ktags.GetTags(tags))
			slr := &compute.ZoneSetLabelsRequest{
				LabelFingerprint: vol.LabelFingerprint,
				Labels:           blockstorage.SanitizeTags(tags),
			}
			op, err := s.service.Disks.SetLabels(s.project, res.Az, vol.Name, slr).Do()
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
func (s *gpdStorage) waitOnOperation(ctx context.Context, op *compute.Operation, zone string) error {
	waitBackoff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    1 * time.Second,
		Max:    10 * time.Second,
	}
	for {
		switch op.Status {
		case operationDone:
			if op.Error != nil {
				errJSON, merr := op.Error.MarshalJSON()
				if merr != nil {
					return fmt.Errorf("Operation %s failed. Failed to marshal error string with error %s", op.OperationType, merr)
				}
				return fmt.Errorf("%s", errJSON)
			}
			log.Infof("Operation %s done", op.OperationType)
			return nil
		case operationPending, operationRunning:
			log.Debugf("Operation %s status: %s %s progress %d", op.OperationType, op.Status, op.StatusMessage, op.Progress)
			time.Sleep(waitBackoff.Duration())
			var err error
			if zone != "" {
				op, err = s.service.ZoneOperations.Get(s.project, zone, op.Name).Do()
			} else {
				op, err = s.service.GlobalOperations.Get(s.project, op.Name).Do()
			}
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unknown operation status")
		}
	}
}

// waitOnSnapshotID waits for the snapshot to be created
func (s *gpdStorage) waitOnSnapshotID(ctx context.Context, id string) error {
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
			log.Infof("Snapshot with snapshot_id: %s completed", id)
			return true, nil
		}
		log.Debugf("Snapshot status: snapshot_id: %s, status: %s", id, snap.Status)
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

func (s *gpdStorage) FromRegion(ctx context.Context, region string) ([]string, error) {
	return staticRegionToZones(region)
}

func staticRegionToZones(region string) ([]string, error) {
	switch region {
	case "asia-east1":
		return []string{
			"asia-east1-a",
			"asia-east1-b",
			"asia-east1-c",
		}, nil
	case "asia-east2":
		return []string{
			"asia-east2-a",
			"asia-east2-b",
			"asia-east2-c",
		}, nil
	case "asia-northeast1":
		return []string{
			"asia-northeast1-a",
			"asia-northeast1-b",
			"asia-northeast1-c",
		}, nil
	case "asia-south1":
		return []string{
			"asia-south1-a",
			"asia-south1-b",
			"asia-south1-c",
		}, nil
	case "asia-southeast1":
		return []string{
			"asia-southeast1-a",
			"asia-southeast1-b",
			"asia-southeast1-c",
		}, nil
	case "australia-southeast1":
		return []string{
			"australia-southeast1-a",
			"australia-southeast1-b",
			"australia-southeast1-c",
		}, nil
	case "europe-north1":
		return []string{
			"europe-north1-a",
			"europe-north1-b",
			"europe-north1-c",
		}, nil
	case "europe-west1":
		return []string{
			"europe-west1-b",
			"europe-west1-c",
			"europe-west1-d",
		}, nil
	case "europe-west2":
		return []string{
			"europe-west2-a",
			"europe-west2-b",
			"europe-west2-c",
		}, nil
	case "europe-west3":
		return []string{
			"europe-west3-a",
			"europe-west3-b",
			"europe-west3-c",
		}, nil
	case "europe-west4":
		return []string{
			"europe-west4-a",
			"europe-west4-b",
			"europe-west4-c",
		}, nil
	case "europe-west6":
		return []string{
			"europe-west6-a",
			"europe-west6-b",
			"europe-west6-c",
		}, nil
	case "northamerica-northeast1":
		return []string{
			"northamerica-northeast1-a",
			"northamerica-northeast1-b",
			"northamerica-northeast1-c",
		}, nil
	case "southamerica-east1":
		return []string{
			"southamerica-east1-a",
			"southamerica-east1-b",
			"southamerica-east1-c",
		}, nil
	case "us-central1":
		return []string{
			"us-central1-a",
			"us-central1-b",
			"us-central1-c",
			"us-central1-f",
		}, nil
	case "us-east1":
		return []string{
			"us-east1-b",
			"us-east1-c",
			"us-east1-d",
		}, nil
	case "us-east4":
		return []string{
			"us-east4-a",
			"us-east4-b",
			"us-east4-c",
		}, nil
	case "us-west1":
		return []string{
			"us-west1-a",
			"us-west1-b",
			"us-west1-c",
		}, nil
	case "us-west2":
		return []string{
			"us-west2-a",
			"us-west2-b",
			"us-west2-c",
		}, nil
	}
	return nil, errors.New("cannot get availability zones for region")
}
