package awsebs

// AWS EBS Volume storage

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jpillora/backoff"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
)

var _ blockstorage.Provider = (*ebsStorage)(nil)

type ebsStorage struct {
	ec2Cli *EC2
}

// EC2 is kasten's wrapper around ec2.EC2 structs
type EC2 struct {
	*ec2.EC2
	DryRun bool
}

const (
	maxRetries = 10
	// ConfigRegion represents region key required in the map "config"
	ConfigRegion = "region"
	// AccessKeyID represents AWS Access key ID
	AccessKeyID = "AWS_ACCESS_KEY_ID"
	// SecretAccessKey represents AWS Secret Access Key
	SecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

func (s *ebsStorage) Type() blockstorage.Type {
	return blockstorage.TypeEBS
}

// NewProvider returns a provider for the EBS storage type in the specified region
func NewProvider(config map[string]string) (blockstorage.Provider, error) {
	awsConfig, region, err := getConfig(config)
	if err != nil {
		return nil, err
	}
	ec2Cli, err := newEC2Client(region, awsConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get EC2 client")
	}
	return &ebsStorage{ec2Cli: ec2Cli}, nil
}

func getConfig(config map[string]string) (*aws.Config, string, error) {
	region, ok := config[ConfigRegion]
	if !ok {
		return nil, "", errors.New("region required for storage type EBS")
	}
	accessKey, ok := config[AccessKeyID]
	if !ok {
		return nil, "", errors.New("AWS_ACCESS_KEY_ID required for storage type EBS")
	}
	secretAccessKey, ok := config[SecretAccessKey]
	if !ok {
		return nil, "", errors.New("AWS_SECRET_ACCESS_KEY required for storage type EBS")
	}
	return &aws.Config{Credentials: credentials.NewStaticCredentials(accessKey, secretAccessKey, "")}, region, nil
}

// newEC2Client returns ec2 client struct.
func newEC2Client(awsRegion string, config *aws.Config) (*EC2, error) {
	httpClient := &http.Client{Transport: http.DefaultTransport}
	s, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}
	return &EC2{EC2: ec2.New(s, &aws.Config{MaxRetries: aws.Int(maxRetries),
		Region: aws.String(awsRegion), HTTPClient: httpClient})}, nil
}

func (s *ebsStorage) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	cvi := &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(volume.Az),
		VolumeType:       aws.String(string(volume.VolumeType)),
		Encrypted:        aws.Bool(volume.Encrypted),
		Size:             aws.Int64(volume.Size),
	}
	// io1 type *requires* IOPS. Others *cannot* specify them.
	if volume.VolumeType == ec2.VolumeTypeIo1 {
		cvi.Iops = aws.Int64(volume.Iops)
	}

	tags := make(map[string]string, len(volume.Tags))
	for _, tag := range volume.Tags {
		tags[tag.Key] = tag.Value
	}

	volID, err := createVolume(ctx, s.ec2Cli, cvi, ktags.GetTags(tags))
	if err != nil {
		return nil, err
	}

	return s.VolumeGet(ctx, volID, volume.Az)
}

func (s *ebsStorage) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	volIDs := []*string{aws.String(id)}
	dvi := &ec2.DescribeVolumesInput{VolumeIds: volIDs}
	dvo, err := s.ec2Cli.DescribeVolumesWithContext(ctx, dvi)
	if err != nil {
		log.Errorf("Failed to get volumes %v Error: %+v", aws.StringValueSlice(volIDs), err)
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

func (s *ebsStorage) volumeParse(ctx context.Context, volume interface{}) *blockstorage.Volume {
	vol := volume.(*ec2.Volume)
	tags := []*blockstorage.KeyValue(nil)
	for _, tag := range vol.Tags {
		tags = append(tags, &blockstorage.KeyValue{Key: aws.StringValue(tag.Key), Value: aws.StringValue(tag.Value)})
	}
	return &blockstorage.Volume{
		Type:         s.Type(),
		ID:           aws.StringValue(vol.VolumeId),
		Az:           aws.StringValue(vol.AvailabilityZone),
		Encrypted:    aws.BoolValue(vol.Encrypted),
		VolumeType:   aws.StringValue(vol.VolumeType),
		Size:         aws.Int64Value(vol.Size),
		Tags:         tags,
		Iops:         aws.Int64Value(vol.Iops),
		CreationTime: blockstorage.TimeStamp(aws.TimeValue(vol.CreateTime)),
	}
}

func (s *ebsStorage) VolumesList(ctx context.Context, tags map[string]string) ([]*blockstorage.Volume, error) {
	var vols []*blockstorage.Volume
	var fltrs []*ec2.Filter
	dvi := &ec2.DescribeVolumesInput{}
	for k, v := range tags {
		fltr := ec2.Filter{Name: &k, Values: []*string{&v}}
		fltrs = append(fltrs, &fltr)
	}

	dvi.SetFilters(fltrs)
	dvo, err := s.ec2Cli.DescribeVolumesWithContext(ctx, dvi)
	if err != nil {
		return nil, err
	}
	for _, v := range dvo.Volumes {
		vols = append(vols, s.volumeParse(ctx, v))
	}
	return vols, nil
}

func (s *ebsStorage) snapshotParse(ctx context.Context, snap *ec2.Snapshot) *blockstorage.Snapshot {
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
		Size:         aws.Int64Value(snap.VolumeSize),
		Region:       aws.StringValue(s.ec2Cli.Config.Region),
		Volume:       vol,
		CreationTime: blockstorage.TimeStamp(aws.TimeValue(snap.StartTime)),
	}
}

func (s *ebsStorage) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	var snaps []*blockstorage.Snapshot
	var fltrs []*ec2.Filter
	dsi := &ec2.DescribeSnapshotsInput{}
	for k, v := range tags {
		fltr := ec2.Filter{Name: &k, Values: []*string{&v}}
		fltrs = append(fltrs, &fltr)
	}

	dsi.SetFilters(fltrs)
	dso, err := s.ec2Cli.DescribeSnapshotsWithContext(ctx, dsi)
	if err != nil {
		return nil, err
	}
	for _, snap := range dso.Snapshots {
		snaps = append(snaps, s.snapshotParse(ctx, snap))
	}
	return snaps, nil
}

// SnapshotCopy copies snapshot 'from' to 'to'. Follows aws restrictions regarding encryption;

// i.e., copying unencrypted to encrypted snapshot is allowed but not vice versa.
func (s *ebsStorage) SnapshotCopy(ctx context.Context, from, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	if to.Region == "" {
		return nil, errors.New("Destination snapshot AvailabilityZone must be specified")
	}
	if to.ID != "" {
		return nil, errors.Errorf("Snapshot %v destination ID must be empty", to)
	}

	// Copy operation must be initiated from the destination region.
	ec2Cli, err := newEC2Client(to.Region, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get EC2 client")
	}
	// Copy tags from source snap to dest.
	tags := make(map[string]string, len(from.Tags))
	for _, tag := range from.Tags {
		tags[tag.Key] = tag.Value
	}

	csi := ec2.CopySnapshotInput{
		Description:       aws.String("Copy of " + from.ID),
		SourceSnapshotId:  aws.String(from.ID),
		SourceRegion:      aws.String(from.Region),
		DestinationRegion: ec2Cli.Config.Region,
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
	rs.Size = from.Size
	return rs, nil
}

func (s *ebsStorage) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	// Snapshot the EBS volume
	csi := (&ec2.CreateSnapshotInput{}).SetVolumeId(volume.ID)
	var snapID string
	alltags := ktags.GetTags(tags)
	log.Infof("Snapshotting EBS volume: %s", *csi.VolumeId)
	csi.SetDryRun(s.ec2Cli.DryRun)
	snap, err := s.ec2Cli.CreateSnapshotWithContext(ctx, csi)
	if isDryRunErr(err) {
		snapID = ""
	} else {
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create snapshot, volume_id: %s", *csi.VolumeId)
		}
		if err = setResourceTags(ctx, s.ec2Cli, aws.StringValue(snap.SnapshotId), alltags); err != nil {
			return nil, err
		}
		err = waitOnSnapshot(ctx, s.ec2Cli, snap)
		if err != nil {
			return nil, errors.Wrapf(err, "Waiting on snapshot %v", snap)
		}
		snapID = aws.StringValue(snap.SnapshotId)
	}

	snaps, err := getSnapshots(ctx, s.ec2Cli, []*string{&snapID})
	if err != nil {
		return nil, err
	}

	ebssnap := snaps[0]
	region, err := availabilityZoneToRegion(ctx, s.ec2Cli, volume.Az)
	if err != nil {
		return nil, err
	}

	ms := s.snapshotParse(ctx, ebssnap)
	ms.Region = region
	for _, tag := range ebssnap.Tags {
		ms.Tags = append(ms.Tags, &blockstorage.KeyValue{Key: aws.StringValue(tag.Key), Value: aws.StringValue(tag.Value)})
	}
	ms.Volume = &volume
	return ms, nil
}

func (s *ebsStorage) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	log.Infof("EBS Snapshot ID %s", snapshot.ID)
	rmsi := &ec2.DeleteSnapshotInput{}
	rmsi.SetSnapshotId(snapshot.ID)
	rmsi.SetDryRun(s.ec2Cli.DryRun)
	_, err := s.ec2Cli.DeleteSnapshotWithContext(ctx, rmsi)
	if isSnapNotFoundErr(err) {
		// If the snapshot is already deleted, we log, but don't return an error.
		log.Debugf("Snapshot already deleted")
		return nil
	}
	if err != nil && !isDryRunErr(err) {
		return errors.Wrap(err, "Failed to delete snapshot")
	}
	return nil
}

func (s *ebsStorage) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	snaps, err := getSnapshots(ctx, s.ec2Cli, []*string{&id})
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

func (s *ebsStorage) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	rmvi := &ec2.DeleteVolumeInput{}
	rmvi.SetVolumeId(volume.ID)
	rmvi.SetDryRun(s.ec2Cli.DryRun)
	_, err := s.ec2Cli.DeleteVolumeWithContext(ctx, rmvi)
	if isVolNotFoundErr(err) {
		// If the volume is already deleted, we log, but don't return an error.
		log.Debugf("Volume already deleted")
		return nil
	}
	if err != nil && !isDryRunErr(err) {
		return errors.Wrapf(err, "Failed to delete volume volID: %s", volume.ID)
	}
	return nil
}

func (s *ebsStorage) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch res := resource.(type) {
	case *blockstorage.Volume:
		return setResourceTags(ctx, s.ec2Cli, res.ID, tags)
	case *blockstorage.Snapshot:
		return setResourceTags(ctx, s.ec2Cli, res.ID, tags)
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

func (s *ebsStorage) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	if snapshot.Volume == nil {
		return nil, errors.New("Snapshot volume information not available")
	}
	if snapshot.Volume.VolumeType == "" || snapshot.Volume.Az == "" || snapshot.Volume.Tags == nil {
		return nil, errors.Errorf("Required volume fields not available, volumeType: %s, Az: %s, VolumeTags: %v", snapshot.Volume.VolumeType, snapshot.Volume.Az, snapshot.Volume.Tags)
	}

	cvi := &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(snapshot.Volume.Az),
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

	volID, err := createVolume(ctx, s.ec2Cli, cvi, ktags.GetTags(tags))
	if err != nil {
		return nil, err
	}
	return s.VolumeGet(ctx, volID, snapshot.Volume.Az)
}

// createVolume creates an EBS volume using the specified parameters
func createVolume(ctx context.Context, ec2Cli *EC2, cvi *ec2.CreateVolumeInput, tags map[string]string) (string, error) {
	// Set tags
	awsTags := mapToEC2Tags(tags)
	ts := []*ec2.TagSpecification{&ec2.TagSpecification{ResourceType: aws.String(ec2.ResourceTypeVolume), Tags: awsTags}}
	cvi.SetTagSpecifications(ts)
	cvi.SetDryRun(ec2Cli.DryRun)
	vol, err := ec2Cli.CreateVolumeWithContext(ctx, cvi)
	if isDryRunErr(err) {
		return "", nil
	}
	if err != nil {
		log.Errorf("Failed to create volume for %v Error: %+v", *cvi, err)
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
		return nil, errors.Wrapf(err, "Failed to get snapshot, snapshot_ids: %p", snapIDs)
	}
	// TODO: handle paging and continuation
	if len(dso.Snapshots) != len(snapIDs) {
		log.Errorf("Did not find all requested snapshots, snapshots_requested: %p, snapshots_found: %p", snapIDs, dso.Snapshots)
		// TODO: Move mapping to HTTP error to the caller
		return nil, errors.New("Object not found")
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
			log.Errorf("Failed to describe volume %s Error: %+v", aws.StringValue(vol.VolumeId), err)
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
			log.Infof("Volume %s complete", *vol.VolumeId)
			return nil
		}
		log.Infof("Volume %s state: %s", *vol.VolumeId, *s.State)
		time.Sleep(volWaitBackoff.Duration())
	}
}

// waitOnSnapshot waits for the snapshot to be created
func waitOnSnapshot(ctx context.Context, ec2Cli *EC2, snap *ec2.Snapshot) error {
	return waitOnSnapshotID(ctx, ec2Cli, *snap.SnapshotId)
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
			return false, errors.New("Object not found")
		}
		s := dso.Snapshots[0]
		if *s.State == ec2.SnapshotStateError {
			return false, errors.New("Snapshot EBS volume failed")
		}
		if *s.State == ec2.SnapshotStateCompleted {
			log.Infof("Snapshot with snapshot_id: %s completed", snapID)
			return true, nil
		}
		log.Debugf("Snapshot progress: snapshot_id: %s, progress: %s", snapID, fmt.Sprintf("%+v", *s.Progress))
		return false, nil
	})
}

// GetRegionFromEC2Metadata retrieves the region from the EC2 metadata service.
// Only works when the call is performed from inside AWS
func GetRegionFromEC2Metadata() (string, error) {
	log.Debug("Retrieving region from metadata")
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
