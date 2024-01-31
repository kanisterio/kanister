package policy

import (
	"strconv"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// keepFlag creates a new flag for the number of backups to keep.
func keepFlag(flagName string, numOfBackups int) flag.Applier {
	return flag.NewStringFlag(flagName, strconv.Itoa(numOfBackups))
}

// KeepLatest creates a new flag for the number of latest backups to keep.
func KeepLatest(numOfBackups int) flag.Applier {
	return keepFlag("--keep-latest", numOfBackups)
}

// KeepHourly creates a new flag for the number of hourly backups to keep.
func KeepHourly(numOfBackups int) flag.Applier {
	return keepFlag("--keep-hourly", numOfBackups)
}

// KeepDaily creates a new flag for the number of daily backups to keep.
func KeepDaily(numOfBackups int) flag.Applier {
	return keepFlag("--keep-daily", numOfBackups)
}

// KeepWeekly creates a new flag for the number of weekly backups to keep.
func KeepWeekly(numOfBackups int) flag.Applier {
	return keepFlag("--keep-weekly", numOfBackups)
}

// KeepMonthly creates a new flag for the number of monthly backups to keep.
func KeepMonthly(numOfBackups int) flag.Applier {
	return keepFlag("--keep-monthly", numOfBackups)
}

// KeepAnnual creates a new flag for the number of annual backups to keep.
func KeepAnnual(numOfBackups int) flag.Applier {
	return keepFlag("--keep-annual", numOfBackups)
}

// BackupRetentionPolicy defines the retention policy flags for backups.
type BackupRetentionPolicy struct {
	KeepLatest  flag.Applier
	KeepHourly  flag.Applier
	KeepDaily   flag.Applier
	KeepWeekly  flag.Applier
	KeepMonthly flag.Applier
	KeepAnnual  flag.Applier
}

// Apply applies the retention policy flags to the CLI.
func (f BackupRetentionPolicy) Apply(cli safecli.CommandAppender) error {
	return flag.Apply(cli,
		f.KeepLatest,
		f.KeepHourly,
		f.KeepDaily,
		f.KeepWeekly,
		f.KeepMonthly,
		f.KeepAnnual,
	)
}

// Global creates a new global flag.
func Global(enable bool) flag.Applier {
	return flag.NewBoolFlag("--global", enable)
}
