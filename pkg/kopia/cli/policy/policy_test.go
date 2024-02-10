package policy

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestPolicyCommands(t *testing.T) { check.TestingT(t) }

// Test Policy Set command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "PolicySet with default args",
		CLI: func() (safecli.CommandBuilder, error) {
			args := SetArgs{
				CommonArgs: test.CommonArgs,
			}
			return Set(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"policy",
			"set",
			"--global",
			"--keep-latest=2147483647",
			"--keep-hourly=0",
			"--keep-daily=0",
			"--keep-weekly=0",
			"--keep-monthly=0",
			"--keep-annual=0",
			"--compression=s2-default",
		},
	},
	{
		Name: "PolicySet with custom args",
		CLI: func() (safecli.CommandBuilder, error) {
			retentionPolicy := NewRetentionPolicyArgs(
				WithKeepLatest(1),
				WithKeepHourly(2),
				WithKeepDaily(3),
				WithKeepWeekly(4),
				WithKeepMonthly(5),
				WithKeepAnnual(6),
			)
			compressionPolicy := NewCompressionPolicyArgs(
				WithCompressionAlgorithm("zip"),
			)
			args := SetArgs{
				CommonArgs:            test.CommonArgs,
				RetentionPolicyArgs:   retentionPolicy,
				CompressionPolicyArgs: compressionPolicy,
			}
			return Set(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"policy",
			"set",
			"--global",
			"--keep-latest=1",
			"--keep-hourly=2",
			"--keep-daily=3",
			"--keep-weekly=4",
			"--keep-monthly=5",
			"--keep-annual=6",
			"--compression=zip",
		},
	},
}))

// Test Policy Show command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "PolicyShow with default args",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ShowArgs{
				CommonArgs: test.CommonArgs,
			}
			return Show(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"policy",
			"show",
			"--global",
		},
	},
	{
		Name: "PolicyShow with JSON output",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ShowArgs{
				CommonArgs: test.CommonArgs,
				JSONOutput: true,
			}
			return Show(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"policy",
			"show",
			"--global",
			"--json",
		},
	},
}))
