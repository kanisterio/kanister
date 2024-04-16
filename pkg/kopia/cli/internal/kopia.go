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

package internal

import (
	"github.com/kanisterio/safecli"
	"github.com/kanisterio/safecli/command"
)

const (
	kopiaBinName = "kopia"
)

// NewKopiaCommand creates a new safecli.Builder for the kopia command.
func NewKopiaCommand(args ...command.Applier) (*safecli.Builder, error) {
	return command.New(kopiaBinName, args...)
}
