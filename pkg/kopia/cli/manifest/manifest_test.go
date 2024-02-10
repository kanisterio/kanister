package manifest

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestMaintenanceCommands(t *testing.T) { check.TestingT(t) }

// Test Maintenance commands
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "maintenance list",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ListArgs{
				CommonArgs: test.CommonArgs,
			}
			return List(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"manifest",
			"list",
			"--json",
			"--filter=type:snapshot",
		},
	},
}))
