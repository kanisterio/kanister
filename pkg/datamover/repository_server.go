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
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia"
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
	kopiaSnap, err := rs.unmarshalKopiaSnapshot()
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
	kopiaSnap, err := rs.unmarshalKopiaSnapshot()
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
	hostname, userPassphrase, err := rs.hostnameAndUserPassphrase()
	if err != nil {
		return "", errors.Wrap(err, "Error Retrieving Hostname and User Passphrase from Repository Server")
	}

	return userPassphrase, repository.ConnectToAPIServer(
		ctx,
		string(rs.repositoryServer.Credentials.ServerTLS.Data[kopia.TLSCertificateKey]),
		userPassphrase,
		hostname,
		rs.repositoryServer.Address,
		rs.repositoryServer.Username,
		rs.repositoryServer.ContentCacheMB,
		rs.repositoryServer.MetadataCacheMB,
	)
}

func (rs *repositoryServer) unmarshalKopiaSnapshot() (*snapshot.SnapshotInfo, error) {
	if rs.snapJSON == "" {
		return nil, errors.New("kopia snapshot information is required to manage data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(rs.snapJSON)
	if err != nil {
		return nil, err
	}
	return &kopiaSnap, nil
}

func (rs *repositoryServer) hostnameAndUserPassphrase() (string, string, error) {
	userCredsJSON, err := json.Marshal(rs.repositoryServer.Credentials.ServerUserAccess.Data)
	if err != nil {
		return "", "", errors.Wrap(err, "Error Unmarshalling User Credentials")
	}
	var userAccessMap map[string]string
	if err := json.Unmarshal([]byte(string(userCredsJSON)), &userAccessMap); err != nil {
		return "", "", errors.Wrap(err, "Failed to unmarshal User Credentials Data")
	}

	var userPassPhrase string
	var hostName string
	for key, val := range userAccessMap {
		hostName = key
		userPassPhrase = val
	}
	decodedUserPassPhrase, err := base64.StdEncoding.DecodeString(userPassPhrase)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to Decode User Passphrase")
	}
	return hostName, string(decodedUserPassPhrase), nil
}

func NewRepositoryServerDataMover(repoServer *param.RepositoryServer, outputName, snapJson string) *repositoryServer {
	return &repositoryServer{
		outputName:       outputName,
		repositoryServer: repoServer,
		snapJSON:         snapJson,
	}
}
