// Copyright 2020 The Kanister Authors.
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

// Package stream provides functionality to stream data to an object store
// by reading it from a given endpoint without loading it into memory.
package stream

import (
	"context"
	"net/http"
	"os"

	"github.com/kanisterio/errkit"
	"github.com/kopia/kopia/fs"
	"github.com/kopia/kopia/fs/virtualfs"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/upload"

	"github.com/kanisterio/kanister/pkg/kopia/repository"
	kansnapshot "github.com/kanisterio/kanister/pkg/kopia/snapshot"
)

const (
	snapshotDescription             = "Snapshot created by kando stream push"
	defaultPermissions  os.FileMode = 0777
)

// Push streams data to object store by reading it from the given endpoint using Kopia's streaming capabilities
func Push(ctx context.Context, configFile, dirPath, filePath, password, sourceEndpoint string) error {
	rep, err := repository.Open(ctx, configFile, password, "kanister stream push")
	if err != nil {
		return errkit.Wrap(err, "Failed to open kopia repository")
	}

	// Create an HTTP client to stream from the source endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", sourceEndpoint, nil)
	if err != nil {
		return errkit.Wrap(err, "Failed to create HTTP request")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errkit.Wrap(err, "Failed to make HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errkit.New("HTTP request failed", "status", resp.Status)
	}

	// Use Kopia's streaming virtualfs which doesn't load data into memory
	// This creates a virtual directory tree rooted at dirPath 
	// with a streaming file as the child entry
	rootDir := virtualfs.NewStaticDirectory(dirPath, []fs.Entry{
		virtualfs.StreamingFileFromReader(filePath, resp.Body),
	})

	// Setup kopia uploader
	u := upload.NewUploader(rep)
	// Fail full snapshot if errors are encountered during upload
	u.FailFast = true

	// Populate the source info with source path and file
	sourceInfo := snapshot.SourceInfo{
		UserName: rep.ClientOptions().Username,
		Host:     rep.ClientOptions().Hostname,
		Path:     dirPath,
	}

	// Create a kopia snapshot
	_, _, err = kansnapshot.SnapshotSource(ctx, rep, u, sourceInfo, rootDir, snapshotDescription)
	return errkit.Wrap(err, "Failed to create kopia snapshot")
}
