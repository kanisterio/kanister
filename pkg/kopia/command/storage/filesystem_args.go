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

package storage

import (
	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	filesystemSubCommand = "filesystem"
	pathFlag             = "--path"
	// DefaultFSMountPath is the mount path for the file store PVC on Kopia API server
	DefaultFSMountPath = "/mnt/data"
)

func kopiaFilesystemArgs(location map[string]string, artifactPrefix string) logsafe.Cmd {
	artifactPrefix = GenerateFullRepoPath(getPrefixFromMap(location), artifactPrefix)

	args := logsafe.NewLoggable(filesystemSubCommand)
	return args.AppendLoggableKV(pathFlag, DefaultFSMountPath+"/"+artifactPrefix)
}
