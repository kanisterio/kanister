package blob

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"github.com/kanisterio/kanister/pkg/safecli"
)

func TestBlobCommands(t *testing.T) { check.TestingT(t) }

// Test Blob commands
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "blob list with default args",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ListArgs{
				CommonArgs: test.CommonArgs,
			}
			return List(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"blob",
			"list",
		},
	},
	{
		Name: "blob stats with default args",
		CLI: func() (safecli.CommandBuilder, error) {
			args := StatsArgs{
				CommonArgs: test.CommonArgs,
			}
			return Stats(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"blob",
			"stats",
			"--raw",
		},
	},
}))
