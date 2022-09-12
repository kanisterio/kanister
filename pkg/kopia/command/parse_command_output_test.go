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
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/kopia/kopia/fs"
	"github.com/kopia/kopia/snapshot"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia"
)

func TestPhysicalSizeFromBlobStatsRaw(t *testing.T) {
	c := qt.New(t)

	for _, tc := range []struct {
		blobStatsOutput string
		expSizeVal      int64
		expCount        int
		errChecker      qt.Checker
	}{
		{
			"Count: 813\nTotal: 11235\n",
			11235,
			813,
			qt.IsNil,
		},
		{
			"Total: 11235\nCount: 813\n",
			11235,
			813,
			qt.IsNil,
		},
		{
			"Count: 0\nTotal: 0\n",
			0,
			0,
			qt.IsNil,
		},
		{
			"Count: 5\nTotal: 0.0\n",
			0,
			0,
			qt.IsNotNil,
		},
		{
			"Count: 5\nTotal: asdf\n",
			0,
			0,
			qt.IsNotNil,
		},
		{
			"Count: 5\nTotal: 11235,\n",
			0,
			0,
			qt.IsNotNil,
		},
		{
			"Total: -11235\n",
			0,
			0,
			qt.IsNotNil,
		},
		{
			"Total: 11235",
			0,
			0,
			qt.IsNotNil,
		},
		{
			"Count: 11235",
			0,
			0,
			qt.IsNotNil,
		},
		{
			"Other-field: 11235",
			0,
			0,
			qt.IsNotNil,
		},
		{
			"random input that doesn't comply with expected format",
			0,
			0,
			qt.IsNotNil,
		},
		{
			`
Count: 26
Total: 65628
Average: 2524
Histogram:

		0 between 0 and 10 (total 0)
		0 between 10 and 100 (total 0)
		11 between 100 and 1000 (total 2132)
		15 between 1000 and 10000 (total 63496)
		0 between 10000 and 100000 (total 0)
		0 between 100000 and 1000000 (total 0)
		0 between 1000000 and 10000000 (total 0)
		0 between 10000000 and 100000000 (total 0)`,
			65628,
			26,
			qt.IsNil,
		},
	} {
		gotSize, gotCount, err := RepoSizeStatsFromBlobStatsRaw(tc.blobStatsOutput)
		c.Check(err, tc.errChecker, qt.Commentf("Failed for log: %s", tc.blobStatsOutput))
		c.Check(gotSize, qt.Equals, tc.expSizeVal)
		c.Check(gotCount, qt.Equals, tc.expCount)
	}
}

func TestSnapSizeStatsFromSnapListAll(t *testing.T) {
	for _, tc := range []struct {
		description     string
		outputGenFunc   func(*qt.C, []*snapshot.Manifest) string
		expManifestList []*snapshot.Manifest
		expCount        int
		expSize         int64
		errChecker      qt.Checker
	}{
		{
			description:     "empty manifest list",
			outputGenFunc:   marshalManifestList,
			expManifestList: []*snapshot.Manifest{},
			expCount:        0,
			expSize:         0,
			errChecker:      qt.IsNil,
		},
		{
			description:   "basic manifest list",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 1,
						},
					},
				},
			},
			expCount:   1,
			expSize:    1,
			errChecker: qt.IsNil,
		},
		{
			description:   "manifest list with multiple snapshots",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 1,
						},
					},
				},
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 10,
						},
					},
				},
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 100,
						},
					},
				},
				{
					RootEntry: &snapshot.DirEntry{
						DirSummary: &fs.DirectorySummary{
							TotalFileSize: 1000,
						},
					},
				},
			},
			expCount:   4,
			expSize:    1111,
			errChecker: qt.IsNil,
		},
		{
			description:   "error: snapshot with no directory summary, size is treated as zero",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{
					RootEntry: &snapshot.DirEntry{},
				},
			},
			expCount:   1,
			expSize:    0,
			errChecker: qt.IsNil,
		},
		{
			description:   "error: snapshot with no root entry, size is treated as zero",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{},
			},
			expCount:   1,
			expSize:    0,
			errChecker: qt.IsNil,
		},
		{
			description: "error: parse empty output",
			outputGenFunc: func(c *qt.C, manifestList []*snapshot.Manifest) string {
				return ""
			},
			expCount:   0,
			expSize:    0,
			errChecker: qt.IsNotNil,
		},
		{
			description: "error: unmarshal fails",
			outputGenFunc: func(c *qt.C, manifestList []*snapshot.Manifest) string {
				return "asdf"
			},
			expCount:   0,
			expSize:    0,
			errChecker: qt.IsNotNil,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			c := qt.New(t)
			outputToParse := tc.outputGenFunc(c, tc.expManifestList)
			gotTotSizeB, gotNumSnapshots, err := SnapSizeStatsFromSnapListAll(outputToParse)
			c.Check(err, tc.errChecker, qt.Commentf("Failed for output: %q", outputToParse))
			c.Check(gotTotSizeB, qt.Equals, tc.expSize)
			c.Check(gotNumSnapshots, qt.Equals, tc.expCount)
			c.Log(err)
		})
	}
}

func marshalManifestList(c *qt.C, manifestList []*snapshot.Manifest) string {
	c.Assert(manifestList, qt.IsNotNil)

	b, err := json.Marshal(manifestList)
	c.Assert(err, qt.IsNil)

	return string(b)
}

// TestKopiaPolicySetGlobalCommand
// Motivation: very basic test for populating the kopia policy set command fields
// Description:
//   - Feed different combinations of RetentionChanges and compression algorithm changes
//      into PolicySetGlobalCommand, the function that constructs the policy set command.
//   - Check that the command has the requested fields, does not have any unrequested
//     fields, and the field values that appear match as requested.
func TestKopiaPolicySetGlobalCommand(t *testing.T) {
	c := qt.New(t)

	const maxInt32 = 1<<31 - 1
	for _, tc := range []struct {
		rc PolicyChangesArg
	}{
		{rc: PolicyChangesArg{
			kopia.KeepLatest: strconv.Itoa(maxInt32),
		}},
		{rc: PolicyChangesArg{
			kopia.KeepLatest:  strconv.Itoa(rand.Intn(maxInt32)),
			kopia.KeepHourly:  strconv.Itoa(rand.Intn(maxInt32)),
			kopia.KeepDaily:   strconv.Itoa(rand.Intn(maxInt32)),
			kopia.KeepWeekly:  strconv.Itoa(rand.Intn(maxInt32)),
			kopia.KeepMonthly: strconv.Itoa(rand.Intn(maxInt32)),
			kopia.KeepAnnual:  strconv.Itoa(rand.Intn(maxInt32)),
		}},
		{rc: PolicyChangesArg{}},
		{rc: PolicyChangesArg{
			kopia.KeepLatest:  strconv.Itoa(0),
			kopia.KeepHourly:  strconv.Itoa(0),
			kopia.KeepDaily:   strconv.Itoa(0),
			kopia.KeepWeekly:  strconv.Itoa(0),
			kopia.KeepMonthly: strconv.Itoa(0),
			kopia.KeepAnnual:  strconv.Itoa(0),
		}},
		{rc: PolicyChangesArg{
			kopia.CompressionAlgorithm: "compr-algo",
		}},
		{rc: PolicyChangesArg{
			kopia.CompressionAlgorithm: kopia.S2DefaultComprAlgo,
		}},
		{rc: PolicyChangesArg{
			kopia.CompressionAlgorithm: kopia.S2DefaultComprAlgo,
			kopia.KeepLatest:           strconv.Itoa(0),
			kopia.KeepHourly:           strconv.Itoa(0),
			kopia.KeepDaily:            strconv.Itoa(0),
			kopia.KeepWeekly:           strconv.Itoa(0),
			kopia.KeepMonthly:          strconv.Itoa(0),
			kopia.KeepAnnual:           strconv.Itoa(0),
		}},
	} {
		encryptionKey := "asdf"
		args := PolicySetGlobalCommandArgs{
			CommandArgs: &CommandArgs{
				EncryptionKey:  encryptionKey,
				ConfigFilePath: "path/kopia.config",
				LogDirectory:   "cache/log",
			},
			Modifications: tc.rc,
		}
		kopiaCmd := PolicySetGlobal(args)

		fieldsFound := make(map[string]bool)
		for i, field := range kopiaCmd {
			switch {
			case hasKnownFlag(field):
				// Executed only for policy set command flags
				// Finds the flag with its value of form `flag=value`
				// Extracts and checks if the correct value is set
				c.Assert(i < len(kopiaCmd), qt.Equals, true)
				flagEqVal := kopiaCmd[i]
				args := strings.Split(flagEqVal, "=")
				c.Check(len(args) > 0, qt.Equals, true)
				key := args[0]
				val := args[1]
				c.Check(val, qt.Equals, tc.rc[key])
				_, ok := fieldsFound[field]
				// Expect no duplicate fields
				c.Check(ok, qt.Equals, false)
				fieldsFound[field] = true
			}
		}

		// Check all changing fields were found in the command
		c.Check(len(fieldsFound), qt.Equals, len(tc.rc))
	}
}

func hasKnownFlag(field string) bool {
	return strings.Contains(field, kopia.KeepLatest) ||
		strings.Contains(field, kopia.KeepHourly) ||
		strings.Contains(field, kopia.KeepDaily) ||
		strings.Contains(field, kopia.KeepWeekly) ||
		strings.Contains(field, kopia.KeepMonthly) ||
		strings.Contains(field, kopia.KeepAnnual) ||
		strings.Contains(field, kopia.CompressionAlgorithm)
}

func TestSnapshotStatsFromSnapshotCreate(t *testing.T) {
	type args struct {
		snapCreateOutput  string
		matchOnlyFinished bool
	}
	tests := []struct {
		name      string
		args      args
		wantStats *SnapshotCreateStats
	}{
		{
			name: "Basic test case",
			args: args{
				snapCreateOutput: " \\ 0 hashing, 1 hashed (2 B), 3 cached (40 KB), uploaded 6.7 GB, estimated 2044.2 MB (95.5%) 0s left",
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     1,
				SizeHashedB:     2,
				FilesCached:     3,
				SizeCachedB:     40000,
				SizeUploadedB:   6700000000,
				SizeEstimatedB:  2044200000,
				ProgressPercent: 95,
			},
		},
		{
			name: "Real test case",
			args: args{
				snapCreateOutput: " - 0 hashing, 283 hashed (219.5 MB), 0 cached (0 B), uploaded 10.5 MB, estimated 6.01 MB (91.7%) 1s left",
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     283,
				SizeHashedB:     219500000,
				FilesCached:     0,
				SizeCachedB:     0,
				SizeUploadedB:   10500000,
				SizeEstimatedB:  6010000,
				ProgressPercent: 91,
			},
		},
		{
			name: "Check multiple digits each field",
			args: args{
				snapCreateOutput: " * 0 hashing, 123 hashed (1234.5 MB), 123 cached (1234 B), uploaded 1234.5 KB, estimated 941.2 KB (100.0%) 0s left",
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     123,
				SizeHashedB:     1234500000,
				FilesCached:     123,
				SizeCachedB:     1234,
				SizeUploadedB:   1234500,
				SizeEstimatedB:  941200,
				ProgressPercent: 100,
			},
		},
		{
			name: "Ignore running output when expecting completed line",
			args: args{
				snapCreateOutput:  "| 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB, estimating...",
				matchOnlyFinished: true,
			},
			wantStats: nil,
		},
		{
			name: "Check estimating when running",
			args: args{
				snapCreateOutput: "| 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB, estimating...",
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     1,
				SizeHashedB:     2,
				FilesCached:     3,
				SizeCachedB:     4,
				SizeUploadedB:   5000,
				SizeEstimatedB:  0,
				ProgressPercent: 0,
			},
		},
		{
			name: "Check estimating when finished",
			args: args{
				snapCreateOutput:  "* 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB, estimating...",
				matchOnlyFinished: true,
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     1,
				SizeHashedB:     2,
				FilesCached:     3,
				SizeCachedB:     4,
				SizeUploadedB:   5000,
				SizeEstimatedB:  0,
				ProgressPercent: 100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			if gotStats := SnapshotStatsFromSnapshotCreate(tt.args.snapCreateOutput, tt.args.matchOnlyFinished); !reflect.DeepEqual(gotStats, tt.wantStats) {
				c.Errorf("SnapshotStatsFromSnapshotCreate() = %v, want %v", gotStats, tt.wantStats)
			}
		})
	}
}

func TestGeneralKopiaCommandLogging(t *testing.T) {
	c := qt.New(t)

	password := "testpass"
	configFile := "path/kopia.config"
	logDir := "cache/log"
	owner := "username@hostname"
	for _, tc := range []struct {
		params      KopiaCommandParams
		expectedCmd string
	}{
		{
			params: KopiaCommandParams{
				SubCommands: []string{"repository", "status"},
			},
			expectedCmd: fmt.Sprint("kopia --log-level=error --config-file=", configFile, " --log-dir=", logDir, " --password=<****> repository status"),
		},
		{
			params: KopiaCommandParams{
				SubCommands:  []string{"repository", "set-client"},
				LoggableFlag: []string{"--read-only"},
			},
			expectedCmd: fmt.Sprint("kopia --log-level=error --config-file=", configFile, " --log-dir=", logDir, " --password=<****> repository set-client --read-only"),
		},
		{
			params: KopiaCommandParams{
				SubCommands: []string{"maintenance", "set"},
				LoggableKV: map[string]string{
					ownerFlag: owner,
				},
			},
			expectedCmd: fmt.Sprint("kopia --log-level=error --config-file=", configFile, " --log-dir=", logDir, " --password=<****> maintenance set --owner=username@hostname"),
		},
		{
			params: KopiaCommandParams{
				SubCommands: []string{"repository", "status"},
				RedactedKV: map[string]string{
					serverCertFingerprint: "test-fingerprint",
				},
			},
			expectedCmd: fmt.Sprint("kopia --log-level=error --config-file=", configFile, " --log-dir=", logDir, " --password=<****> repository status --server-cert-fingerprint=<****>"),
		},
	} {
		cmd := GeneralCommand(tc.params, password, configFile, logDir)
		c.Check(cmd.String(), qt.Equals, tc.expectedCmd)
	}
}

func TestIsEqualSnapshotCreateStats(t *testing.T) {
	for _, tc := range []struct {
		description string
		a           *SnapshotCreateStats
		b           *SnapshotCreateStats
		expResult   bool
	}{
		{
			"Both nil",
			nil,
			nil,
			true,
		},
		{
			"First nil",
			nil,
			&SnapshotCreateStats{
				FilesHashed:     1,
				SizeHashedB:     2,
				FilesCached:     3,
				SizeCachedB:     4,
				SizeUploadedB:   5,
				SizeEstimatedB:  6,
				ProgressPercent: 7,
			},
			false,
		},
		{
			"Second nil",
			&SnapshotCreateStats{
				FilesHashed:     1,
				SizeHashedB:     2,
				FilesCached:     3,
				SizeCachedB:     4,
				SizeUploadedB:   5,
				SizeEstimatedB:  6,
				ProgressPercent: 7,
			},
			nil,
			false,
		},
		{
			"Not nil, equal",
			&SnapshotCreateStats{
				FilesHashed:     1,
				SizeHashedB:     2,
				FilesCached:     3,
				SizeCachedB:     4,
				SizeUploadedB:   5,
				SizeEstimatedB:  6,
				ProgressPercent: 7,
			},
			&SnapshotCreateStats{
				FilesHashed:     1,
				SizeHashedB:     2,
				FilesCached:     3,
				SizeCachedB:     4,
				SizeUploadedB:   5,
				SizeEstimatedB:  6,
				ProgressPercent: 7,
			},
			true,
		},
		{
			"Not nil, not equal",
			&SnapshotCreateStats{
				FilesHashed:     1,
				SizeHashedB:     2,
				FilesCached:     3,
				SizeCachedB:     4,
				SizeUploadedB:   5,
				SizeEstimatedB:  6,
				ProgressPercent: 7,
			},
			&SnapshotCreateStats{
				FilesHashed:     5,
				SizeHashedB:     7,
				FilesCached:     2,
				SizeCachedB:     8,
				SizeUploadedB:   5,
				SizeEstimatedB:  2,
				ProgressPercent: 1,
			},
			false,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			c := qt.New(t)
			result := IsEqualSnapshotCreateStats(tc.a, tc.b)
			c.Check(result, qt.Equals, tc.expResult)
		})
	}
}

type KopiaParseUtilsTestSuite struct{}

var _ = check.Suite(&KopiaParseUtilsTestSuite{})

func (s *KopiaParseUtilsTestSuite) TestSnapshotIDsFromSnapshot(c *check.C) {
	for _, tc := range []struct {
		log            string
		expectedSnapID string
		expectedRootID string
		errChecker     check.Checker
	}{
		{"Created snapshot with root k23cf6d7ff418a0110636399da458abb5 and ID beda41fb4ba7478025778fdc8312355c in 10.8362ms", "beda41fb4ba7478025778fdc8312355c", "k23cf6d7ff418a0110636399da458abb5", check.IsNil},
		{"Created snapshot with root rootID and ID snapID", "snapID", "rootID", check.IsNil},
		{" Created snapshot snapID (root rootID)", "", "", check.NotNil},
		{"root 123abcd", "", "", check.NotNil},
		{"Invalid message", "", "", check.NotNil},
		{"Created snapshot with root abc123\n in 5.5001ms", "", "", check.NotNil},
		{"", "", "", check.NotNil},
		{"Created snapshot", "", "", check.NotNil},
		{"Created snapshot ", "", "", check.NotNil},
		{"Created snapshot with root", "", "", check.NotNil},
		{"Created snapshot with root rootID", "", "", check.NotNil},
		{"Created snapshot with root rootID and ID\n snapID in 10ms", "", "", check.NotNil},
		{"Created snapshot with root rootID in 10ms", "", "", check.NotNil},
		{"Created snapshot and ID snapID in 10ms", "", "", check.NotNil},
		{"Created snapshot with ID snapID in 10ms", "", "", check.NotNil},
		{"Created snapshot snapID\n(root rootID) in 10.8362ms", "", "", check.NotNil},
		{"Created snapshot snapID in 10.8362ms", "", "", check.NotNil},
		{"Created snapshot (root rootID) in 10.8362ms", "", "", check.NotNil},
		{"Created snapshot root rootID in 10.8362ms", "", "", check.NotNil},
		{"Created snapshot root rootID and ID snapID in 10.8362ms", "", "", check.NotNil},
		{" root rootID and ID snapID in 10.8362ms", "", "", check.NotNil},
		{"uploaded snapshot beda41fb4ba7478025778fdc8312355c (root k23cf6d7ff418a0110636399da458abb5) in 10.8362ms", "", "", check.NotNil},
	} {
		snapID, rootID, err := SnapshotIDsFromSnapshot(tc.log)
		c.Check(snapID, check.Equals, tc.expectedSnapID, check.Commentf("Failed for log: %s", tc.log))
		c.Check(rootID, check.Equals, tc.expectedRootID, check.Commentf("Failed for log: %s", tc.log))
		c.Check(err, tc.errChecker, check.Commentf("Failed for log: %s", tc.log))
	}
}

func (s *KopiaParseUtilsTestSuite) TestLatestSnapshotInfoFromManifestList(c *check.C) {
	for _, tc := range []struct {
		output             string
		checker            check.Checker
		expectedSnapID     string
		expectedBackupPath string
	}{
		{
			output: `[
				{"id":"00000000000000000000001","length":604,"labels":{"hostname":"h2","path":"/tmp/aaa1","type":"snapshot","username":"u2"},"mtime":"2021-05-19T11:53:50.882509009Z"},
				{"id":"00000000000000000000002","length":603,"labels":{"hostname":"h2","path":"/tmp/aaa2","type":"snapshot","username":"u2"},"mtime":"2021-05-19T12:24:11.258017051Z"},
				{"id":"00000000000000000000003","length":602,"labels":{"hostname":"h2","path":"/tmp/aaa3","type":"snapshot","username":"u2"},"mtime":"2021-05-19T12:24:25.767315039Z"}
			   ]`,
			expectedSnapID:     "00000000000000000000003",
			expectedBackupPath: "/tmp/aaa3",
			checker:            check.IsNil,
		},
		{
			output:             "",
			expectedSnapID:     "",
			expectedBackupPath: "",
			checker:            check.NotNil,
		},
		{
			output: `[
				{"id":"","length":602,"labels":{"hostname":"h2","path":"/tmp/aaa3","type":"snapshot","username":"u2"},"mtime":"2021-05-19T12:24:25.767315039Z"}
			   ]`,
			expectedSnapID:     "",
			expectedBackupPath: "",
			checker:            check.NotNil,
		},
		{
			output: `[
				{"id":"00000000000000000000003","length":602,"labels":{"hostname":"h2","path":"","type":"snapshot","username":"u2"},"mtime":"2021-05-19T12:24:25.767315039Z"}
			   ]`,
			expectedSnapID:     "",
			expectedBackupPath: "",
			checker:            check.NotNil,
		},
	} {
		snapID, backupPath, err := LatestSnapshotInfoFromManifestList(tc.output)
		c.Assert(err, tc.checker)
		c.Assert(snapID, check.Equals, tc.expectedSnapID)
		c.Assert(backupPath, check.Equals, tc.expectedBackupPath)
	}
}

func (s *KopiaParseUtilsTestSuite) TestSnapshotInfoFromSnapshotCreateOutput(c *check.C) {
	for _, tc := range []struct {
		output         string
		checker        check.Checker
		expectedSnapID string
		expectedRootID string
	}{
		{
			output: `Snapshotting u2@h2:/tmp/aaa1 ...
			* 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB, estimating...
		   {"id":"00000000000000000000001","source":{"host":"h2","userName":"u2","path":"/tmp/aaa1"},"description":"","startTime":"2021-05-26T05:29:07.206854927Z","endTime":"2021-05-26T05:29:07.207328392Z","rootEntry":{"name":"aaa1","type":"d","mode":"0755","mtime":"2021-05-19T15:45:34.448853232Z","obj":"root00000000000000000000001","summ":{"size":0,"files":1,"symlinks":0,"dirs":1,"maxTime":"2021-05-19T15:45:34.448853232Z","numFailed":0}}}
		   `,
			checker:        check.IsNil,
			expectedSnapID: "00000000000000000000001",
			expectedRootID: "root00000000000000000000001",
		},
		{
			output: `Snapshotting u2@h2:/tmp/aaa1 ...
			* 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB, estimating...
		   `,
			checker:        check.NotNil,
			expectedSnapID: "",
			expectedRootID: "",
		},
		{
			output: `ERROR: unable to get local filesystem entry: resolveSymlink: stat: lstat /tmp/aaa2: no such file or directory
			`,
			checker:        check.NotNil,
			expectedSnapID: "",
			expectedRootID: "",
		},
	} {
		snapID, rootID, err := SnapshotInfoFromSnapshotCreateOutput(tc.output)
		c.Assert(err, tc.checker)
		c.Assert(snapID, check.Equals, tc.expectedSnapID)
		c.Assert(rootID, check.Equals, tc.expectedRootID)
	}
}
