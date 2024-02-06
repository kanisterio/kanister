// Copyright 2024 The Kanister Authors.
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

package storage

import (
	"fmt"
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/fs"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
	cmdlog "github.com/kanisterio/kanister/pkg/kopia/cli/internal/log"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestStorageFlags(t *testing.T) { check.TestingT(t) }

type StorageSuite struct{}

var _ = check.Suite(&StorageSuite{})

func (s *StorageSuite) TestLocationMethods(c *check.C) {
	type expected struct {
		Type             rs.LocType
		Region           string
		BucketName       string
		Endpoint         string
		Prefix           string
		IsInsecure       bool
		HasSkipSSLVerify bool
	}

	tests := []struct {
		name     string
		location model.Location
		expected expected
	}{
		{
			name: "Test1",
			location: model.Location{
				rs.TypeKey:          []byte("Type1"),
				rs.RegionKey:        []byte("Region1"),
				rs.BucketKey:        []byte("Bucket1"),
				rs.EndpointKey:      []byte("http://Endpoint1"),
				rs.PrefixKey:        []byte("Prefix1"),
				rs.SkipSSLVerifyKey: []byte("true"),
			},
			expected: expected{
				Type:             "Type1",
				Region:           "Region1",
				BucketName:       "Bucket1",
				Endpoint:         "http://Endpoint1",
				Prefix:           "Prefix1",
				IsInsecure:       true,
				HasSkipSSLVerify: true,
			},
		},
		{
			name: "Test2",
			location: model.Location{
				rs.TypeKey:          []byte("Type2"),
				rs.RegionKey:        []byte("Region2"),
				rs.BucketKey:        []byte("Bucket2"),
				rs.EndpointKey:      []byte("https://Endpoint2"),
				rs.PrefixKey:        []byte("Prefix2"),
				rs.SkipSSLVerifyKey: []byte("false"),
			},
			expected: expected{
				Type:             "Type2",
				Region:           "Region2",
				BucketName:       "Bucket2",
				Endpoint:         "https://Endpoint2",
				Prefix:           "Prefix2",
				IsInsecure:       false,
				HasSkipSSLVerify: false,
			},
		},
	}

	for _, tt := range tests {
		c.Assert(tt.location.Type(), check.Equals, tt.expected.Type)
		c.Assert(tt.location.Region(), check.Equals, tt.expected.Region)
		c.Assert(tt.location.BucketName(), check.Equals, tt.expected.BucketName)
		c.Assert(tt.location.Endpoint(), check.Equals, tt.expected.Endpoint)
		c.Assert(tt.location.Prefix(), check.Equals, tt.expected.Prefix)
		c.Assert(tt.location.IsInsecureEndpoint(), check.Equals, tt.expected.IsInsecure)
		c.Assert(tt.location.HasSkipSSLVerify(), check.Equals, tt.expected.HasSkipSSLVerify)
	}
}

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name:        "Empty Storage should generate an error",
		Flag:        Storage(nil, ""),
		ExpectedErr: cli.ErrUnsupportedStorage,
	},
	{
		Name: "Filesystem without prefix and with repo path should generate repo path",
		Flag: Storage(
			model.Location{
				rs.TypeKey:   []byte("filestore"),
				rs.PrefixKey: []byte(""),
			},
			"dir1/subdir/",
		),
		ExpectedCLI: []string{
			"filesystem",
			fmt.Sprintf("--path=%s/dir1/subdir/", fs.DefaultFSMountPath),
		},
	},
	{
		Name: "Filesystem with prefix and repo path should generate merged prefix and repo path",
		Flag: Storage(
			model.Location{
				rs.TypeKey:   []byte("filestore"),
				rs.PrefixKey: []byte("test-prefix"),
			},
			"dir1/subdir/",
		),
		ExpectedCLI: []string{
			"filesystem",
			fmt.Sprintf("--path=%s/test-prefix/dir1/subdir/", fs.DefaultFSMountPath),
		},
	},
	{
		Name: "Unsupported storage type should generate an error",
		Flag: Storage(
			model.Location{
				rs.TypeKey: []byte("ftp"),
			},
			"prefixfs",
		),
		ExpectedErr:    cli.ErrUnsupportedStorage,
		ExpectedErrMsg: "failed to apply storage args: unsupported location type: 'ftp': unsupported storage",
	},
	{
		Name: "GCS should generate gcs args",
		Flag: Storage(
			model.Location{
				rs.TypeKey:   []byte("gcs"),
				rs.BucketKey: []byte("bucket"),
				rs.PrefixKey: []byte("/path/to/prefix"),
			},
			"prefixfs",
		),
		ExpectedCLI: []string{
			"gcs",
			"--bucket=bucket",
			"--credentials-file=/tmp/creds.txt",
			"--prefix=/path/to/prefix/prefixfs/",
		},
	},
	{
		Name: "Azure should generate azure args",
		Flag: Storage(
			model.Location{
				rs.TypeKey:   []byte("azure"),
				rs.BucketKey: []byte("bucket"),
				rs.PrefixKey: []byte("/path/to/prefix"),
			},
			"prefixfs",
		),
		ExpectedCLI: []string{
			"azure",
			"--container=bucket",
			"--prefix=/path/to/prefix/prefixfs/",
		},
	},
	{
		Name: "S3 should generate s3 args",
		Flag: Storage(
			model.Location{
				rs.TypeKey:          []byte("s3"),
				rs.EndpointKey:      []byte("http://endpoint.com"), // disable TLS
				rs.PrefixKey:        []byte("/path/to/prefix"),
				rs.RegionKey:        []byte("us-east-1"),
				rs.BucketKey:        []byte("bucket"),
				rs.SkipSSLVerifyKey: []byte("true"),
			},
			"prefixfs",
		),
		ExpectedCLI: []string{
			"s3",
			"--region=us-east-1",
			"--bucket=bucket",
			"--endpoint=endpoint.com",
			"--prefix=/path/to/prefix/prefixfs/",
			"--disable-tls",
			"--disable-tls-verification",
		},
	},
	{
		Name: "S3 with no prefix should use only repo path prefix",
		Flag: Storage(
			model.Location{
				rs.TypeKey:          []byte("s3"),
				rs.EndpointKey:      []byte("http://endpoint.com"), // disable TLS
				rs.RegionKey:        []byte("us-east-1"),
				rs.BucketKey:        []byte("bucket"),
				rs.SkipSSLVerifyKey: []byte("true"),
			},
			"prefixfs",
		),
		ExpectedCLI: []string{
			"s3",
			"--region=us-east-1",
			"--bucket=bucket",
			"--endpoint=endpoint.com",
			"--prefix=prefixfs",
			"--disable-tls",
			"--disable-tls-verification",
		},
	},
	{
		Name: "S3 with no endpoint should omit endpoint flag",
		Flag: Storage(
			model.Location{
				rs.TypeKey:   []byte("s3"),
				rs.PrefixKey: []byte("/path/to/prefix"),
				rs.RegionKey: []byte("us-east-1"),
				rs.BucketKey: []byte("bucket"),
			},
			"prefixfs",
		),
		ExpectedCLI: []string{
			"s3",
			"--region=us-east-1",
			"--bucket=bucket",
			"--prefix=/path/to/prefix/prefixfs/",
		},
	},
	{
		Name: "S3 endpoint with trailing slashes should be trimmed",
		Flag: Storage(
			model.Location{
				rs.TypeKey:     []byte("s3"),
				rs.EndpointKey: []byte("https://endpoint.com//////"), // slashes will be trimmed
				rs.PrefixKey:   []byte("/path/to/prefix"),
				rs.RegionKey:   []byte("us-east-1"),
				rs.BucketKey:   []byte("bucket"),
			},
			"prefixfs",
		),
		ExpectedCLI: []string{
			"s3",
			"--region=us-east-1",
			"--bucket=bucket",
			"--endpoint=endpoint.com",
			"--prefix=/path/to/prefix/prefixfs/",
		},
	},
	{
		Name: "S3 compliant should generate s3 args",
		Flag: Storage(
			model.Location{
				rs.TypeKey:          []byte("s3Compliant"),
				rs.EndpointKey:      []byte("http://endpoint.com"), // disable TLS
				rs.PrefixKey:        []byte("/path/to/prefix"),
				rs.RegionKey:        []byte("us-east-1"),
				rs.BucketKey:        []byte("bucket"),
				rs.SkipSSLVerifyKey: []byte("true"),
			},
			"prefixfs",
		),
		ExpectedCLI: []string{
			"s3",
			"--region=us-east-1",
			"--bucket=bucket",
			"--endpoint=endpoint.com",
			"--prefix=/path/to/prefix/prefixfs/",
			"--disable-tls",
			"--disable-tls-verification",
		},
	},
}))

// MockFlagWithError is a mock flag that always returns an error
type MockFlagWithError struct{}

var errMock = fmt.Errorf("mock error")

func (f MockFlagWithError) Apply(cli safecli.CommandAppender) error {
	return errMock
}

func (s *StorageSuite) TestNewStorageBuilderWithErrorFlag(c *check.C) {
	b, err := command.NewCommandBuilder(command.FileSystem, MockFlagWithError{})
	c.Assert(b, check.IsNil)
	c.Assert(err, check.Equals, errMock)
}

func (s *StorageSuite) TestStorageGetLogger(c *check.C) {
	storage := Storage(nil, "prefix")
	c.Assert(storage.GetLogger(), check.NotNil)

	nopLog := &cmdlog.NopLogger{}
	storage = Storage(nil, "prefix", WithLogger(nopLog))
	c.Assert(storage.GetLogger(), check.Equals, nopLog)
}

// MockFactory is a mock storage factory
type MockFactory struct{}

func (f MockFactory) Create(locType rs.LocType) model.StorageBuilder {
	return func(s model.StorageFlag) (*safecli.Builder, error) {
		return safecli.NewBuilder("mockfactory"), nil
	}
}

func (s *StorageSuite) TestStorageFactory(c *check.C) {
	storage := Storage(nil, "prefix")
	c.Assert(storage.GetLogger(), check.NotNil)

	mockFactory := &MockFactory{}
	storage = Storage(nil, "prefix", WithFactory(mockFactory))
	c.Assert(storage.Factory, check.Equals, mockFactory)
	b, err := storage.Factory.Create("anything")(model.StorageFlag{})
	c.Assert(b, check.NotNil)
	c.Assert(err, check.IsNil)
	c.Assert(b.Build(), check.DeepEquals, []string{"mockfactory"})
}
