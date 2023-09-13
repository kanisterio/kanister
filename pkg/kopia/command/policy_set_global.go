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

package command

import "strconv"

type PolicySetGlobalCommandArgs struct {
	*CommandArgs
	Modifications policyChanges
}

// PolicySetGlobal returns the kopia command for modifying the global policy
func PolicySetGlobal(cmdArgs PolicySetGlobalCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(policySubCommand, setSubCommand, globalFlag)
	for field, val := range cmdArgs.Modifications {
		args = args.AppendLoggableKV(field, val)
	}

	return stringSliceCommand(args)
}

func GetPolicyModifications() map[string]string {
	const maxInt32 = 1<<31 - 1

	pc := map[string]string{
		// Retention changes
		keepLatest:  strconv.Itoa(maxInt32),
		keepHourly:  strconv.Itoa(0),
		keepDaily:   strconv.Itoa(0),
		keepWeekly:  strconv.Itoa(0),
		keepMonthly: strconv.Itoa(0),
		keepAnnual:  strconv.Itoa(0),

		// Compression changes
		compressionAlgorithm: s2DefaultComprAlgo,
	}
	return pc
}
