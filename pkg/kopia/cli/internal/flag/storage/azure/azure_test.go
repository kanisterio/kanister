package azure

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/safecli"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestStorageAzure(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "Empty Azure storage flag should generate subcommand with default flags",
		CLI: func() (safecli.CommandBuilder, error) {
			return New(model.StorageFlag{})
		},
		ExpectedCLI: []string{"azure"},
	},
	{
		Name: "Azure with values should generate subcommand with specific flags",
		CLI: func() (safecli.CommandBuilder, error) {
			return New(model.StorageFlag{
				RepoPathPrefix: "repo/path/prefix",
				Location: model.Location{
					rs.PrefixKey: []byte("prefix"),
					rs.BucketKey: []byte("container"),
				},
			})
		},
		ExpectedCLI: []string{
			"azure",
			"--container=container",
			"--prefix=prefix/repo/path/prefix/",
		},
	},
}))
