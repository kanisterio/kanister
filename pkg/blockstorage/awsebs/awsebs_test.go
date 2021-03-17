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

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type AWSEBSSuite struct{}

var _ = Suite(&AWSEBSSuite{})

func (s AWSEBSSuite) TestQueryRegionToZones(c *C) {
	c.Skip("Only works on AWS")
	ctx := context.Background()
	region := "us-east-1"
	ec2Cli, err := newEC2Client(region, aws.NewConfig().WithCredentials(credentials.NewEnvCredentials()))
	c.Assert(err, IsNil)
	provider := &EbsStorage{Ec2Cli: ec2Cli}
	zs, err := provider.queryRegionToZones(ctx, region)
	c.Assert(err, IsNil)
	c.Assert(zs, DeepEquals, []string{"us-east-1a", "us-east-1b", "us-east-1c", "us-east-1d", "us-east-1e", "us-east-1f"})
}

func (s AWSEBSSuite) TestVolumeParse(c *C) {
	expected := blockstorage.Volume{
		Az:           "the-availability-zone",
		CreationTime: blockstorage.TimeStamp(time.Date(2008, 8, 21, 5, 50, 0, 0, time.UTC)),
		Encrypted:    true,
		ID:           "the-volume-id",
		Iops:         42,
		SizeInBytes:  45097156608, // 42 * 1024 * 1024 * 1024
		Tags: blockstorage.VolumeTags{
			{Key: "a-tag", Value: "a-value"},
			{Key: "another-tag", Value: "another-value"},
		},
		Type:       blockstorage.TypeEBS,
		VolumeType: "the-volume-type",
		Attributes: map[string]string{
			"State": "the-state",
		},
	}

	storage := EbsStorage{}
	volume := storage.volumeParse(context.TODO(), &ec2.Volume{
		AvailabilityZone: aws.String("the-availability-zone"),
		CreateTime:       aws.Time(time.Date(2008, 8, 21, 5, 50, 0, 0, time.UTC)),
		Encrypted:        aws.Bool(true),
		Iops:             aws.Int64(42),
		Size:             aws.Int64(42),
		State:            aws.String("the-state"),
		Tags: []*ec2.Tag{
			{Key: aws.String("a-tag"), Value: aws.String("a-value")},
			{Key: aws.String("another-tag"), Value: aws.String("another-value")},
		},
		VolumeId:   aws.String("the-volume-id"),
		VolumeType: aws.String("the-volume-type"),
	})

	c.Assert(volume, Not(IsNil))
	c.Check(volume.Az, Equals, expected.Az)
	c.Check(volume.CreationTime, Equals, expected.CreationTime)
	c.Check(volume.Encrypted, Equals, expected.Encrypted)
	c.Check(volume.ID, Equals, expected.ID)
	c.Check(volume.Iops, Equals, expected.Iops)
	c.Check(volume.SizeInBytes, Equals, expected.SizeInBytes)
	c.Check(volume.Tags, DeepEquals, expected.Tags)
	c.Check(volume.Type, Equals, blockstorage.TypeEBS)
	c.Check(volume.VolumeType, Equals, expected.VolumeType)
	c.Check(volume.Attributes, DeepEquals, expected.Attributes)
}
