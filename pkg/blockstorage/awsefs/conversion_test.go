// Copyright 2019 Kasten Inc.
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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	. "gopkg.in/check.v1"
)

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
				SizeInBytes:  &awsefs.FileSystemSize{Value: aws.Int64(1000)}, // 1000 bytes
				Encrypted:    aws.Bool(true),
				Tags:         []*awsefs.Tag{},
			},
			expected: &blockstorage.Volume{
				ID:           fsID,
				Az:           az,
				CreationTime: blockstorage.TimeStamp(date),
				Size:         1, // 1000 bytes should be converted to 1 GiB (round-up)
				Type:         blockstorage.TypeEFS,
				Encrypted:    true,
				Tags:         blockstorage.VolumeTags{},
			},
		},
		{
			input: &awsefs.FileSystemDescription{
				FileSystemId: aws.String(fsID),
				CreationTime: aws.Time(date),
				SizeInBytes:  &awsefs.FileSystemSize{Value: aws.Int64(2 * (1 << 30))}, // 2 GiB
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
				Size:         2,
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
		c.Check(vol, DeepEquals, tc.expected)
	}
}
