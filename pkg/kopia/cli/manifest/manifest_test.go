package manifest

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"github.com/kanisterio/kanister/pkg/safecli"
	"gopkg.in/check.v1"
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
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"manifest",
			"list",
			"--json",
			"--filter=type:snapshot",
		},
	},
}))
