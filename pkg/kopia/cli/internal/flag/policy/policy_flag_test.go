package policy

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestPolicy(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Global(false) should not generate a flag",
		Flag: Global(false),
	},
	{
		Name:        "Global(true) should generate a flag",
		Flag:        Global(true),
		ExpectedCLI: []string{"--global"},
	},
	{
		Name: "Empty BackupRetentionPolicy should not generate a flag",
		Flag: BackupRetentionPolicy{},
	},
	{
		Name: "BackupRetentionPolicy with values should generate multiple flags with given values",
		Flag: BackupRetentionPolicy{
			KeepLatest: KeepLatest(1),
			KeepHourly: KeepHourly(2),
			KeepDaily:  KeepDaily(3),

			KeepWeekly:  KeepWeekly(4),
			KeepMonthly: KeepMonthly(5),
			KeepAnnual:  KeepAnnual(6),
		},
		ExpectedCLI: []string{
			"--keep-latest=1",
			"--keep-hourly=2",
			"--keep-daily=3",
			"--keep-weekly=4",
			"--keep-monthly=5",
			"--keep-annual=6",
		},
	},
}))
