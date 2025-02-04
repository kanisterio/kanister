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

	"github.com/kopia/kopia/fs"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/policy"
	"gopkg.in/check.v1"
)

type KopiaParseUtilsTestSuite struct{}

var _ = check.Suite(&KopiaParseUtilsTestSuite{})

func (kParse *KopiaParseUtilsTestSuite) TestSnapshotIDsFromSnapshot(c *check.C) {
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

func (kParse *KopiaParseUtilsTestSuite) TestLatestSnapshotInfoFromManifestList(c *check.C) {
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

func (kParse *KopiaParseUtilsTestSuite) TestSnapshotInfoFromSnapshotCreateOutput(c *check.C) {
	for _, tc := range []struct {
		output         string
		checker        check.Checker
		expectedSnapID string
		expectedRootID string
	}{
		{
			output: `Snapshotting u2@h2:/tmp/aaa1 ...
			* 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB, estimating...
		   {"id":"00000000000000000000001","source":{"host":"h2","userName":"u2","path":"/tmp/aaa1"},"description":"","startTime":"2021-05-26T05:29:07.206854927Z","endTime":"2021-05-26T05:29:07.207328392Z","rootEntry":{"name":"aaa1","type":"d","mode":"0755","mtime":"2021-05-19T15:45:34.448853232Z","obj":"ka68ba7abe0818b24a2b0647aeeb02f29","summ":{"size":0,"files":1,"symlinks":0,"dirs":1,"maxTime":"2021-05-19T15:45:34.448853232Z","numFailed":0}}}
		   `,
			checker:        check.IsNil,
			expectedSnapID: "00000000000000000000001",
			expectedRootID: "ka68ba7abe0818b24a2b0647aeeb02f29",
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
		{
			output:         `{"id":"1b6639b9797dc77dd4ddf57723918187","source":{"host":"da","userName":"kk","path":"/mnt/nfspvc"},"description":"","startTime":"2023-07-13T00:08:08.049239555Z","endTime":"2023-07-13T00:08:08.054904252Z","incomplete":"canceled","rootEntry":{"name":"nfspvc","type":"d","mode":"0755","mtime":"2023-07-11T20:33:41.386653643Z","obj":"k453085aaf775ecb9018a3fa8e276ca5d","summ":{"size":0,"files":0,"symlinks":0,"dirs":2,"maxTime":"2023-07-11T20:33:27.628326361Z","incomplete":"canceled","numFailed":1,"errors":[{"path":"for1001","error":"permission denied"}]}}}`,
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

func (kParse *KopiaParseUtilsTestSuite) TestSnapSizeStatsFromSnapListAll(c *check.C) {
	for _, tc := range []struct {
		description     string
		outputGenFunc   func(*check.C, []*snapshot.Manifest) string
		expManifestList []*snapshot.Manifest
		expCount        int
		expSize         int64
		errChecker      check.Checker
	}{
		{
			description:     "empty manifest list",
			outputGenFunc:   marshalManifestList,
			expManifestList: []*snapshot.Manifest{},
			expCount:        0,
			expSize:         0,
			errChecker:      check.IsNil,
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
			errChecker: check.IsNil,
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
			errChecker: check.IsNil,
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
			errChecker: check.IsNil,
		},
		{
			description:   "error: snapshot with no root entry, size is treated as zero",
			outputGenFunc: marshalManifestList,
			expManifestList: []*snapshot.Manifest{
				{},
			},
			expCount:   1,
			expSize:    0,
			errChecker: check.IsNil,
		},
		{
			description: "error: parse empty output",
			outputGenFunc: func(c *check.C, manifestList []*snapshot.Manifest) string {
				return ""
			},
			expCount:   0,
			expSize:    0,
			errChecker: check.NotNil,
		},
		{
			description: "error: unmarshal fails",
			outputGenFunc: func(c *check.C, manifestList []*snapshot.Manifest) string {
				return "asdf"
			},
			expCount:   0,
			expSize:    0,
			errChecker: check.NotNil,
		},
	} {
		outputToParse := tc.outputGenFunc(c, tc.expManifestList)
		gotTotSizeB, gotNumSnapshots, err := SnapSizeStatsFromSnapListAll(outputToParse)
		c.Check(err, tc.errChecker, check.Commentf("Failed for output: %q", outputToParse))
		c.Check(gotTotSizeB, check.Equals, tc.expSize)
		c.Check(gotNumSnapshots, check.Equals, tc.expCount)
		c.Log(err)
	}
}

func (kParse *KopiaParseUtilsTestSuite) TestSnapshotStatsFromSnapshotCreate(c *check.C) {
	interstitialSubstrings := []string{
		"",
		" (3 errors ignored)",
		" (5 fatal errors)",
		" (5 fatal errors) (3 errors ignored)",
	}
	type args struct {
		// this is a format string, with the %s being replace by one of the interstitial substrings above.
		snapCreateOutputFmt string
		matchOnlyFinished   bool
	}
	tests := []struct {
		name      string
		args      args
		wantStats *SnapshotCreateStats
	}{
		{
			name: "Basic test case",
			args: args{
				snapCreateOutputFmt: " \\ 0 hashing, 1 hashed (2 B), 3 cached (40 KB), uploaded 6.7 GB%s, estimated 2044.2 MB (95.5%%) 0s left",
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
				snapCreateOutputFmt: " - 0 hashing, 283 hashed (219.5 MB), 0 cached (0 B), uploaded 10.5 MB%s, estimated 6.01 MB (91.7%%) 1s left",
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
				snapCreateOutputFmt: " * 0 hashing, 123 hashed (1234.5 MB), 123 cached (1234 B), uploaded 1234.5 KB%s, estimated 941.2 KB (100.0%%) 0s left",
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
				snapCreateOutputFmt: "| 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB%s, estimating...",
				matchOnlyFinished:   true,
			},
			wantStats: nil,
		},
		{
			name: "Check estimating when running",
			args: args{
				snapCreateOutputFmt: "| 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB%s, estimating...",
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
				snapCreateOutputFmt: "* 0 hashing, 1 hashed (2 B), 3 cached (4 B), uploaded 5 KB%s, estimating...",
				matchOnlyFinished:   true,
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
		{
			name: "Progress 100% and still not ready - set 99%",
			args: args{
				snapCreateOutputFmt: "| 0 hashing, 123 hashed (1234.5 MB), 123 cached (1234 B), uploaded 1234.5 KB%s, estimated 941.2 KB (100.0%%) 0s left",
				matchOnlyFinished:   false,
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     123,
				SizeHashedB:     1234500000,
				FilesCached:     123,
				SizeCachedB:     1234,
				SizeUploadedB:   1234500,
				SizeEstimatedB:  941200,
				ProgressPercent: 99,
			},
		},
		{
			name: "Progress is over 100% and still not ready - set 99%",
			args: args{
				snapCreateOutputFmt: "| 0 hashing, 20 hashed (95.1 MB), 20730 cached (4.4 GB), uploaded 21.5 MB%s, estimated 1.3 GB (340.0%%) 0s left",
				matchOnlyFinished:   false,
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     20,
				SizeHashedB:     95100000,
				FilesCached:     20730,
				SizeCachedB:     4400000000,
				SizeUploadedB:   21500000,
				SizeEstimatedB:  1300000000,
				ProgressPercent: 99,
			},
		},
		{
			name: "Progress is over 100% and finished - set 100%",
			args: args{
				snapCreateOutputFmt: " * 0 hashing, 20 hashed (95.1 MB), 20730 cached (4.4 GB), uploaded 21.5 MB%s, estimated 1.3 GB (340.0%%) 0s left",
				matchOnlyFinished:   false,
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     20,
				SizeHashedB:     95100000,
				FilesCached:     20730,
				SizeCachedB:     4400000000,
				SizeUploadedB:   21500000,
				SizeEstimatedB:  1300000000,
				ProgressPercent: 100,
			},
		},
		{
			name: "Progress is less 100% and finished - set 100%",
			args: args{
				snapCreateOutputFmt: " * 0 hashing, 283 hashed (219.5 MB), 0 cached (0 B), uploaded 10.5 MB%s, estimated 6.01 MB (91.7%%) 1s left",
			},
			wantStats: &SnapshotCreateStats{
				FilesHashed:     283,
				SizeHashedB:     219500000,
				FilesCached:     0,
				SizeCachedB:     0,
				SizeUploadedB:   10500000,
				SizeEstimatedB:  6010000,
				ProgressPercent: 100,
			},
		},
	}
	for _, tt := range tests {
		for _, interstital := range interstitialSubstrings {
			outputString := fmt.Sprintf(tt.args.snapCreateOutputFmt, interstital)
			stats := SnapshotStatsFromSnapshotCreate(outputString, tt.args.matchOnlyFinished)
			c.Check(stats, check.DeepEquals, tt.wantStats, check.Commentf("Failed for %s [%s]", tt.name, outputString))
		}
	}
}

func (kParse *KopiaParseUtilsTestSuite) TestRestoreStatsFromRestoreOutput(c *check.C) {
	type args struct {
		restoreOutput string
	}
	tests := []struct {
		name      string
		args      args
		wantStats *RestoreStats
	}{
		{
			name: "Basic test case",
			args: args{
				restoreOutput: "Processed 2 (397.5 MB) of 3 (3.1 GB) 14.9 MB/s (12.6%) remaining 3m3s.",
			},
			wantStats: &RestoreStats{
				FilesProcessed:  2,
				SizeProcessedB:  397500000,
				FilesTotal:      3,
				SizeTotalB:      3100000000,
				ProgressPercent: 12,
			},
		},
		{
			name: "Real test case",
			args: args{
				restoreOutput: "Processed 2 (13.7 MB) of 2 (3.1 GB) 8.5 MB/s (0.4%) remaining 6m10s.",
			},
			wantStats: &RestoreStats{
				FilesProcessed:  2,
				SizeProcessedB:  13700000,
				FilesTotal:      2,
				SizeTotalB:      3100000000,
				ProgressPercent: 0,
			},
		},
		{
			name: "Ignore incomplete stats without during estimation",
			args: args{
				restoreOutput: "Processed 2 (32.8 KB) of 2 (3.1 GB).",
			},
			wantStats: nil,
		},
		{
			name: "Progress is over 100% and still not ready - set 99%",
			args: args{
				restoreOutput: "Processed 2 (13.7 MB) of 2 (3.1 GB) 8.5 MB/s (120.4%) remaining 6m10s.",
			},
			wantStats: &RestoreStats{
				FilesProcessed:  2,
				SizeProcessedB:  13700000,
				FilesTotal:      2,
				SizeTotalB:      3100000000,
				ProgressPercent: 99,
			},
		},
	}
	for _, tt := range tests {
		stats := RestoreStatsFromRestoreOutput(tt.args.restoreOutput)
		c.Check(stats, check.DeepEquals, tt.wantStats, check.Commentf("Failed for %s", tt.name))
	}
}

func (kParse *KopiaParseUtilsTestSuite) TestPhysicalSizeFromBlobStatsRaw(c *check.C) {
	for _, tc := range []struct {
		blobStatsOutput string
		expSizeVal      int64
		expCount        int
		errChecker      check.Checker
	}{
		{
			"Count: 813\nTotal: 11235\n",
			11235,
			813,
			check.IsNil,
		},
		{
			"Total: 11235\nCount: 813\n",
			11235,
			813,
			check.IsNil,
		},
		{
			"Count: 0\nTotal: 0\n",
			0,
			0,
			check.IsNil,
		},
		{
			"Count: 5\nTotal: 0.0\n",
			0,
			0,
			check.NotNil,
		},
		{
			"Count: 5\nTotal: asdf\n",
			0,
			0,
			check.NotNil,
		},
		{
			"Count: 5\nTotal: 11235,\n",
			0,
			0,
			check.NotNil,
		},
		{
			"Total: -11235\n",
			0,
			0,
			check.NotNil,
		},
		{
			"Total: 11235",
			0,
			0,
			check.NotNil,
		},
		{
			"Count: 11235",
			0,
			0,
			check.NotNil,
		},
		{
			"Other-field: 11235",
			0,
			0,
			check.NotNil,
		},
		{
			"random input that doesn't comply with expected format",
			0,
			0,
			check.NotNil,
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
			check.IsNil,
		},
	} {
		gotSize, gotCount, err := RepoSizeStatsFromBlobStatsRaw(tc.blobStatsOutput)
		c.Check(err, tc.errChecker, check.Commentf("Failed for log: %s", tc.blobStatsOutput))
		c.Check(gotSize, check.Equals, tc.expSizeVal)
		c.Check(gotCount, check.Equals, tc.expCount)
	}
}

func (kParse *KopiaParseUtilsTestSuite) TestIsEqualSnapshotCreateStats(c *check.C) {
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
		result := IsEqualSnapshotCreateStats(tc.a, tc.b)
		c.Check(result, check.Equals, tc.expResult)
	}
}

func (kParse *KopiaParseUtilsTestSuite) TestErrorsFromOutput(c *check.C) {
	for caseIdx, tc := range []struct {
		log            string
		expectedErrors []string
	}{
		// Some real error case
		{"ERROR open repository: repository is not connected. See https://kopia.io/docs/repositories/", []string{"open repository: repository is not connected. See https://kopia.io/docs/repositories/"}},
		// The same as previous but with coloured ERROR word
		{string([]byte{27, 91, 51, 49, 109, 69, 82, 82, 79, 82, 27, 91, 48, 109, 32, 111, 112, 101, 110, 32, 114, 101, 112, 111, 115, 105, 116, 111, 114, 121, 58, 32, 114, 101, 112, 111, 115, 105, 116, 111, 114, 121, 32, 105, 115, 32, 110, 111, 116, 32, 99, 111, 110, 110, 101, 99, 116, 101, 100, 46}), []string{"open repository: repository is not connected."}},
		// Multiple error lines (seems not possible in real life, but just to cover this possibility)
		{"ERROR open repository: repository is not connected. See https://kopia.io/docs/repositories/\r\nERROR another error", []string{"open repository: repository is not connected. See https://kopia.io/docs/repositories/", "another error"}},
		// Error surrounded by other output
		{"some text\r\nERROR open repository: repository is not connected. See https://kopia.io/docs/repositories/\r\nanother text line", []string{"open repository: repository is not connected. See https://kopia.io/docs/repositories/"}},
		// No error in output
		{"some text\r\nanother text line", []string{}},
		{"  2009-11-10 23:00:00 UTC <ERROR> some error\n", []string{"some error"}},
		{"error setting attributes: could not change owner/group", []string{"setting attributes: could not change owner/group"}},
		{"error restoring: restore error: error copying: copy file: error creating file:", []string{"restoring: restore error: error copying: copy file: error creating file:"}},
	} {
		errs := ErrorsFromOutput(tc.log)
		fc := check.Commentf("Failed for case #%v. Log: %s", caseIdx, tc.log)
		c.Check(len(errs), check.Equals, len(tc.expectedErrors), fc)
		for i, e := range errs {
			c.Check(e.Error(), check.Equals, tc.expectedErrors[i], fc)
		}
	}
}

func (kParse *KopiaParseUtilsTestSuite) TestParsePolicyShow(c *check.C) {
	for _, tc := range []struct {
		description   string
		outputGenFunc func(*check.C, policy.Policy) string
		expPolicyShow policy.Policy
		errChecker    check.Checker
	}{
		{
			description:   "empty policy show",
			outputGenFunc: marshalPolicy,
			expPolicyShow: policy.Policy{},
			errChecker:    check.IsNil,
		},
		{
			description:   "default policy show",
			outputGenFunc: marshalPolicy,
			expPolicyShow: *policy.DefaultPolicy,
			errChecker:    check.IsNil,
		},
		{
			description: "error: parse empty output",
			outputGenFunc: func(*check.C, policy.Policy) string {
				return ""
			},
			errChecker: check.NotNil,
		},
		{
			description: "error: unmarshal fails",
			outputGenFunc: func(*check.C, policy.Policy) string {
				return "asdf"
			},
			errChecker: check.NotNil,
		},
	} {
		outputToParse := tc.outputGenFunc(c, tc.expPolicyShow)
		gotPolicy, err := ParsePolicyShow(outputToParse)
		c.Check(err, tc.errChecker, check.Commentf("Failed for output: %q", outputToParse))
		c.Log(err)
		c.Check(gotPolicy, check.DeepEquals, tc.expPolicyShow)
	}
}

func marshalManifestList(c *check.C, manifestList []*snapshot.Manifest) string {
	c.Assert(manifestList, check.NotNil)

	b, err := json.Marshal(manifestList)
	c.Assert(err, check.IsNil)

	return string(b)
}

func marshalPolicy(c *check.C, policy policy.Policy) string {
	b, err := json.Marshal(policy)
	c.Assert(err, check.IsNil)

	return string(b)
}
