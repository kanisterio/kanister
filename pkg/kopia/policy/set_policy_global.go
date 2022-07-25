// Copyright 2022 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
