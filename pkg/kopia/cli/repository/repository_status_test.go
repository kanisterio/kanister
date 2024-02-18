package repository

import (
	"testing"

	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositoryStatusCommand(t *testing.T) { check.TestingT(t) }

// Test Repository Status command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "repository status with default args",
		Command: func() (*safecli.Builder, error) {
			args := StatusArgs{
				Common: common,
			}
			return Status(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"status",
		},
	},
	{
		Name: "repository status with JSON output",
		Command: func() (*safecli.Builder, error) {
			args := StatusArgs{
				Common:     common,
				JSONOutput: true,
			}
			return Status(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"status",
			"--json",
		},
	},
}))
