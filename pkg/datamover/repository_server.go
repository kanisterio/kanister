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
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
)

// Check that RepositoryServer implements DataMover interface
var _ DataMover = (*repositoryServer)(nil)

type repositoryServer struct {
	outputName       string
	repositoryServer *param.RepositoryServer
	snapJSON         string
}

func (rs *repositoryServer) Pull(ctx context.Context, sourcePath, destinationPath string) error {
	kopiaSnap, err := rs.unmarshalKopiaSnapshot(ctx)
	if err != nil {
		return err
	}
	password, err := rs.connectToKopiaRepositoryServer(ctx)
	if err != nil {
		return err
	}
	return kopiaLocationPull(ctx, kopiaSnap.ID, destinationPath, sourcePath, password)
}

func (rs *repositoryServer) Push(ctx context.Context, sourcePath, destinationPath string) error {
	password, err := rs.connectToKopiaRepositoryServer(ctx)
	if err != nil {
		return err
	}
	return kopiaLocationPush(ctx, destinationPath, rs.outputName, sourcePath, password)
}

func (rs *repositoryServer) Delete(ctx context.Context, destinationPath string) error {
	kopiaSnap, err := rs.unmarshalKopiaSnapshot(ctx)
	if err != nil {
		return err
	}
	password, err := rs.connectToKopiaRepositoryServer(ctx)
	if err != nil {
		return err
	}
	return kopiaLocationDelete(ctx, kopiaSnap.ID, destinationPath, password)
}

func (rs *repositoryServer) connectToKopiaRepositoryServer(ctx context.Context) (string, error) {
	hostname, userPassphrase, certData, err := secretsFromRepositoryServerCR(rs.repositoryServer)
	if err != nil {
		return "", errors.Wrap(err, "Error Retrieving Connection Data from Repository Server")
	}
	return userPassphrase, repository.ConnectToAPIServer(
		ctx,
		certData,
		userPassphrase,
		hostname,
		rs.repositoryServer.Address,
		rs.repositoryServer.Username,
		rs.repositoryServer.ContentCacheMB,
		rs.repositoryServer.MetadataCacheMB,
	)
}

func (rs *repositoryServer) unmarshalKopiaSnapshot(ctx context.Context) (*snapshot.SnapshotInfo, error) {
	if rs.snapJSON == "" {
		return nil, errors.New("kopia snapshot information is required to delete data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(rs.snapJSON)
	if err != nil {
		return nil, err
	}
	return &kopiaSnap, nil
}

func NewRepositoryServerDataMover(repoServer *param.RepositoryServer, outputName, snapJson string) *repositoryServer {
	return &repositoryServer{
		outputName:       outputName,
		repositoryServer: repoServer,
		snapJSON:         snapJson,
	}
}
