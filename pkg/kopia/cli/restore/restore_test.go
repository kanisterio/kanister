package restore

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"gopkg.in/check.v1"
)

func TestCLIRestore(t *testing.T) { check.TestingT(t) }

type CLIRestoreSuite struct{}

var _ = check.Suite(&CLIRestoreSuite{})

func (s *CLIRestoreSuite) TestApply(c *check.C) {
	tests := []struct {
		name   string
		args   RestoreArgs
		expCLI []string
		err    error
	}{
		{
			name: "Empty",
			args: RestoreArgs{},
			err:  cli.ErrInvalidID,
		},
		{
			name: "Empty TargetPath",
			args: RestoreArgs{
				RootID: "snapshot-id",
			},
			err: cli.ErrInvalidTargetPath,
		},
		{
			name: "Restore with no-ignore-permission-errors flag",
			args: RestoreArgs{
				CommonArgs: cli.CommonArgs{
					RepoPassword:   "encr-key",
					ConfigFilePath: "path/kopia.config",
					LogDirectory:   "cache/log",
				},
				RootID:     "snapshot-id",
				TargetPath: "target/path",
			},
			expCLI: []string{
				"kopia",
				"--log-level=error",
				"--config-file=path/kopia.config",
				"--log-dir=cache/log",
				"--password=encr-key",
				"restore",
				"snapshot-id",
				"target/path",
				"--no-ignore-permission-errors",
			},
		},
		{
			name: "Restore with ignore-permission-errors flag",
			args: RestoreArgs{
				CommonArgs: cli.CommonArgs{
					RepoPassword:   "encr-key",
					ConfigFilePath: "path/kopia.config",
					LogDirectory:   "cache/log",
				},
				RootID:                 "snapshot-id",
				TargetPath:             "target/path",
				IgnorePermissionErrors: true,
			},
			expCLI: []string{
				"kopia",
				"--log-level=error",
				"--config-file=path/kopia.config",
				"--log-dir=cache/log",
				"--password=encr-key",
				"restore",
				"snapshot-id",
				"target/path",
				"--ignore-permission-errors",
			},
		},
	}

	for _, tt := range tests {
		b, err := Restore(tt.args)
		cmt := check.Commentf("FAIL: %v", tt.name)
		if tt.err != nil {
			c.Assert(err, check.Equals, tt.err, cmt)
		} else {
			c.Assert(b.Build(), check.DeepEquals, tt.expCLI, cmt)
			c.Assert(err, check.IsNil, cmt)
		}
	}
}
