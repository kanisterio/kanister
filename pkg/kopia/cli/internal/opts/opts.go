// Copyright 2024 The Kanister Authors.
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

package opts

import "github.com/kanisterio/safecli/command"

// All creates a new all option.
func All(enabled bool) command.Applier {
	return command.NewOption("--all", enabled)
}

// Delta creates a new delta option.
func Delta(enabled bool) command.Applier {
	return command.NewOption("--delta", enabled)
}

// ShowIdentical creates a new show identical option.
func ShowIdentical(enabled bool) command.Applier {
	return command.NewOption("--show-identical", enabled)
}

// ReadOnly creates a new read only option.
func ReadOnly(enabled bool) command.Applier {
	return command.NewOption("--readonly", enabled)
}

// CheckForUpdates creates a new check for updates option.
func CheckForUpdates(enabled bool) command.Applier {
	return command.NewToggleOption("--check-for-updates", "--no-check-for-updates", enabled)
}

// JSON creates a new JSON option.
func JSON(enabled bool) command.Applier {
	return command.NewOption("--json", enabled)
}

// Delete creates a new delete option.
func Delete(enabled bool) command.Applier {
	return command.NewOption("--delete", enabled)
}
