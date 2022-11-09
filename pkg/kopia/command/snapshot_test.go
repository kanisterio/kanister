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

import (
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestSnapshotCommands(t *testing.T) {
	c := qt.New(t)

	commandArgs := &CommandArgs{
		RepoPassword:   "encr-key",
		ConfigFilePath: "path/kopia.config",
		LogDirectory:   "cache/log",
	}

	for _, tc := range []struct {
		f           func() []string
		expectedLog string
	}{
		{
			f: func() []string {
				args := SnapshotCreateCommandArgs{
					CommandArgs:            commandArgs,
					PathToBackup:           "path/to/backup",
					ProgressUpdateInterval: 0,
				}
				return SnapshotCreate(args)
			},
			expectedLog: "kopia --log-level=info --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot create path/to/backup --json --parallel=8 --progress-update-interval=1h",
		},
		{
			f: func() []string {
				args := SnapshotCreateCommandArgs{
					CommandArgs:            commandArgs,
					PathToBackup:           "path/to/backup",
					ProgressUpdateInterval: 1*time.Minute + 35*time.Second,
				}
				return SnapshotCreate(args)
			},
			expectedLog: "kopia --log-level=info --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot create path/to/backup --json --parallel=8 --progress-update-interval=2m",
		},
		{
			f: func() []string {
				args := SnapshotExpireCommandArgs{
					CommandArgs: commandArgs,
					RootID:      "root-id",
					MustDelete:  true,
				}
				return SnapshotExpire(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot expire root-id --delete",
		},
		{
			f: func() []string {
				args := SnapshotRestoreCommandArgs{
					CommandArgs:   commandArgs,
					SnapID:        "snapshot-id",
					TargetPath:    "target/path",
					SparseRestore: false,
				}
				return SnapshotRestore(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot restore snapshot-id target/path",
		},
		{
			f: func() []string {
				args := SnapshotRestoreCommandArgs{
					CommandArgs:   commandArgs,
					SnapID:        "snapshot-id",
					TargetPath:    "target/path",
					SparseRestore: true,
				}
				return SnapshotRestore(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot restore snapshot-id target/path --sparse",
		},
		{
			f: func() []string {
				args := SnapshotDeleteCommandArgs{
					CommandArgs: &CommandArgs{
						RepoPassword:   "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					SnapID: "snapshot-id",
				}
				return SnapshotDelete(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot delete snapshot-id --unsafe-ignore-source",
		},
		{
			f: func() []string {
				args := SnapListAllWithSnapIDsCommandArgs{
					CommandArgs: &CommandArgs{
						RepoPassword:   "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
				}
				return SnapListAllWithSnapIDs(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> manifest list --json --filter=type:snapshot",
		},
		{
			f: func() []string {
				args := SnapListByTagsCommandArgs{
					CommandArgs: commandArgs,
					Tags:        []string{"tag1:val1", "tag2:val2"},
				}
				return SnapListByTags(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> snapshot list --all --delta --show-identical --json --tags tag1:val1 --tags tag2:val2",
		},
	} {
		cmd := tc.f()
		c.Check(cmd, qt.Equals, tc.expectedLog)
	}
}
