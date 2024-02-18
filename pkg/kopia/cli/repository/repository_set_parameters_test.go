package repository

import (
	"testing"

	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositorySetParametersCommand(t *testing.T) { check.TestingT(t) }

// Test Repository Set Parameters command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "repository set-parameters with default retention",
		Command: func() (*safecli.Builder, error) {
			args := SetParametersArgs{
				Common: common,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"set-parameters",
		},
	},
	{
		Name: "repository set-parameters with custom retention args",
		Command: func() (*safecli.Builder, error) {
			args := SetParametersArgs{
				Common:          common,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"set-parameters",
			"--retention-mode=Locked",
			"--retention-period=15m0s",
		},
	},
	{
		Name: "repository set-parameters with custom retention mode only",
		Command: func() (*safecli.Builder, error) {
			args := SetParametersArgs{
				Common:        common,
				RetentionMode: retentionMode,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"set-parameters",
			"--retention-mode=Locked",
			"--retention-period=0s",
		},
	},
}))
