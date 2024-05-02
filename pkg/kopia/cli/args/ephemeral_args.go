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

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/args"
)

// The following function can be used to register additional
// kopia arguments.
func RegisterKopiaRepositoryCreateArgs(flag, value string) {
	args.KopiaRepositoryCreateFlagValueMap[flag] = value
}

func RegisterKopiaRepositoryConnectServerArgs(flag, value string) {
	args.KopiaRepositoryConnectServerFlagValueMap[flag] = value
}

func RegisterKopiaUserAddSetArgs(flag, value string) {
	args.KopiaUserAddSetFlagValueMap[flag] = value
}

// The following functions cna be used to unregister
// additional kopia arguments.
func UnRegisterKopiaRepositoryCreateArgs(flag string) {
	delete(args.KopiaRepositoryCreateFlagValueMap, flag)
}

func UnRegisterKopiaRepositoryConnectServerArgs(flag string) {
	delete(args.KopiaRepositoryConnectServerFlagValueMap, flag)
}

func UnRegisterKopiaUserAddSetArgs(flag string) {
	delete(args.KopiaUserAddSetFlagValueMap, flag)
}

// The following functions return a key-value map of additional
// registered kopia arguments.
func ExtraKopiaRepositoryCreateArgs() map[string]string {
	return args.KopiaRepositoryCreateFlagValueMap
}

func ExtraKopiaRepositoryConnectServerArgs() map[string]string {
	return args.KopiaRepositoryConnectServerFlagValueMap
}

func ExtraKopiaUserAddSetArgs() map[string]string {
	return args.KopiaUserAddSetFlagValueMap
}
