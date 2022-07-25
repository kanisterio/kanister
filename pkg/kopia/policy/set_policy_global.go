package policy

import (
	"strconv"

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kopia/command"
)

// PolicySetGlobalCommand creates the command for setting the global policy to the desired settings.
func PolicySetGlobalCommand(encryptionKey, configFilePath, logDirectory string) []string {
	const maxInt32 = 1<<31 - 1

	pc := command.PolicyChangesArg{
		// Retention changes
		kopia.KeepLatest:  strconv.Itoa(maxInt32),
		kopia.KeepHourly:  strconv.Itoa(0),
		kopia.KeepDaily:   strconv.Itoa(0),
		kopia.KeepWeekly:  strconv.Itoa(0),
		kopia.KeepMonthly: strconv.Itoa(0),
		kopia.KeepAnnual:  strconv.Itoa(0),

		// Compression changes
		kopia.CompressionAlgorithm: kopia.S2DefaultComprAlgo,
	}

	args := command.PolicySetGlobalCommandArgs{
		CommandArgs: &command.CommandArgs{
			EncryptionKey:  encryptionKey,
			ConfigFilePath: configFilePath,
			LogDirectory:   logDirectory,
		},
		Modifications: pc,
	}
	return command.PolicySetGlobal(args)
}
