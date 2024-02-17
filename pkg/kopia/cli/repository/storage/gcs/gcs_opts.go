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

package gcs

import (
	"github.com/kanisterio/safecli/command"
)

var (
	subcmdGCS = command.NewArgument("gcs")
)

// optBucket creates a new bucket option with a given name.
func optBucket(name string) command.Applier {
	return command.NewOptionWithArgument("--bucket", name)
}

// optPrefix creates a new prefix option with a given prefix.
func optPrefix(prefix string) command.Applier {
	return command.NewOptionWithArgument("--prefix", prefix)
}

// optCredentialsFile creates a new GCS credentials file option with a given file path.
func optCredentialsFile(path string) command.Applier {
	return command.NewOptionWithArgument("--credentials-file", path)
}
