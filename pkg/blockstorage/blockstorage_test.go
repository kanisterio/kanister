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

package blockstorage_test

import (
	"context"
	"os"
	"strings"
	"testing"

	. "gopkg.in/check.v1"

	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	envconfig "github.com/kanisterio/kanister/pkg/config"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube/volume"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	clusterRegionAWS = "us-west-2"
	testTagKey       = "kanister.io/testid"
	testTagValue     = "unittest"
)

func Test(t *testing.T) { TestingT(t) }

type BlockStorageProviderSuite struct {
	storageType   blockstorage.Type
	storageRegion string
	storageAZ     string
	provider      blockstorage.Provider
	volumes       []*blockstorage.Volume
	snapshots     []*blockstorage.Snapshot
	args          map[string]string
}

var _ = Suite(&BlockStorageProviderSuite{storageType: blockstorage.TypeEBS, storageRegion: clusterRegionAWS, storageAZ: "us-west-2b"})
var _ = Suite(&BlockStorageProviderSuite{storageType: blockstorage.TypeGPD, storageRegion: "", storageAZ: "us-west1-b"})
var _ = Suite(&BlockStorageProviderSuite{storageType: blockstorage.TypeGPD, storageRegion: "", storageAZ: "us-west1-c__us-west1-a"})
var _ = Suite(&BlockStorageProviderSuite{storageType: blockstorage.TypeAD, storageRegion: "", storageAZ: "eastus2-1"})

func (s *BlockStorageProviderSuite) SetUpSuite(c *C) {
	var err error
	s.args = make(map[string]string)
	config := s.getConfig(c, s.storageRegion)
	s.provider, err = getter.New().Get(s.storageType, config)
	c.Assert(err, IsNil)
}

func (s *BlockStorageProviderSuite) TearDownTest(c *C) {
	for _, snapshot := range s.snapshots {
		c.Assert(s.provider.SnapshotDelete(context.Background(), snapshot), IsNil)
	}
	s.snapshots = nil

	for _, volume := range s.volumes {
		c.Assert(s.provider.VolumeDelete(context.Background(), volume), IsNil)
	}
	s.volumes = nil
}

func (s *BlockStorageProviderSuite) TestCreateVolume(c *C) {
	vol := s.createVolume(c)
	// Check setting tags on the volume
	tags := map[string]string{"testtag": "testtagvalue"}
	err := s.provider.SetTags(context.Background(), vol, tags)
	c.Assert(err, IsNil)
	volUpdated, err := s.provider.VolumeGet(context.Background(), vol.ID, vol.Az)
	c.Assert(err, IsNil)
	// Check previously set tags still exist
	s.checkTagsExist(c, blockstorage.KeyValueToMap(volUpdated.Tags), blockstorage.KeyValueToMap(vol.Tags))
	// Check new tags were set
	s.checkTagsExist(c, blockstorage.KeyValueToMap(volUpdated.Tags), tags)
	// Check std tags
	s.checkStdTagsExist(c, blockstorage.KeyValueToMap(volUpdated.Tags))

	// Test VolumesList
	s.testVolumesList(c)

	err = s.provider.VolumeDelete(context.Background(), volUpdated)
	c.Assert(err, IsNil)
	// We ensure that multiple deletions are handled.
	err = s.provider.VolumeDelete(context.Background(), volUpdated)
	c.Assert(err, IsNil)
	s.volumes = nil
}

func (s *BlockStorageProviderSuite) TestCreateSnapshot(c *C) {
	snapshot := s.createSnapshot(c)
	// Check setting tags on the snapshot
	tags := map[string]string{"testtag": "testtagvalue"}
	err := s.provider.SetTags(context.Background(), snapshot, tags)
	c.Assert(err, IsNil)
	snap, err := s.provider.SnapshotGet(context.Background(), snapshot.ID)
	c.Assert(err, IsNil)
	// Check previously set tags still exist
	s.checkTagsExist(c, blockstorage.KeyValueToMap(snap.Tags), blockstorage.KeyValueToMap(snapshot.Tags))
	// Check new tags were set
	s.checkTagsExist(c, blockstorage.KeyValueToMap(snap.Tags), tags)
	// Check std tags exist
	s.checkStdTagsExist(c, blockstorage.KeyValueToMap(snap.Tags))

	snapshotGet, err := s.provider.SnapshotGet(context.Background(), snapshot.ID)
	c.Assert(err, IsNil)
	c.Assert(snapshotGet.ID, Equals, snapshot.ID)

	if s.provider.Type() != blockstorage.TypeAD {
		// Also test creating a volume from this snapshot
		tags = map[string]string{testTagKey: testTagValue, "kanister.io/testname": c.TestName()}
		vol, err := s.provider.VolumeCreateFromSnapshot(context.Background(), *snapshot, tags)
		c.Assert(err, IsNil)
		s.volumes = append(s.volumes, vol)
		for _, tag := range snapshot.Volume.Tags {
			if _, found := tags[tag.Key]; !found {
				tags[tag.Key] = tag.Value
			}
		}
		// Check tags were merged
		s.checkTagsExist(c, blockstorage.KeyValueToMap(vol.Tags), tags)
		s.checkStdTagsExist(c, blockstorage.KeyValueToMap(vol.Tags))

		err = s.provider.SnapshotDelete(context.Background(), snapshot)
		c.Assert(err, IsNil)
		// We ensure that multiple deletions are handled.
		err = s.provider.SnapshotDelete(context.Background(), snapshot)
		c.Assert(err, IsNil)
		s.snapshots = nil
		_, err = s.provider.SnapshotGet(context.Background(), snapshot.ID)
		c.Assert(err, NotNil)
		c.Assert(strings.Contains(err.Error(), blockstorage.SnapshotDoesNotExistError), Equals, true)
	}
}

func (s *BlockStorageProviderSuite) TestSnapshotCopy(c *C) {
	if s.storageType == blockstorage.TypeGPD {
		c.Skip("Skip snapshot copy test for GPD provider since the SnapshotCopy is yet to be implemented for GPD ")
	}
	var snap *blockstorage.Snapshot
	var err error
	srcSnapshot := s.createSnapshot(c)
	var dstSnapshot *blockstorage.Snapshot
	switch s.storageType {
	case blockstorage.TypeEBS:
		dstSnapshot = &blockstorage.Snapshot{
			Type:        srcSnapshot.Type,
			Encrypted:   false,
			SizeInBytes: srcSnapshot.SizeInBytes,
			Region:      "us-east-1",
			Volume:      nil,
		}
	case blockstorage.TypeAD:
		dstSnapshot = &blockstorage.Snapshot{
			Type:        srcSnapshot.Type,
			Encrypted:   false,
			SizeInBytes: srcSnapshot.SizeInBytes,
			Region:      "westus2",
			Volume:      nil,
		}
		snap, err = s.provider.SnapshotCopyWithArgs(context.TODO(), *srcSnapshot, *dstSnapshot, s.args)
		c.Assert(err, IsNil)
	}

	if s.storageType != blockstorage.TypeAD {
		snap, err = s.provider.SnapshotCopy(context.TODO(), *srcSnapshot, *dstSnapshot)
		c.Assert(err, IsNil)
	}

	log.Print("Snapshot copied", field.M{"FromSnapshotID": srcSnapshot.ID, "ToSnapshotID": snap.ID})

	config := s.getConfig(c, dstSnapshot.Region)
	provider, err := getter.New().Get(s.storageType, config)
	c.Assert(err, IsNil)

	snapDetails, err := provider.SnapshotGet(context.TODO(), snap.ID)
	c.Assert(err, IsNil)

	c.Check(snapDetails.Region, Equals, dstSnapshot.Region)
	c.Check(snapDetails.SizeInBytes, Equals, srcSnapshot.SizeInBytes)

	err = provider.SnapshotDelete(context.TODO(), snap)
	c.Assert(err, IsNil)
	err = provider.SnapshotDelete(context.TODO(), srcSnapshot)
	c.Assert(err, IsNil)
}

func (s *BlockStorageProviderSuite) testVolumesList(c *C) {
	var zone string
	tags := map[string]string{"testtag": "testtagvalue"}
	zone = s.storageAZ
	vols, err := s.provider.VolumesList(context.Background(), tags, zone)
	c.Assert(err, IsNil)
	c.Assert(vols, NotNil)
	c.Assert(vols, FitsTypeOf, []*blockstorage.Volume{})
	c.Assert(vols, Not(HasLen), 0)
	c.Assert(vols[0].Type, Equals, s.provider.Type())
}

func (s *BlockStorageProviderSuite) TestSnapshotsList(c *C) {
	var tags map[string]string
	testSnaphot := s.createSnapshot(c)
	tags = map[string]string{testTagKey: testTagValue}
	snaps, err := s.provider.SnapshotsList(context.Background(), tags)
	c.Assert(err, IsNil)
	c.Assert(snaps, NotNil)
	c.Assert(snaps, FitsTypeOf, []*blockstorage.Snapshot{})
	c.Assert(snaps, Not(HasLen), 0)
	c.Assert(snaps[0].Type, Equals, s.provider.Type())
	_ = s.provider.SnapshotDelete(context.Background(), testSnaphot)
}

// Helpers
func (s *BlockStorageProviderSuite) createVolume(c *C) *blockstorage.Volume {
	tags := []*blockstorage.KeyValue{
		{Key: testTagKey, Value: testTagValue},
		{Key: "kanister.io/testname", Value: c.TestName()},
	}
	vol := blockstorage.Volume{
		SizeInBytes: 1 * blockstorage.BytesInGi,
		Tags:        tags,
	}
	size := vol.SizeInBytes

	vol.Az = s.storageAZ
	if s.isRegional(vol.Az) {
		vol.SizeInBytes = 200 * blockstorage.BytesInGi
		size = vol.SizeInBytes
	}

	ret, err := s.provider.VolumeCreate(context.Background(), vol)
	c.Assert(err, IsNil)
	s.volumes = append(s.volumes, ret)
	c.Assert(ret.SizeInBytes, Equals, int64(size))
	s.checkTagsExist(c, blockstorage.KeyValueToMap(ret.Tags), blockstorage.KeyValueToMap(tags))
	s.checkStdTagsExist(c, blockstorage.KeyValueToMap(ret.Tags))
	return ret
}

func (s *BlockStorageProviderSuite) createSnapshot(c *C) *blockstorage.Snapshot {
	vol := s.createVolume(c)
	tags := map[string]string{testTagKey: testTagValue, "kanister.io/testname": c.TestName()}
	ret, err := s.provider.SnapshotCreate(context.Background(), *vol, tags)
	c.Assert(err, IsNil)
	s.snapshots = append(s.snapshots, ret)
	s.checkTagsExist(c, blockstorage.KeyValueToMap(ret.Tags), tags)
	c.Assert(s.provider.SnapshotCreateWaitForCompletion(context.Background(), ret), IsNil)
	c.Assert(ret.Volume, NotNil)
	return ret
}

func (s *BlockStorageProviderSuite) checkTagsExist(c *C, actual map[string]string, expected map[string]string) {
	if s.provider.Type() != blockstorage.TypeEBS {
		expected = blockstorage.SanitizeTags(expected)
	}

	for k, v := range expected {
		c.Check(actual[k], Equals, v)
	}
}

func (s *BlockStorageProviderSuite) checkStdTagsExist(c *C, actual map[string]string) {
	stdTags := ktags.GetStdTags()
	for k := range stdTags {
		c.Check(actual[k], NotNil)
	}
}

func (s *BlockStorageProviderSuite) getConfig(c *C, region string) map[string]string {
	config := make(map[string]string)
	switch s.storageType {
	case blockstorage.TypeEBS:
		config[awsconfig.ConfigRegion] = region
		accessKey := envconfig.GetEnvOrSkip(c, awsconfig.AccessKeyID)
		secretAccessKey := envconfig.GetEnvOrSkip(c, awsconfig.SecretAccessKey)
		config[awsconfig.AccessKeyID] = accessKey
		config[awsconfig.SecretAccessKey] = secretAccessKey
		config[awsconfig.ConfigRole] = os.Getenv(awsconfig.ConfigRole)
	case blockstorage.TypeGPD:
		creds := envconfig.GetEnvOrSkip(c, blockstorage.GoogleCloudCreds)
		config[blockstorage.GoogleCloudCreds] = creds
	case blockstorage.TypeAD:
		config[blockstorage.AzureSubscriptionID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureSubscriptionID)
		config[blockstorage.AzureTenantID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureTenantID)
		config[blockstorage.AzureClientID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureClientID)
		config[blockstorage.AzureClientSecret] = envconfig.GetEnvOrSkip(c, blockstorage.AzureClientSecret)
		config[blockstorage.AzureResurceGroup] = envconfig.GetEnvOrSkip(c, blockstorage.AzureResurceGroup)
		config[blockstorage.AzureCloudEnvironmentID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureCloudEnvironmentID)
		s.args[blockstorage.AzureMigrateStorageAccount] = envconfig.GetEnvOrSkip(c, blockstorage.AzureMigrateStorageAccount)
		s.args[blockstorage.AzureMigrateStorageKey] = envconfig.GetEnvOrSkip(c, blockstorage.AzureMigrateStorageKey)
		s.args[blockstorage.AzureMigrateResourceGroup] = envconfig.GetEnvOrSkip(c, blockstorage.AzureMigrateResourceGroup)
	default:
		c.Errorf("Unknown blockstorage storage type %s", s.storageType)
	}
	return config
}

func (b *BlockStorageProviderSuite) isRegional(az string) bool {
	return strings.Contains(az, volume.RegionZoneSeparator)
}

func (b *BlockStorageProviderSuite) TestFilterSnasphotWithTags(c *C) {
	snapshot1 := &blockstorage.Snapshot{ID: "snap1", Tags: blockstorage.SnapshotTags{
		{Key: "key1", Value: "val1"},
		{Key: "key3", Value: ""},
	}}
	snapshot2 := &blockstorage.Snapshot{ID: "snap2", Tags: blockstorage.SnapshotTags{
		{Key: "key2", Value: "val2"},
	}}

	filterTags := map[string]string{"key1": "val1"}
	snaps := blockstorage.FilterSnapshotsWithTags([]*blockstorage.Snapshot{snapshot1, snapshot2}, filterTags)
	c.Assert(len(snaps), Equals, 1)

	snaps = blockstorage.FilterSnapshotsWithTags([]*blockstorage.Snapshot{snapshot1, snapshot2}, nil)
	c.Assert(len(snaps), Equals, 2)

	snaps = blockstorage.FilterSnapshotsWithTags([]*blockstorage.Snapshot{snapshot1, snapshot2}, map[string]string{})
	c.Assert(len(snaps), Equals, 2)

	snaps = blockstorage.FilterSnapshotsWithTags([]*blockstorage.Snapshot{snapshot1, snapshot2}, map[string]string{"bad": "tag"})
	c.Assert(len(snaps), Equals, 0)
}
