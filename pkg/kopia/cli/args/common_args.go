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

package args

// Common provides the common arguments for Kopia CLI.
type Common struct {
	ConfigFilePath string // the path to the config file.
	LogDirectory   string // the directory where logs are stored.
	LogLevel       string // the level of logging. Default is "error".
	RepoPassword   string // the password for the repository.
}
