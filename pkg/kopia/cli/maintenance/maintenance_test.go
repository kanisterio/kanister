package maintenance

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
		Name: "maintenance info with disabled JSON output",
		CLI: func() (safecli.CommandBuilder, error) {
			args := InfoArgs{
				CommonArgs: test.CommonArgs,
				JSONOutput: false,
			}
			return Info(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"maintenance",
			"info",
		},
	},
	{
		Name: "maintenance info with enabled JSON output",
		CLI: func() (safecli.CommandBuilder, error) {
			args := InfoArgs{
				CommonArgs: test.CommonArgs,
				JSONOutput: true,
			}
			return Info(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"maintenance",
			"info",
			"--json",
		},
	},
	{
		Name: "maintenance run with default log-level",
		CLI: func() (safecli.CommandBuilder, error) {
			args := RunArgs{
				CommonArgs: test.CommonArgs,
			}
			return Run(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"maintenance",
			"run",
		},
	},
	{
		Name: "maintenance run with error log-level",
		CLI: func() (safecli.CommandBuilder, error) {
			cmnArgs := test.CommonArgs
			cmnArgs.LogLevel = "error"
			args := RunArgs{
				CommonArgs: cmnArgs,
			}
			return Run(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"maintenance",
			"run",
		},
	},
	{
		Name: "maintenance run with info log-level",
		CLI: func() (safecli.CommandBuilder, error) {
			cmnArgs := test.CommonArgs
			cmnArgs.LogLevel = "info"
			args := RunArgs{
				CommonArgs: cmnArgs,
			}
			return Run(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=info",
			"--log-dir=cache/log",
			"--password=encr-key",
			"maintenance",
			"run",
		},
	},
	{
		Name: "maintenance set owner",
		CLI: func() (safecli.CommandBuilder, error) {
			args := SetOwnerArgs{
				CommonArgs:  test.CommonArgs,
				CustomOwner: "username@hostname",
			}
			return SetOwner(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"maintenance",
			"set",
			"--owner=username@hostname",
		},
	},
}))
