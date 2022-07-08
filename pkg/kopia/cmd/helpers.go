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

package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/utils"
)

type policyChanges map[string]string

func kopiaCacheArgs(args logsafe.Cmd, cacheDirectory string, contentCacheMB, metadataCacheMB int) logsafe.Cmd {
	args = args.AppendLoggableKV(cacheDirectoryFlag, cacheDirectory)
	args = args.AppendLoggableKV(contentCacheSizeMBFlag, strconv.Itoa(contentCacheMB))
	args = args.AppendLoggableKV(metadataCacheSizeMBFlag, strconv.Itoa(metadataCacheMB))
	return args
}

// GetCacheSizeSettingsForSnapshot returns the feature setting cache size values to be used
// for initializing repositories that will be performing general command workloads that benefit from
// cacheing metadata only.
func GetCacheSizeSettingsForSnapshot() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(kopia.DataStoreGeneralContentCacheSizeMBVarName, kopia.DefaultDataStoreGeneralContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(kopia.DataStoreGeneralMetadataCacheSizeMBVarName, kopia.DefaultDataStoreGeneralMetadataCacheSizeMB)
}

// GetCacheSizeSettingsForRestore returns the feature setting cache size values to be used
// for initializing repositories that will be performing restore workloads
func GetCacheSizeSettingsForRestore() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(kopia.DataStoreRestoreContentCacheSizeMBVarName, kopia.DefaultDataStoreRestoreContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(kopia.DataStoreRestoreMetadataCacheSizeMBVarName, kopia.DefaultDataStoreRestoreMetadataCacheSizeMB)
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
	snapID, rootID, err := kopia.SnapshotInfoFromSnapshotCreateOutput(snapCreateStdoutOutput)
	if err != nil {
		return nil, err
	}

	return &SnapshotCreateInfo{
		SnapshotID: snapID,
		RootID:     rootID,
		Stats:      SnapshotStatsFromSnapshotCreate(snapCreateStderrOutput),
	}, nil
}

// SnapshotCreateStats is a container for stats parsed from the output of a `kopia
// snapshot create` command.
type SnapshotCreateStats struct {
	FilesHashed   int64
	SizeHashedB   int64
	FilesCached   int64
	SizeCachedB   int64
	SizeUploadedB int64
}

// SnapshotStatsFromSnapshotCreate parses the output of a kopia snapshot
// create execution for a log of the stats for that execution.
func SnapshotStatsFromSnapshotCreate(snapCreateStderrOutput string) (stats *SnapshotCreateStats) {
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
	// }, nil
	pattern := regexp.MustCompile(`\*.+[^\d](\d+) hashed \(([^\)]+)\), (\d+) cached \(([^\)]+)\), uploaded ([^\)]+),.+`)
	for _, l := range logs {
		match := pattern.FindStringSubmatch(l)
		if match != nil && len(match) >= 6 {
			numHashedStr := match[1]
			hashedSizeHumanized := match[2]
			numCachedStr := match[3]
			cachedSizeHumanized := match[4]
			uploadedSizeHumanized := match[5]

			numHashed, err := strconv.Atoi(numHashedStr)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse number of hashed files", field.M{"numHashedStr": numHashedStr})
				continue
			}

			numCached, err := strconv.Atoi(numCachedStr)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse number of cached files", field.M{"numCachedStr": numCachedStr})
				continue
			}

			hashedSizeBytes, err := humanize.ParseBytes(hashedSizeHumanized)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse hashed size string", field.M{"hashedSizeHumanized": hashedSizeHumanized})
				continue
			}

			cachedSizeBytes, err := humanize.ParseBytes(cachedSizeHumanized)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse cached size string", field.M{"cachedSizeHumanized": cachedSizeHumanized})
				continue
			}

			uploadedSizeBytes, err := humanize.ParseBytes(uploadedSizeHumanized)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse uploaded size string", field.M{"uploadedSizeHumanized": uploadedSizeHumanized})
				continue
			}

			stats = &SnapshotCreateStats{
				FilesHashed:   int64(numHashed),
				SizeHashedB:   int64(hashedSizeBytes),
				FilesCached:   int64(numCached),
				SizeCachedB:   int64(cachedSizeBytes),
				SizeUploadedB: int64(uploadedSizeBytes),
			}
		}
	}

	if stats == nil {
		log.Error().Print("could not find well-formed stats in snapshot create output")
	}

	return stats
}

// RepoSizeStatsFromBlobStatsRaw takes a string as input, interprets it as a kopia blob stats
// output in an expected format (Contains the line "Total: <size>"), and returns the integer
// size in bytes or an error if parsing is unsuccessful.
func RepoSizeStatsFromBlobStatsRaw(blobStats string) (phySizeTotal int64, blobCount int, err error) {
	if blobStats == "" {
		return phySizeTotal, blobCount, errors.New("received empty blob stats string")
	}

	sizePattern := regexp.MustCompile(`Total: (\d+)$`)
	countPattern := regexp.MustCompile(`Count: (\d+)$`)

	var countStr, sizeStr string

	for _, l := range strings.Split(blobStats, "\n") {
		if countStr == "" {
			countMatch := countPattern.FindStringSubmatch(l)
			if countMatch != nil && len(countMatch) >= 2 {
				countStr = countMatch[1]
			}
		}

		if sizeStr == "" {
			sizeMatch := sizePattern.FindStringSubmatch(l)
			if sizeMatch != nil && len(sizeMatch) >= 2 {
				sizeStr = sizeMatch[1]
			}
		}

		if !(countStr == "" || sizeStr == "") {
			// Both strings have been matched
			break
		}
	}

	if countStr == "" {
		return phySizeTotal, blobCount, errors.New("could not find count field in the blob stats")
	}

	if sizeStr == "" {
		return phySizeTotal, blobCount, errors.New("could not find size field in the blob stats")
	}

	countVal, err := strconv.Atoi(countStr)
	if err != nil {
		return phySizeTotal, blobCount, errors.Wrap(err, fmt.Sprintf("unable to convert parsed count value %s", countStr))
	}

	sizeValBytes, err := strconv.Atoi(sizeStr)
	if err != nil {
		return phySizeTotal, blobCount, errors.Wrap(err, fmt.Sprintf("unable to convert parsed size value %s", countStr))
	}

	return int64(sizeValBytes), countVal, nil
}
