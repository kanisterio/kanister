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

package awsebs

// AWS EBS Volume storage

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"

	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	"github.com/kanisterio/kanister/pkg/blockstorage/zone"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

var _ blockstorage.Provider = (*EbsStorage)(nil)
var _ zone.Mapper = (*EbsStorage)(nil)

// EbsStorage implements blockstorage.Provider
type EbsStorage struct {
	Ec2Cli *EC2
	Role   string
	config *aws.Config
}

// EC2 is kasten's wrapper around ec2.EC2 structs
type EC2 struct {
	*ec2.EC2
	DryRun bool
}

const (
	maxRetries = 10
)

// Type is part of blockstorage.Provider
func (s *EbsStorage) Type() blockstorage.Type {
	return blockstorage.TypeEBS
}

// NewProvider returns a provider for the EBS storage type in the specified region
func NewProvider(ctx context.Context, config map[string]string) (blockstorage.Provider, error) {
	awsConfig, region, err := awsconfig.GetConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	ec2Cli, err := newEC2Client(region, awsConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get EC2 client")
	}
	return &EbsStorage{Ec2Cli: ec2Cli, Role: config[awsconfig.ConfigRole], config: awsConfig}, nil
}

// newEC2Client returns ec2 client struct.
func newEC2Client(awsRegion string, config *aws.Config) (*EC2, error) {
	if config == nil {
		return nil, errors.New("Invalid empty AWS config")
	}
	s, err := session.NewSession(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session for EBS")
	}
	conf := config.WithMaxRetries(maxRetries).WithRegion(awsRegion).WithCredentials(config.Credentials)
	return &EC2{EC2: ec2.New(s, conf)}, nil
}

// VolumeCreate is part of blockstorage.Provider
func (s *EbsStorage) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	cvi := &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(volume.Az),
		VolumeType:       aws.String(string(volume.VolumeType)),
		Encrypted:        aws.Bool(volume.Encrypted),
		Size:             aws.Int64(blockstorage.SizeInGi(volume.SizeInBytes)),
	}
	// io1 type *requires* IOPS. Others *cannot* specify them.
	if volume.VolumeType == ec2.VolumeTypeIo1 {
		cvi.Iops = aws.Int64(volume.Iops)
	}

	tags := make(map[string]string, len(volume.Tags))
	for _, tag := range volume.Tags {
		tags[tag.Key] = tag.Value
	}

	volID, err := createVolume(ctx, s.Ec2Cli, cvi, ktags.GetTags(tags))
	if err != nil {
		return nil, err
	}

	return s.VolumeGet(ctx, volID, volume.Az)
}

// CheckVolumeCreate checks if client as permission to create volumes
func (s *EbsStorage) CheckVolumeCreate(ctx context.Context) (bool, error) {
	var zoneName *string
	var err error
	var size int64 = 1
	var dryRun bool = true

	ec2Cli, err := newEC2Client(*s.config.Region, s.config)
	if err != nil {
		return false, errors.Wrap(err, "Could not get EC2 client")
	}
	dai := &ec2.DescribeAvailabilityZonesInput{}
	az, err := ec2Cli.DescribeAvailabilityZones(dai)
	if err != nil {
		return false, errors.New("Fail to get available zone for EC2 client")
	}
	if az != nil {
		zoneName = az.AvailabilityZones[1].ZoneName
	} else {
		return false, errors.New("No available zone for EC2 client")
	}

	cvi := &ec2.CreateVolumeInput{
		AvailabilityZone: zoneName,
		Size:             &size,
		DryRun:           &dryRun,
	}
	_, err = s.Ec2Cli.CreateVolume(cvi)
	if !isDryRunErr(err) {
		return false, errors.Wrap(err, "Could not create volume with EC2 client")
	}
	return true, nil
}

// VolumeGet is part of blockstorage.Provider
func (s *EbsStorage) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	volIDs := []*string{aws.String(id)}
	dvi := &ec2.DescribeVolumesInput{VolumeIds: volIDs}
	dvo, err := s.Ec2Cli.DescribeVolumesWithContext(ctx, dvi)
	if err != nil {
		log.WithError(err).Print("Failed to get volumes", field.M{"VolumeIds": volIDs})
		return nil, err
	}
	if len(dvo.Volumes) != len(volIDs) {
		return nil, errors.New("Object not found")
	}
	vols := dvo.Volumes
	if len(vols) == 0 {
		return nil, errors.New("Volume with volume_id: " + id + " not found")
	}
	if len(vols) > 1 {
		return nil, errors.Errorf("Found an unexpected number of volumes: volume_id=%s result_count=%d", id, len(vols))
	}
	vol := vols[0]
	mv := s.volumeParse(ctx, vol)
	return mv, nil
}

func (s *EbsStorage) volumeParse(ctx context.Context, volume interface{}) *blockstorage.Volume {
	vol := volume.(*ec2.Volume)
	tags := []*blockstorage.KeyValue(nil)
	for _, tag := range vol.Tags {
		tags = append(tags, &blockstorage.KeyValue{Key: aws.StringValue(tag.Key), Value: aws.StringValue(tag.Value)})
	}
	var attrs map[string]string
	if vol.State != nil {
		attrs = map[string]string{
			"State": *vol.State,
		}
	}
	return &blockstorage.Volume{
		Type:         s.Type(),
		ID:           aws.StringValue(vol.VolumeId),
		Az:           aws.StringValue(vol.AvailabilityZone),
		Encrypted:    aws.BoolValue(vol.Encrypted),
		VolumeType:   aws.StringValue(vol.VolumeType),
		SizeInBytes:  aws.Int64Value(vol.Size) * blockstorage.BytesInGi,
		Attributes:   attrs,
		Tags:         tags,
		Iops:         aws.Int64Value(vol.Iops),
		CreationTime: blockstorage.TimeStamp(aws.TimeValue(vol.CreateTime)),
	}
}

// VolumesList is part of blockstorage.Provider
func (s *EbsStorage) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	var fltrs []*ec2.Filter
	dvi := &ec2.DescribeVolumesInput{}
	for k, v := range tags {
		fltr := ec2.Filter{Name: aws.String("tag:" + k), Values: []*string{&v}}
		fltrs = append(fltrs, &fltr)
	}

	dvi.SetFilters(fltrs)
	dvo, err := s.Ec2Cli.DescribeVolumesWithContext(ctx, dvi)
	if err != nil {
		return nil, err
	}
	for _, v := range dvo.Volumes {
		vols = append(vols, s.volumeParse(ctx, v))
	}
	return vols, nil
}

func (s *EbsStorage) snapshotParse(ctx context.Context, snap *ec2.Snapshot) *blockstorage.Snapshot {
	tags := []*blockstorage.KeyValue(nil)
	for _, tag := range snap.Tags {
		tags = append(tags, &blockstorage.KeyValue{Key: *tag.Key, Value: *tag.Value})
	}
	vol := &blockstorage.Volume{
		Type: s.Type(),
		ID:   aws.StringValue(snap.VolumeId),
	}
	// TODO: fix getting region from zone
	return &blockstorage.Snapshot{
		ID:           aws.StringValue(snap.SnapshotId),
		Tags:         tags,
		Type:         s.Type(),
		Encrypted:    aws.BoolValue(snap.Encrypted),
		SizeInBytes:  aws.Int64Value(snap.VolumeSize) * blockstorage.BytesInGi,
		Region:       aws.StringValue(s.Ec2Cli.Config.Region),
		Volume:       vol,
		CreationTime: blockstorage.TimeStamp(aws.TimeValue(snap.StartTime)),
	}
}

// SnapshotsList is part of blockstorage.Provider
func (s *EbsStorage) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	var fltrs []*ec2.Filter
	dsi := &ec2.DescribeSnapshotsInput{}
	for k, v := range tags {
		fltr := ec2.Filter{Name: aws.String("tag:" + k), Values: []*string{&v}}
		fltrs = append(fltrs, &fltr)
	}

	dsi.SetFilters(fltrs)
	dso, err := s.Ec2Cli.DescribeSnapshotsWithContext(ctx, dsi)
	if err != nil {
		return nil, err
	}
	for _, snap := range dso.Snapshots {
		snaps = append(snaps, s.snapshotParse(ctx, snap))
	}
	return snaps, nil
}

// SnapshotCopy is part of blockstorage.Provider
// SnapshotCopy copies snapshot 'from' to 'to'. Follows aws restrictions regarding encryption;
// i.e., copying unencrypted to encrypted snapshot is allowed but not vice versa.
func (s *EbsStorage) SnapshotCopy(ctx context.Context, from, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	if to.Region == "" {
		return nil, errors.New("Destination snapshot AvailabilityZone must be specified")
	}
	if to.ID != "" {
		return nil, errors.Errorf("Snapshot %v destination ID must be empty", to)
	}
	// Copy operation must be initiated from the destination region.
	ec2Cli, err := newEC2Client(to.Region, s.Ec2Cli.Config.Copy())
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get EC2 client")
	}
	// Include a presigned URL when the regions are different. Include it
	// independent of whether or not the snapshot is encrypted.
	var presignedURL *string
	if to.Region != from.Region {
		si := ec2.CopySnapshotInput{
			SourceSnapshotId:  aws.String(from.ID),
			SourceRegion:      aws.String(from.Region),
			DestinationRegion: ec2Cli.Config.Region,
		}
		rq, _ := ec2Cli.CopySnapshotRequest(&si)
		su, err2 := rq.Presign(120 * time.Minute)
		if err2 != nil {
			return nil, errors.Wrap(err2, "Could not presign URL for snapshot copy request")
		}
		presignedURL = &su
	}
	// Copy tags from source snap to dest.
	tags := make(map[string]string, len(from.Tags))
	for _, tag := range from.Tags {
		tags[tag.Key] = tag.Value
	}

	var encrypted *bool
	// encrypted can not be set to false.
	// Only unspecified or `true` are supported in `CopySnapshotInput`
	if from.Encrypted {
		encrypted = &from.Encrypted
	}
	csi := ec2.CopySnapshotInput{
		Description:       aws.String("Copy of " + from.ID),
		SourceSnapshotId:  aws.String(from.ID),
		SourceRegion:      aws.String(from.Region),
		DestinationRegion: ec2Cli.Config.Region,
		Encrypted:         encrypted,
		PresignedUrl:      presignedURL,
	}
	cso, err := ec2Cli.CopySnapshotWithContext(ctx, &csi)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to copy snapshot %v", csi)
	}
	snapID := aws.StringValue(cso.SnapshotId)
	if err = setResourceTags(ctx, ec2Cli, snapID, ktags.GetTags(tags)); err != nil {
		return nil, err
	}
	if err = waitOnSnapshotID(ctx, ec2Cli, snapID); err != nil {
		return nil, errors.Wrapf(err, "Snapshot %s did not complete", snapID)
	}
	snaps, err := getSnapshots(ctx, ec2Cli, []*string{aws.String(snapID)})
	if err != nil {
		return nil, err
	}

	// aws: Snapshots created by the CopySnapshot action have an arbitrary volume ID
	//      that should not be used for any purpose.
	rs := s.snapshotParse(ctx, snaps[0])
	*rs.Volume = *from.Volume
	rs.Region = to.Region
	rs.SizeInBytes = from.SizeInBytes
	return rs, nil
}

// SnapshotCopyWithArgs is part of blockstorage.Provider
func (s *EbsStorage) SnapshotCopyWithArgs(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot, args map[string]string) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Copy Snapshot with Args not implemented")
}

// SnapshotCreate is part of blockstorage.Provider
func (s *EbsStorage) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	// Snapshot the EBS volume
	csi := (&ec2.CreateSnapshotInput{}).SetVolumeId(volume.ID)
	csi.SetTagSpecifications([]*ec2.TagSpecification{
		{
			ResourceType: aws.String(ec2.ResourceTypeSnapshot),
			Tags:         mapToEC2Tags(ktags.GetTags(tags)),
		},
	})
	log.Print("Snapshotting EBS volume", field.M{"volume_id": *csi.VolumeId})
	csi.SetDryRun(s.Ec2Cli.DryRun)
	snap, err := s.Ec2Cli.CreateSnapshotWithContext(ctx, csi)
	if err != nil && !isDryRunErr(err) {
		return nil, errors.Wrapf(err, "Failed to create snapshot, volume_id: %s", *csi.VolumeId)
	}

	region, err := availabilityZoneToRegion(ctx, s.Ec2Cli, volume.Az)
	if err != nil {
		return nil, err
	}

	ms := s.snapshotParse(ctx, snap)
	ms.Region = region
	for _, tag := range snap.Tags {
		ms.Tags = append(ms.Tags, &blockstorage.KeyValue{Key: aws.StringValue(tag.Key), Value: aws.StringValue(tag.Value)})
	}
	ms.Volume = &volume
	return ms, nil
}

// SnapshotCreateWaitForCompletion is part of blockstorage.Provider
func (s *EbsStorage) SnapshotCreateWaitForCompletion(ctx context.Context, snap *blockstorage.Snapshot) error {
	if s.Ec2Cli.DryRun {
		return nil
	}
	if err := waitOnSnapshotID(ctx, s.Ec2Cli, snap.ID); err != nil {
		return errors.Wrapf(err, "Waiting on snapshot %v", snap)
	}
	return nil
}

// SnapshotDelete is part of blockstorage.Provider
func (s *EbsStorage) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	log.Print("Deleting EBS Snapshot", field.M{"SnapshotID": snapshot.ID})
	rmsi := &ec2.DeleteSnapshotInput{}
	rmsi.SetSnapshotId(snapshot.ID)
	rmsi.SetDryRun(s.Ec2Cli.DryRun)
	_, err := s.Ec2Cli.DeleteSnapshotWithContext(ctx, rmsi)
	if isSnapNotFoundErr(err) {
		// If the snapshot is already deleted, we log, but don't return an error.
		log.Debug().Print("Snapshot already deleted")
		return nil
	}
	if err != nil && !isDryRunErr(err) {
		return errors.Wrap(err, "Failed to delete snapshot")
	}
	return nil
}

// SnapshotGet is part of blockstorage.Provider
func (s *EbsStorage) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	snaps, err := getSnapshots(ctx, s.Ec2Cli, []*string{&id})
	if err != nil {
		return nil, err
	}
	snap := snaps[0]
	ms := s.snapshotParse(ctx, snap)
	for _, tag := range snap.Tags {
		ms.Tags = append(ms.Tags, &blockstorage.KeyValue{Key: aws.StringValue(tag.Key), Value: aws.StringValue(tag.Value)})
	}

	return ms, nil
}

// VolumeDelete is part of blockstorage.Provider
func (s *EbsStorage) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	rmvi := &ec2.DeleteVolumeInput{}
	rmvi.SetVolumeId(volume.ID)
	rmvi.SetDryRun(s.Ec2Cli.DryRun)
	_, err := s.Ec2Cli.DeleteVolumeWithContext(ctx, rmvi)
	if isVolNotFoundErr(err) {
		// If the volume is already deleted, we log, but don't return an error.
		log.Debug().Print("Volume already deleted")
		return nil
	}
	if err != nil && !isDryRunErr(err) {
		return errors.Wrapf(err, "Failed to delete volume volID: %s", volume.ID)
	}
	return nil
}

// SetTags is part of blockstorage.Provider
func (s *EbsStorage) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch res := resource.(type) {
	case *blockstorage.Volume:
		return setResourceTags(ctx, s.Ec2Cli, res.ID, tags)
	case *blockstorage.Snapshot:
		return setResourceTags(ctx, s.Ec2Cli, res.ID, tags)
	default:
		return errors.Wrapf(nil, "Unknown resource type: %v", res)
	}
}

// setResourceTags sets tags on the specified resource
func setResourceTags(ctx context.Context, ec2Cli *EC2, resourceID string, tags map[string]string) error {
	cti := &ec2.CreateTagsInput{Resources: []*string{&resourceID}, Tags: mapToEC2Tags(tags)}
	if _, err := ec2Cli.CreateTags(cti); err != nil {
		return errors.Wrapf(err, "Failed to set tags, resource_id:%s", resourceID)
	}
	return nil
}

// VolumeCreateFromSnapshot is part of blockstorage.Provider
func (s *EbsStorage) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	if snapshot.Volume == nil {
		return nil, errors.New("Snapshot volume information not available")
	}
	if snapshot.Volume.VolumeType == "" || snapshot.Volume.Az == "" || snapshot.Volume.Tags == nil {
		return nil, errors.Errorf("Required volume fields not available, volumeType: %s, Az: %s, VolumeTags: %v", snapshot.Volume.VolumeType, snapshot.Volume.Az, snapshot.Volume.Tags)
	}
	kubeCli, err := kube.NewClient()
	if err != nil {
		log.WithError(err).Print("Failed to initialize kubernetes client")
	}
	zones, err := zone.FromSourceRegionZone(ctx, s, kubeCli, snapshot.Region, snapshot.Volume.Az)
	if err != nil {
		return nil, err
	}
	if len(zones) != 1 {
		return nil, errors.Errorf("Length of zone slice should be 1, got %d", len(zones))
	}
	cvi := &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(zones[0]),
		SnapshotId:       aws.String(snapshot.ID),
		VolumeType:       aws.String(string(snapshot.Volume.VolumeType)),
	}
	// io1 type *requires* IOPS. Others *cannot* specify them.
	if snapshot.Volume.VolumeType == ec2.VolumeTypeIo1 {
		cvi.Iops = aws.Int64(snapshot.Volume.Iops)
	}
	// Incorporate pre-existing tags.
	for _, tag := range snapshot.Volume.Tags {
		if _, found := tags[tag.Key]; !found {
			tags[tag.Key] = tag.Value
		}
	}

	volID, err := createVolume(ctx, s.Ec2Cli, cvi, ktags.GetTags(tags))
	if err != nil {
		if isVolNotFoundErr(err) {
			return nil, errors.Wrap(err, "This may indicate insufficient permissions for KMS keys.")
		}
		return nil, err
	}
	return s.VolumeGet(ctx, volID, snapshot.Volume.Az)
}

// createVolume creates an EBS volume using the specified parameters
func createVolume(ctx context.Context, ec2Cli *EC2, cvi *ec2.CreateVolumeInput, tags map[string]string) (string, error) {
	// Set tags
	awsTags := mapToEC2Tags(tags)
	ts := []*ec2.TagSpecification{{ResourceType: aws.String(ec2.ResourceTypeVolume), Tags: awsTags}}
	cvi.SetTagSpecifications(ts)
	cvi.SetDryRun(ec2Cli.DryRun)
	vol, err := ec2Cli.CreateVolumeWithContext(ctx, cvi)
	if isDryRunErr(err) {
		return "", nil
	}
	if err != nil {
		log.WithError(err).Print("Failed to create volume", field.M{"input": cvi})
		return "", err
	}

	err = waitOnVolume(ctx, ec2Cli, vol)
	if err != nil {
		return "", err
	}
	return aws.StringValue(vol.VolumeId), nil
}

// getSnapshots returns the snapshot metadata for the specified snapshot ids
func getSnapshots(ctx context.Context, ec2Cli *EC2, snapIDs []*string) ([]*ec2.Snapshot, error) {
	dsi := &ec2.DescribeSnapshotsInput{SnapshotIds: snapIDs}
	dso, err := ec2Cli.DescribeSnapshotsWithContext(ctx, dsi)
	if err != nil {
		return nil, errors.Wrapf(err, blockstorage.SnapshotDoesNotExistError+", snapshot_ids: %p", snapIDs)
	}
	// TODO: handle paging and continuation
	if len(dso.Snapshots) != len(snapIDs) {
		log.Error().Print("Did not find all requested snapshots", field.M{"snapshots_requested": snapIDs, "snapshots_found": dso.Snapshots})
		// TODO: Move mapping to HTTP error to the caller
		return nil, errors.New(blockstorage.SnapshotDoesNotExistError)
	}
	return dso.Snapshots, nil
}

// availabilityZoneToRegion converts from Az to Region
func availabilityZoneToRegion(ctx context.Context, awsCli *EC2, az string) (ar string, err error) {
	azi := &ec2.DescribeAvailabilityZonesInput{
		ZoneNames: []*string{&az},
	}

	azo, err := awsCli.DescribeAvailabilityZonesWithContext(ctx, azi)
	if err != nil {
		return "", errors.Wrapf(err, "Could not determine region for availability zone (AZ) %s", az)
	}

	if len(azo.AvailabilityZones) == 0 {
		return "", errors.New("Region unavailable for availability zone" + az)
	}

	return aws.StringValue(azo.AvailabilityZones[0].RegionName), nil
}

func mapToEC2Tags(tags map[string]string) []*ec2.Tag {
	// Set tags
	awsTags := make([]*ec2.Tag, 0, len(tags))
	for k, v := range tags {
		awsTags = append(awsTags, &ec2.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return awsTags
}

// waitOnVolume waits for the volume to be created
func waitOnVolume(ctx context.Context, ec2Cli *EC2, vol *ec2.Volume) error {
	volWaitBackoff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    10 * time.Millisecond,
		Max:    10 * time.Second,
	}
	dvi := &ec2.DescribeVolumesInput{}
	dvi = dvi.SetVolumeIds([]*string{vol.VolumeId})
	for {
		dvo, err := ec2Cli.DescribeVolumesWithContext(ctx, dvi)
		if err != nil {
			log.WithError(err).Print("Failed to describe volume", field.M{"VolumeID": aws.StringValue(vol.VolumeId)})
			return err
		}
		if len(dvo.Volumes) != 1 {
			return errors.New("Object not found")
		}
		s := dvo.Volumes[0]
		if *s.State == ec2.VolumeStateError {
			return errors.New("Creating EBS volume failed")
		}
		if *s.State == ec2.VolumeStateAvailable {
			log.Print("Volume creation complete", field.M{"VolumeID": *vol.VolumeId})
			return nil
		}
		log.Print("Update", field.M{"Volume": *vol.VolumeId, "State": *s.State})
		time.Sleep(volWaitBackoff.Duration())
	}
}

func waitOnSnapshotID(ctx context.Context, ec2Cli *EC2, snapID string) error {
	snapWaitBackoff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    1 * time.Second,
		Max:    10 * time.Second,
	}
	dsi := &ec2.DescribeSnapshotsInput{}
	dsi = dsi.SetSnapshotIds([]*string{&snapID})
	return poll.WaitWithBackoff(ctx, snapWaitBackoff, func(ctx context.Context) (bool, error) {
		dso, err := ec2Cli.DescribeSnapshotsWithContext(ctx, dsi)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to describe snapshot, snapshot_id: %s", snapID)
		}
		if len(dso.Snapshots) != 1 {
			return false, errors.New(blockstorage.SnapshotDoesNotExistError)
		}
		s := dso.Snapshots[0]
		if *s.State == ec2.SnapshotStateError {
			return false, errors.New("Snapshot EBS volume failed")
		}
		if *s.State == ec2.SnapshotStateCompleted {
			log.Print("Snapshot completed", field.M{"SnapshotID": snapID})
			return true, nil
		}
		log.Debug().Print("Snapshot progress", field.M{"snapshot_id": snapID, "progress": *s.Progress})
		return false, nil
	})
}

// GetRegionFromEC2Metadata retrieves the region from the EC2 metadata service.
// Only works when the call is performed from inside AWS
func GetRegionFromEC2Metadata() (string, error) {
	log.Debug().Print("Retrieving region from metadata")
	conf := aws.Config{
		HTTPClient: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   2 * time.Second,
		},
		MaxRetries: aws.Int(1),
	}
	ec2MetaData := ec2metadata.New(session.Must(session.NewSession()), &conf)

	awsRegion, err := ec2MetaData.Region()
	return awsRegion, errors.Wrap(err, "Failed to get AWS Region")
}

// FromRegion is part of zone.Mapper
func (s *EbsStorage) FromRegion(ctx context.Context, region string) ([]string, error) {
	ec2Cli, err := newEC2Client(region, s.config)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get EC2 client while fetching zones FromRegion (%s)", region)
	}
	trueBool := true
	filterKey := "region-name"
	zones, err := ec2Cli.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{
		AllAvailabilityZones: &trueBool,
		Filters: []*ec2.Filter{
			{Name: &filterKey, Values: []*string{&region}},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get availability zones for region %s", region)
	}
	zoneList := []string{}
	for _, zone := range zones.AvailabilityZones {
		zoneList = append(zoneList, *zone.ZoneName)
	}
	return zoneList, nil
}

func (s *EbsStorage) GetRegions(ctx context.Context) ([]string, error) {
	trueBool := true
	result, err := s.Ec2Cli.DescribeRegions(&ec2.DescribeRegionsInput{AllRegions: &trueBool})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to describe regions")
	}
	regions := []string{}

	for _, region := range result.Regions {
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}

// SnapshotRestoreTargets is part of blockstorage.RestoreTargeter
func (s *EbsStorage) SnapshotRestoreTargets(ctx context.Context, snapshot *blockstorage.Snapshot) (global bool, regionsAndZones map[string][]string, err error) {
	// A few checks from VolumeCreateFromSnapshot
	if snapshot.Volume == nil {
		return false, nil, errors.New("Snapshot volume information not available")
	}
	if snapshot.Volume.VolumeType == "" || snapshot.Volume.Az == "" || snapshot.Volume.Tags == nil {
		return false, nil, errors.Errorf("Required volume fields not available, volumeType: %s, Az: %s, VolumeTags: %v", snapshot.Volume.VolumeType, snapshot.Volume.Az, snapshot.Volume.Tags)
	}
	// EBS snapshots can only be restored in their region
	zl, err := s.FromRegion(ctx, snapshot.Region)
	if err != nil {
		return false, nil, err
	}
	return false, map[string][]string{snapshot.Region: zl}, nil
}
