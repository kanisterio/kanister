package repository

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositoryConnectCommand(t *testing.T) { check.TestingT(t) }

// Test Repository Connect command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "repository connect with default retention",
		Command: func() (*safecli.Builder, error) {
			args := ConnectArgs{
				Common:         common,
				Cache:          cache,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
				Location:       locFS,
			}
			return Connect(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"connect",
			"--no-check-for-updates",
			"--cache-directory=/tmp/cache.dir",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
			"--override-hostname=test-hostname",
			"--override-username=test-username",
			"filesystem",
			"--path=/mnt/data/test-prefix/test-path/prefix/",
		},
	},
	{
		Name: "repository connect with PIT and ReadOnly",
		Command: func() (*safecli.Builder, error) {
			pit, _ := strfmt.ParseDateTime("2021-02-03T01:02:03Z")
			args := ConnectArgs{
				Common:         common,
				Cache:          cache,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
				Location:       locFS,
				PointInTime:    pit,
				ReadOnly:       true,
			}
			return Connect(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"connect",
			"--no-check-for-updates",
			"--readonly",
			"--cache-directory=/tmp/cache.dir",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
			"--override-hostname=test-hostname",
			"--override-username=test-username",
			"filesystem",
			"--path=/mnt/data/test-prefix/test-path/prefix/",
			"--point-in-time=2021-02-03T01:02:03.000Z",
		},
	},
}))
