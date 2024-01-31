package s3

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/safecli"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestStorageS3(t *testing.T) { check.TestingT(t) }

var logger = &test.StringLogger{}

var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "Empty S3 storage flag should generate subcommand with default flags",
		CLI: func() (safecli.CommandBuilder, error) {
			return New(model.StorageFlag{})
		},
		ExpectedCLI: []string{
			"s3",
		},
	},
	{
		Name: "S3 with values should generate subcommand with specific flags",
		CLI: func() (safecli.CommandBuilder, error) {
			return New(model.StorageFlag{
				RepoPathPrefix: "repo/path/prefix",
				Location: model.Location{
					rs.PrefixKey:        []byte("prefix"),
					rs.EndpointKey:      []byte("http://endpoint/path/"),
					rs.RegionKey:        []byte("region"),
					rs.BucketKey:        []byte("bucket"),
					rs.SkipSSLVerifyKey: []byte("true"),
				},
				Logger: logger,
			})
		},
		ExpectedCLI: []string{
			"s3",
			"--region=region",
			"--bucket=bucket",
			"--endpoint=endpoint/path",
			"--prefix=prefix/repo/path/prefix/",
			"--disable-tls",
			"--disable-tls-verification",
		},

		Logger: logger,
		LoggerRegex: []string{
			"Removing leading",
			"Removing trailing",
		},
	},
}))
