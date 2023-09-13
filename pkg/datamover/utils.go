// Copyright 2023 The Kanister Authors.
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

package datamover

import (
	"context"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	usePipeParam = `-`
)

func targetWriter(target string) (io.Writer, error) {
	if target != usePipeParam {
		return os.OpenFile(target, os.O_RDWR|os.O_CREATE, 0755)
	}
	return os.Stdout, nil
}

func locationPull(ctx context.Context, p *param.Profile, path string, target io.Writer) error {
	return location.Read(ctx, target, *p, path)
}

// kopiaLocationPull pulls the data from a kopia snapshot into the given target
func kopiaLocationPull(ctx context.Context, backupID, path, targetPath, password string) error {
	switch targetPath {
	case usePipeParam:
		return snapshot.Read(ctx, os.Stdout, backupID, path, password)
	default:
		return snapshot.ReadFile(ctx, backupID, targetPath, password)
	}
}

// kopiaLocationPush pushes the data from the source using a kopia snapshot
func kopiaLocationPush(ctx context.Context, path, outputName, sourcePath, password string) (*snapshot.SnapshotInfo, error) {
	var snapInfo *snapshot.SnapshotInfo
	var err error
	switch sourcePath {
	case usePipeParam:
		snapInfo, err = snapshot.Write(ctx, os.Stdin, path, password)
	default:
		snapInfo, err = snapshot.WriteFile(ctx, path, sourcePath, password)
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to push data using kopia")
	}
	snapInfoJSON, err := snapshot.MarshalKopiaSnapshot(snapInfo)
	if err != nil {
		return nil, err
	}

	return snapInfo, output.PrintOutput(outputName, snapInfoJSON)
}

func sourceReader(source string) (io.Reader, error) {
	if source != usePipeParam {
		return os.Open(source)
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe file stdin")
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("Stdin must be piped when the source parameter is \"-\"")
	}
	return os.Stdin, nil
}

func locationPush(ctx context.Context, p *param.Profile, path string, source io.Reader) error {
	return location.Write(ctx, source, *p, path)
}

// kopiaLocationDelete deletes the kopia snapshot with given backupID
func kopiaLocationDelete(ctx context.Context, backupID, path, password string) error {
	return snapshot.Delete(ctx, backupID, path, password)
}

func locationDelete(ctx context.Context, p *param.Profile, path string) error {
	return location.Delete(ctx, *p, path)
}
