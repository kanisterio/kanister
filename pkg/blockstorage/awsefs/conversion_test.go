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
