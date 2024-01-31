package storage

import (
	"fmt"
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/fs"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
	cmdlog "github.com/kanisterio/kanister/pkg/kopia/cli/internal/log"
	"github.com/kanisterio/kanister/pkg/safecli"
	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/pkg/errors"
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

func (s *StorageSuite) TestStorageFlag(c *check.C) {
	tests := []struct {
		name    string
		storage flag.Applier
		expCLI  []string
		err     error
		errMsg  string
	}{
		{
			name:    "Empty Storage should generate an error",
			storage: Storage(nil, ""),
			//errMsg:  "Failed to generate storage args: unsupported type for the location: ",
			err: cli.ErrUnsupportedStorage,
		},
		{
			name: "Filesystem without prefix and with repo path should generate repo path",
			storage: Storage(
				model.Location{
					rs.TypeKey:   []byte("filestore"),
					rs.PrefixKey: []byte(""),
				},
				"dir1/subdir/",
			),
			expCLI: []string{
				"filesystem",
				fmt.Sprintf("--path=%s/dir1/subdir/", fs.DefaultFSMountPath),
			},
		},
		{
			name: "Filesystem with prefix and repo path should generate merged prefix and repo path",
			storage: Storage(
				model.Location{
					rs.TypeKey:   []byte("filestore"),
					rs.PrefixKey: []byte("test-prefix"),
				},
				"dir1/subdir/",
			),
			expCLI: []string{
				"filesystem",
				fmt.Sprintf("--path=%s/test-prefix/dir1/subdir/", fs.DefaultFSMountPath),
			},
		},
		{
			name: "S3 should generate s3 args",
			storage: Storage(
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
			expCLI: []string{
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
			name: "S3 with no prefix should use onlu repo path prefix",
			storage: Storage(
				model.Location{
					rs.TypeKey:          []byte("s3"),
					rs.EndpointKey:      []byte("http://endpoint.com"), // disable TLS
					rs.RegionKey:        []byte("us-east-1"),
					rs.BucketKey:        []byte("bucket"),
					rs.SkipSSLVerifyKey: []byte("true"),
				},
				"prefixfs",
			),
			expCLI: []string{
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
			name: "S3 with no endpoint should omit endpoint flag",
			storage: Storage(
				model.Location{
					rs.TypeKey:   []byte("s3"),
					rs.PrefixKey: []byte("/path/to/prefix"),
					rs.RegionKey: []byte("us-east-1"),
					rs.BucketKey: []byte("bucket"),
				},
				"prefixfs",
			),
			expCLI: []string{
				"s3",
				"--region=us-east-1",
				"--bucket=bucket",
				"--prefix=/path/to/prefix/prefixfs/",
			},
		},
		{
			name: "S3 endpoint with trailing slashes should be trimmed",
			storage: Storage(
				model.Location{
					rs.TypeKey:     []byte("s3"),
					rs.EndpointKey: []byte("https://endpoint.com//////"), // slashes will be trimmed
					rs.PrefixKey:   []byte("/path/to/prefix"),
					rs.RegionKey:   []byte("us-east-1"),
					rs.BucketKey:   []byte("bucket"),
				},
				"prefixfs",
			),
			expCLI: []string{
				"s3",
				"--region=us-east-1",
				"--bucket=bucket",
				"--endpoint=endpoint.com",
				"--prefix=/path/to/prefix/prefixfs/",
			},
		},
		{
			name: "GCS should generate gcs args",
			storage: Storage(
				model.Location{
					rs.TypeKey:   []byte("gcs"),
					rs.BucketKey: []byte("bucket"),
					rs.PrefixKey: []byte("/path/to/prefix"),
				},
				"prefixfs",
			),
			expCLI: []string{
				"gcs",
				"--bucket=bucket",
				"--credentials-file=/tmp/creds.txt",
				"--prefix=/path/to/prefix/prefixfs/",
			},
		},
		{
			name: "Azure should generate azure args",
			storage: Storage(
				model.Location{
					rs.TypeKey:   []byte("azure"),
					rs.BucketKey: []byte("bucket"),
					rs.PrefixKey: []byte("/path/to/prefix"),
				},
				"prefixfs",
			),
			expCLI: []string{
				"azure",
				"--container=bucket",
				"--prefix=/path/to/prefix/prefixfs/",
			},
		},
		{
			name: "Unsupported storage type should generate an error",
			storage: Storage(
				model.Location{
					rs.TypeKey: []byte("ftp"),
				},
				"prefixfs",
			),
			errMsg: "failed to apply storage args: unsupported location type: 'ftp': unsupported storage",
			err:    cli.ErrUnsupportedStorage,
		},
	}

	for _, tt := range tests {
		b := safecli.NewBuilder()
		err := tt.storage.Apply(b)

		cmt := check.Commentf("FAIL: %v", tt.name)
		if tt.errMsg != "" {
			c.Assert(err.Error(), check.Equals, tt.errMsg, cmt)
		}

		if tt.err == nil {
			c.Assert(err, check.IsNil, cmt)
		} else {
			if errors.Cause(err) != nil {
				c.Assert(errors.Cause(err), check.DeepEquals, tt.err, cmt)
			} else {
				c.Assert(err, check.Equals, tt.err, cmt)
			}
		}
		c.Assert(b.Build(), check.DeepEquals, tt.expCLI, cmt)
	}
}

type MockFlagWithError struct{}

func (f MockFlagWithError) Flag() string {
	return "mock"
}

var ErrMock = fmt.Errorf("mock error")

func (f MockFlagWithError) Apply(cli safecli.CommandAppender) error {
	return ErrMock
}

func (s *StorageSuite) TestNewStorageBuilderWithErrorFlag(c *check.C) {
	b, err := command.NewCommandBuilder(command.Azure, MockFlagWithError{})
	c.Assert(b, check.IsNil)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.DeepEquals, ErrMock)
}

func (s *StorageSuite) TestStorageGetLogger(c *check.C) {
	storage := Storage(nil, "prefix")
	c.Assert(storage.GetLogger(), check.NotNil)

	nopLog := &cmdlog.NopLogger{}
	storage = Storage(nil, "prefix", WithLogger(nopLog))
	c.Assert(storage.GetLogger(), check.Equals, nopLog)
}

type MockFactory struct{}

func (f MockFactory) Create(locType rs.LocType) model.StorageBuilder {
	return func(s model.StorageFlag) (*safecli.Builder, error) {
		return safecli.NewBuilder("mock"), nil
	}
}

func (s *StorageSuite) TestStorageFactory(c *check.C) {
	storage := Storage(nil, "prefix")
	c.Assert(storage.GetLogger(), check.NotNil)

	mockFactory := &MockFactory{}
	storage = Storage(nil, "prefix", WithFactory(mockFactory))
	c.Assert(storage.Factory, check.Equals, mockFactory)
}
