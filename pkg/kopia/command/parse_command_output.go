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
	"bufio"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/kanisterio/errkit"
	"github.com/kopia/kopia/repo/manifest"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/policy"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	pathKey       = "path"
	typeKey       = "type"
	snapshotValue = "snapshot"

	//nolint:lll
	snapshotCreateOutputRegEx       = `(?P<spinner>[|/\-\\\*]).+[^\d](?P<numHashed>\d+) hashed \((?P<hashedSize>[^\)]+)\), (?P<numCached>\d+) cached \((?P<cachedSize>[^\)]+)\), uploaded (?P<uploadedSize>[^\)]+), (?:estimating...|estimated (?P<estimatedSize>[^\)]+) \((?P<estimatedProgress>[^\)]+)\%\).+)`
	restoreOutputRegEx              = `Processed (?P<processedCount>\d+) \((?P<processedSize>.*)\) of (?P<totalCount>\d+) \((?P<totalSize>.*)\) (?P<dataRate>.*) \((?P<percentage>.*)%\) remaining (?P<remainingTime>.*)\.`
	extractSnapshotIDRegEx          = `Created snapshot with root ([^\s]+) and ID ([^\s]+).*$`
	repoTotalSizeFromBlobStatsRegEx = `Total: (\d+)$`
	repoCountFromBlobStatsRegEx     = `Count: (\d+)$`
)

// SnapshotIDsFromSnapshot extracts root ID of a snapshot from the logs
func SnapshotIDsFromSnapshot(output string) (snapID, rootID string, err error) {
	if output == "" {
		return snapID, rootID, errkit.New("Received empty output")
	}

	logs := regexp.MustCompile("[\r\n]").Split(output, -1)
	pattern := regexp.MustCompile(extractSnapshotIDRegEx)
	for _, l := range logs {
		// Log should contain "Created snapshot with root ABC and ID XYZ..."
		match := pattern.FindAllStringSubmatch(l, 1)
		if len(match) > 0 && len(match[0]) > 2 {
			snapID = match[0][2]
			rootID = match[0][1]
			return snapID, rootID, nil
		}
	}
	return snapID, rootID, errkit.New("Failed to find Root ID from output")
}

// LatestSnapshotInfoFromManifestList returns snapshot ID and backup path of the latest snapshot from `manifests list` output
func LatestSnapshotInfoFromManifestList(output string) (string, string, error) {
	manifestList := []manifest.EntryMetadata{}
	snapID := ""
	backupPath := ""

	err := json.Unmarshal([]byte(output), &manifestList)
	if err != nil {
		return snapID, backupPath, errkit.Wrap(err, "Failed to unmarshal manifest list")
	}
	for _, manifest := range manifestList {
		for key, value := range manifest.Labels {
			if key == pathKey {
				backupPath = value
			}
			if key == typeKey && value == snapshotValue {
				snapID = string(manifest.ID)
			}
		}
	}
	if snapID == "" {
		return "", "", errkit.New("Failed to get latest snapshot ID from manifest list")
	}
	if backupPath == "" {
		return "", "", errkit.New("Failed to get latest snapshot backup path from manifest list")
	}
	return snapID, backupPath, nil
}

// SnapshotInfoFromSnapshotCreateOutput returns snapshot ID and root ID from snapshot create output
func SnapshotInfoFromSnapshotCreateOutput(output string) (string, string, error) {
	snapID := ""
	rootID := ""
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		snapManifest := &snapshot.Manifest{}
		err := json.Unmarshal([]byte(scanner.Text()), snapManifest)
		if err != nil {
			continue
		}
		snapID = string(snapManifest.ID)
		if snapManifest.RootEntry != nil {
			rootID = snapManifest.RootEntry.ObjectID.String()
			if snapManifest.RootEntry.DirSummary != nil && snapManifest.RootEntry.DirSummary.FatalErrorCount > 0 {
				return "", "", errkit.New(fmt.Sprintf("Error occurred during snapshot creation. Output %s", output))
			}
		}
	}
	if snapID == "" {
		return "", "", errkit.New(fmt.Sprintf("Failed to get snapshot ID from create snapshot output %s", output))
	}
	if rootID == "" {
		return "", "", errkit.New(fmt.Sprintf("Failed to get root ID from create snapshot output %s", output))
	}
	return snapID, rootID, nil
}

// SnapSizeStatsFromSnapListAll returns a list of snapshot logical sizes assuming the input string
// is formatted as the output of a kopia snapshot list --all command.
func SnapSizeStatsFromSnapListAll(output string) (totalSizeB int64, numSnapshots int, err error) {
	if output == "" {
		return 0, 0, errkit.New("Received empty output")
	}

	snapList, err := ParseSnapshotManifestList(output)
	if err != nil {
		return 0, 0, errkit.Wrap(err, "Parsing snapshot list output as snapshot manifest list")
	}

	totalSizeB = sumSnapshotSizes(snapList)

	return totalSizeB, len(snapList), nil
}

func sumSnapshotSizes(snapList []*snapshot.Manifest) (sum int64) {
	noSizeDataCount := 0
	for _, snapInfo := range snapList {
		if snapInfo.RootEntry == nil ||
			snapInfo.RootEntry.DirSummary == nil {
			noSizeDataCount++

			continue
		}

		sum += snapInfo.RootEntry.DirSummary.TotalFileSize
	}

	if noSizeDataCount > 0 {
		log.Error().Print("Found snapshot manifests without size data", field.M{"count": noSizeDataCount})
	}

	return sum
}

func ParseSnapshotManifestList(output string) ([]*snapshot.Manifest, error) {
	snapInfoList := []*snapshot.Manifest{}

	if err := json.Unmarshal([]byte(output), &snapInfoList); err != nil {
		return nil, errkit.Wrap(err, "Failed to unmarshal snapshot manifest list")
	}

	return snapInfoList, nil
}

// SnapshotCreateInfo is a container for data that can be parsed from the output of
// `kopia snapshot create`.
type SnapshotCreateInfo struct {
	SnapshotID string
	RootID     string
	Stats      *SnapshotCreateStats
}

// ParseSnapshotCreateOutput parses the output of a snapshot create command into
// a new SnapshotCreateInfo struct and returns its pointer. The Stats field may be nil
// if the stats were unable to be parsed. The root ID and snapshot ID are fetched from
// structured stdout and stats are parsed from stderr output.
func ParseSnapshotCreateOutput(snapCreateStdoutOutput, snapCreateStderrOutput string) (*SnapshotCreateInfo, error) {
	snapID, rootID, err := SnapshotInfoFromSnapshotCreateOutput(snapCreateStdoutOutput)
	if err != nil {
		return nil, err
	}

	return &SnapshotCreateInfo{
		SnapshotID: snapID,
		RootID:     rootID,
		Stats:      SnapshotStatsFromSnapshotCreate(snapCreateStderrOutput, true),
	}, nil
}

// SnapshotCreateStats is a container for stats parsed from the output of a `kopia
// snapshot create` command.
type SnapshotCreateStats struct {
	FilesHashed     int64
	SizeHashedB     int64
	FilesCached     int64
	SizeCachedB     int64
	SizeUploadedB   int64
	SizeEstimatedB  int64
	ProgressPercent int64
}

var (
	kopiaProgressPattern = regexp.MustCompile(snapshotCreateOutputRegEx)
	kopiaRestorePattern  = regexp.MustCompile(restoreOutputRegEx)
)

// SnapshotStatsFromSnapshotCreate parses the output of a `kopia snapshot
// create` line-by-line in search of progress statistics.
// It returns nil if no statistics are found, or the most recent statistic
// if multiple are encountered.
func SnapshotStatsFromSnapshotCreate(
	snapCreateStderrOutput string,
	matchOnlyFinished bool,
) (stats *SnapshotCreateStats) {
	if snapCreateStderrOutput == "" {
		return nil
	}
	logs := regexp.MustCompile("[\r\n]").Split(snapCreateStderrOutput, -1)

	// Match a pattern starting with "*" (signifying upload finished), and containing
	// the repeated pattern "<\d+> <type> (<humanized size base 10>),",
	// where <type> is "hashed", "cached", and "uploaded".
	// Example input:
	// 	 * 0 hashing, 1 hashed (2 B), 3 cached (40 KB), uploaded 6.7 GB, estimated 1092.3 MB (100.0%) 0s left
	// Expected output:
	// SnapshotCreateStats{
	// 		filesHashed:  1,
	// 		sizeHashedB: 2,
	// 		filesCached:  3,
	// 		sizeCachedB: 40000,
	// 		sizeUploadedB: 6700000000,
	// 		sizeEstimatedB: 1092000000,
	// 		progressPercent: 100,
	// }, nil

	for _, l := range logs {
		lineStats := parseKopiaProgressLine(l, matchOnlyFinished)
		if lineStats != nil {
			stats = lineStats
		}
	}

	return stats
}

func parseKopiaProgressLine(line string, matchOnlyFinished bool) (stats *SnapshotCreateStats) {
	match := kopiaProgressPattern.FindStringSubmatch(line)
	if len(match) < 9 {
		return nil
	}

	groups := make(map[string]string)
	for i, name := range kopiaProgressPattern.SubexpNames() {
		if i != 0 && name != "" {
			groups[name] = match[i]
		}
	}

	isFinalResult := groups["spinner"] == "*"
	if matchOnlyFinished && !isFinalResult {
		return nil
	}

	numHashed, err := strconv.Atoi(groups["numHashed"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse number of hashed files", field.M{"numHashed": groups["numHashed"]})
		return nil
	}

	numCached, err := strconv.Atoi(groups["numCached"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse number of cached files", field.M{"numCached": groups["numCached"]})
		return nil
	}

	hashedSizeBytes, err := humanize.ParseBytes(groups["hashedSize"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse hashed size string", field.M{"hashedSize": groups["hashedSize"]})
		return nil
	}

	cachedSizeBytes, err := humanize.ParseBytes(groups["cachedSize"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse cached size string", field.M{"cachedSize": groups["cachedSize"]})
		return nil
	}

	uploadedSizeBytes, err := humanize.ParseBytes(groups["uploadedSize"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse uploaded size string", field.M{"uploadedSize": groups["uploadedSize"]})
		return nil
	}

	var estimatedSizeBytes uint64
	var progressPercent float64
	estimationCompleted := len(groups["estimatedSize"]) != 0 || len(groups["estimatedProgress"]) != 0

	if estimationCompleted {
		estimatedSizeBytes, err = humanize.ParseBytes(groups["estimatedSize"])
		if err != nil {
			log.WithError(err).Print("Skipping entry due to inability to parse estimated size string", field.M{"estimatedSize": groups["estimatedSize"]})
			return nil
		}

		progressPercent, err = strconv.ParseFloat(groups["estimatedProgress"], 64)
		if err != nil {
			log.WithError(err).Print("Skipping entry due to inability to parse progress percent string", field.M{"estimatedProgress": groups["estimatedProgress"]})
			return nil
		}
	}

	if isFinalResult {
		progressPercent = 100
	} else if progressPercent >= 100 {
		// It may happen that kopia reports progress of 100 or higher without actual completing the task.
		// This can occur due to inaccurate estimation.
		// In such case, we will return the progress as 99% to avoid confusion.
		progressPercent = 99
	}

	return &SnapshotCreateStats{
		FilesHashed:     int64(numHashed),
		SizeHashedB:     int64(hashedSizeBytes),
		FilesCached:     int64(numCached),
		SizeCachedB:     int64(cachedSizeBytes),
		SizeUploadedB:   int64(uploadedSizeBytes),
		SizeEstimatedB:  int64(estimatedSizeBytes),
		ProgressPercent: int64(progressPercent),
	}
}

// RestoreStats is a container for stats parsed from the output of a
// `kopia restore` command.
type RestoreStats struct {
	FilesProcessed  int64
	SizeProcessedB  int64
	FilesTotal      int64
	SizeTotalB      int64
	ProgressPercent int64
}

// RestoreStatsFromRestoreOutput parses the output of a `kopia restore`
// line-by-line in search of progress statistics.
// It returns nil if no statistics are found, or the most recent statistic
// if multiple are encountered.
func RestoreStatsFromRestoreOutput(
	restoreStderrOutput string,
) (stats *RestoreStats) {
	if restoreStderrOutput == "" {
		return nil
	}
	logs := regexp.MustCompile("[\r\n]").Split(restoreStderrOutput, -1)

	for _, l := range logs {
		lineStats := parseKopiaRestoreProgressLine(l)
		if lineStats != nil {
			stats = lineStats
		}
	}

	return stats
}

// parseKopiaRestoreProgressLine parses restore stats from the output log line,
// which is expected to be in the following format:
// Processed 5 (1.4 GB) of 5 (1.8 GB) 291.1 MB/s (75.2%) remaining 1s.
func parseKopiaRestoreProgressLine(line string) (stats *RestoreStats) {
	match := kopiaRestorePattern.FindStringSubmatch(line)
	if len(match) < 8 {
		return nil
	}

	groups := make(map[string]string)
	for i, name := range kopiaRestorePattern.SubexpNames() {
		if i != 0 && name != "" {
			groups[name] = match[i]
		}
	}

	processedCount, err := strconv.Atoi(groups["processedCount"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse number of processed files", field.M{"processedCount": groups["processedCount"]})
		return nil
	}

	processedSize, err := humanize.ParseBytes(groups["processedSize"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse amount of processed bytes", field.M{"processedSize": groups["processedSize"]})
		return nil
	}

	totalCount, err := strconv.Atoi(groups["totalCount"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse expected number of files", field.M{"totalCount": groups["totalCount"]})
		return nil
	}

	totalSize, err := humanize.ParseBytes(groups["totalSize"])
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse expected amount of bytes", field.M{"totalSize": groups["totalSize"]})
		return nil
	}

	progressPercent, err := strconv.ParseFloat(groups["percentage"], 64)
	if err != nil {
		log.WithError(err).Print("Skipping entry due to inability to parse progress percent string", field.M{"progressPercent": groups["progressPercent"]})
		return nil
	}

	if progressPercent >= 100 {
		// It may happen that kopia reports progress of 100 or higher without actually
		// completing the task. This can occur due to inaccurate estimation.
		// In such cases, we will return the progress as 99% to avoid confusion.
		progressPercent = 99
	}

	return &RestoreStats{
		FilesProcessed:  int64(processedCount),
		SizeProcessedB:  int64(processedSize),
		FilesTotal:      int64(totalCount),
		SizeTotalB:      int64(totalSize),
		ProgressPercent: int64(progressPercent),
	}
}

// RepoSizeStatsFromBlobStatsRaw takes a string as input, interprets it as a kopia blob stats
// output in an expected format (Contains the line "Total: <size>"), and returns the integer
// size in bytes or an error if parsing is unsuccessful.
func RepoSizeStatsFromBlobStatsRaw(blobStats string) (phySizeTotal int64, blobCount int, err error) {
	if blobStats == "" {
		return phySizeTotal, blobCount, errkit.New("received empty blob stats string")
	}

	sizePattern := regexp.MustCompile(repoTotalSizeFromBlobStatsRegEx)
	countPattern := regexp.MustCompile(repoCountFromBlobStatsRegEx)

	var countStr, sizeStr string

	for _, l := range strings.Split(blobStats, "\n") {
		if countStr == "" {
			countMatch := countPattern.FindStringSubmatch(l)
			if len(countMatch) >= 2 {
				countStr = countMatch[1]
			}
		}

		if sizeStr == "" {
			sizeMatch := sizePattern.FindStringSubmatch(l)
			if len(sizeMatch) >= 2 {
				sizeStr = sizeMatch[1]
			}
		}

		if !(countStr == "" || sizeStr == "") {
			// Both strings have been matched
			break
		}
	}

	if countStr == "" {
		return phySizeTotal, blobCount, errkit.New("could not find count field in the blob stats")
	}

	if sizeStr == "" {
		return phySizeTotal, blobCount, errkit.New("could not find size field in the blob stats")
	}

	countVal, err := strconv.Atoi(countStr)
	if err != nil {
		return phySizeTotal, blobCount, errkit.Wrap(err, fmt.Sprintf("unable to convert parsed count value %s", countStr))
	}

	sizeValBytes, err := strconv.Atoi(sizeStr)
	if err != nil {
		return phySizeTotal, blobCount, errkit.Wrap(err, fmt.Sprintf("unable to convert parsed size value %s", countStr))
	}

	return int64(sizeValBytes), countVal, nil
}

func IsEqualSnapshotCreateStats(a, b *SnapshotCreateStats) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.FilesHashed == b.FilesHashed &&
		a.SizeHashedB == b.SizeHashedB &&
		a.FilesCached == b.FilesCached &&
		a.SizeCachedB == b.SizeCachedB &&
		a.SizeUploadedB == b.SizeUploadedB &&
		a.SizeEstimatedB == b.SizeEstimatedB &&
		a.ProgressPercent == b.ProgressPercent
}

var ANSIEscapeCode = regexp.MustCompile(`\x1b[^m]*?m`)
var kopiaErrorPattern = regexp.MustCompile(`(?:ERROR\s+|.*\<ERROR\>\s*|error\s+)(.*)`)

// ErrorsFromOutput parses the output of a kopia and returns an error, if found
func ErrorsFromOutput(output string) []error {
	if output == "" {
		return nil
	}

	var err []error

	lines := regexp.MustCompile("[\r\n]").Split(output, -1)
	for _, l := range lines {
		clean := ANSIEscapeCode.ReplaceAllString(l, "") // Strip all ANSI escape codes from line
		match := kopiaErrorPattern.FindAllStringSubmatch(clean, 1)
		if len(match) > 0 {
			err = append(err, errkit.New(match[0][1]))
		}
	}

	return err
}

// ParsePolicyShow parses the output of a kopia policy show command.
func ParsePolicyShow(output string) (policy.Policy, error) {
	policy := policy.Policy{}

	if err := json.Unmarshal([]byte(output), &policy); err != nil {
		return policy, errkit.Wrap(err, "Failed to unmarshal snapshot manifest list")
	}

	return policy, nil
}
