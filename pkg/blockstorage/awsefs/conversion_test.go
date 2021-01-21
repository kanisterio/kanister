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

package awsefs

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type AWSEFSConversionTestSuite struct{}

var _ = Suite(&AWSEFSConversionTestSuite{})

func (s *AWSEFSConversionTestSuite) TestVolumeConversion(c *C) {
	az := "us-west-2a"
	fsID := "fs-123456"
	date := time.Date(2018, 10, 1, 1, 1, 1, 1, time.UTC)

	tcs := []struct {
		input    *awsefs.FileSystemDescription
		expected *blockstorage.Volume
	}{
		{
			input: &awsefs.FileSystemDescription{
				FileSystemId: aws.String(fsID),
				CreationTime: aws.Time(date),
				SizeInBytes:  &awsefs.FileSystemSize{Value: aws.Int64(1024)},
				Encrypted:    aws.Bool(true),
				Tags:         []*awsefs.Tag{},
			},
			expected: &blockstorage.Volume{
				ID:           fsID,
				Az:           az,
				CreationTime: blockstorage.TimeStamp(date),
				SizeInBytes:  1024,
				Type:         blockstorage.TypeEFS,
				Encrypted:    true,
				Tags:         blockstorage.VolumeTags{},
			},
		},
		{
			input: &awsefs.FileSystemDescription{
				FileSystemId: aws.String(fsID),
				CreationTime: aws.Time(date),
				SizeInBytes:  &awsefs.FileSystemSize{Value: aws.Int64(2048)},
				Encrypted:    aws.Bool(false),
				Tags: []*awsefs.Tag{
					&awsefs.Tag{Key: aws.String("key1"), Value: aws.String("value1")},
					&awsefs.Tag{Key: aws.String("key2"), Value: aws.String("value2")},
				},
			},
			expected: &blockstorage.Volume{
				ID:           fsID,
				Az:           az,
				CreationTime: blockstorage.TimeStamp(date),
				SizeInBytes:  2048,
				Type:         blockstorage.TypeEFS,
				Encrypted:    false,
				Tags: blockstorage.VolumeTags(
					[]*blockstorage.KeyValue{
						&blockstorage.KeyValue{Key: "key1", Value: "value1"},
						&blockstorage.KeyValue{Key: "key2", Value: "value2"},
					},
				),
			},
		},
	}

	for _, tc := range tcs {
		vol := volumeFromEFSDescription(tc.input, az)
		c.Check(vol.Az, Equals, tc.expected.Az)
		c.Check(vol.ID, Equals, tc.expected.ID)
		c.Check(vol.CreationTime, Equals, tc.expected.CreationTime)
		c.Check(vol.SizeInBytes, Equals, tc.expected.SizeInBytes)
		c.Check(vol.Type, Equals, tc.expected.Type)
		c.Check(vol.Encrypted, Equals, tc.expected.Encrypted)
		c.Check(vol.Tags, HasLen, len(tc.expected.Tags))
	}
}
