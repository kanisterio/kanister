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
	"regexp"
	"strings"

	"github.com/kopia/kopia/repo/manifest"
	"github.com/kopia/kopia/snapshot"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	pathKey       = "path"
	typeKey       = "type"
	snapshotValue = "snapshot"
)

// SnapshotIDsFromSnapshot extracts root ID of a snapshot from the logs
func SnapshotIDsFromSnapshot(output string) (snapID, rootID string, err error) {
	if output == "" {
		return snapID, rootID, errors.New("Received empty output")
	}

	logs := regexp.MustCompile("[\r\n]").Split(output, -1)
	pattern := regexp.MustCompile(`Created snapshot with root ([^\s]+) and ID ([^\s]+).*$`)
	for _, l := range logs {
		// Log should contain "Created snapshot with root ABC and ID XYZ..."
		match := pattern.FindAllStringSubmatch(l, 1)
		if len(match) > 0 && len(match[0]) > 2 {
			snapID = match[0][2]
			rootID = match[0][1]
			return
		}
	}
	return snapID, rootID, errors.New("Failed to find Root ID from output")
}

// LatestSnapshotInfoFromManifestList returns snapshot ID and backup path of the latest snapshot from `manifests list` output
func LatestSnapshotInfoFromManifestList(output string) (string, string, error) {
	manifestList := []manifest.EntryMetadata{}
	snapID := ""
	backupPath := ""

	err := json.Unmarshal([]byte(output), &manifestList)
	if err != nil {
		return snapID, backupPath, errors.Wrap(err, "Failed to unmarshal manifest list")
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
		return "", "", errors.New("Failed to get latest snapshot ID from manifest list")
	}
	if backupPath == "" {
		return "", "", errors.New("Failed to get latest snapshot backup path from manifest list")
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
		if snapManifest != nil {
			snapID = string(snapManifest.ID)
			if snapManifest.RootEntry != nil {
				rootID = string(snapManifest.RootEntry.ObjectID)
			}
		}
	}
	if snapID == "" {
		return "", "", errors.New("Failed to get snapshot ID from create snapshot output")
	}
	if rootID == "" {
		return "", "", errors.New("Failed to get root ID from create snapshot output")
	}
	return snapID, rootID, nil
}

// SnapSizeStatsFromSnapListAll returns a list of snapshot logical sizes assuming the input string
// is formatted as the output of a kopia snapshot list --all command.
func SnapSizeStatsFromSnapListAll(output string) (totalSizeB int64, numSnapshots int, err error) {
	if output == "" {
		return 0, 0, errors.New("Received empty output")
	}

	snapList, err := parseSnapshotManifestList(output)
	if err != nil {
		return 0, 0, errors.Wrap(err, "Parsing snapshot list output as snapshot manifest list")
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

func parseSnapshotManifestList(output string) ([]*snapshot.Manifest, error) {
	snapInfoList := []*snapshot.Manifest{}

	if err := json.Unmarshal([]byte(output), &snapInfoList); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal snapshot manifest list")
	}

	return snapInfoList, nil
}
