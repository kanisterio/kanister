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

	"github.com/kanisterio/errkit"

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
)

// Check that RepositoryServer implements DataMover interface
var _ DataMover = (*RepositoryServer)(nil)

type RepositoryServer struct {
	outputName       string
	repositoryServer *param.RepositoryServer
	snapJSON         string
	hostName         string
}

func (rs *RepositoryServer) Pull(ctx context.Context, sourcePath, destinationPath string) error {
	kopiaSnap, err := rs.unmarshalKopiaSnapshot()
	if err != nil {
		return err
	}
	password, err := rs.connectToKopiaRepositoryServer(ctx, repository.ReadOnlyAccess)
	if err != nil {
		return err
	}
	return kopiaLocationPull(ctx, kopiaSnap.ID, destinationPath, sourcePath, password)
}

func (rs *RepositoryServer) Push(ctx context.Context, sourcePath, destinationPath string) error {
	password, err := rs.connectToKopiaRepositoryServer(ctx, repository.WriteAccess)
	if err != nil {
		return err
	}
	_, err = kopiaLocationPush(ctx, destinationPath, rs.outputName, sourcePath, password)
	return err
}

func (rs *RepositoryServer) Delete(ctx context.Context, destinationPath string) error {
	kopiaSnap, err := rs.unmarshalKopiaSnapshot()
	if err != nil {
		return err
	}
	password, err := rs.connectToKopiaRepositoryServer(ctx, repository.WriteAccess)
	if err != nil {
		return err
	}
	return kopiaLocationDelete(ctx, kopiaSnap.ID, destinationPath, password)
}

func (rs *RepositoryServer) connectToKopiaRepositoryServer(ctx context.Context, accessMode repository.AccessMode) (string, error) {
	hostname, userPassphrase, err := rs.hostnameAndUserPassphrase()
	if err != nil {
		return "", errkit.Wrap(err, "Error Retrieving Hostname and User Passphrase from Repository Server")
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
		accessMode,
	)
}

func (rs *RepositoryServer) unmarshalKopiaSnapshot() (*snapshot.SnapshotInfo, error) {
	if rs.snapJSON == "" {
		return nil, errkit.New("kopia snapshot information is required to manage data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(rs.snapJSON)
	if err != nil {
		return nil, err
	}
	return &kopiaSnap, nil
}

func (rs *RepositoryServer) hostnameAndUserPassphrase() (string, string, error) {
	var hostname, userPassphrase string
	userAccessMap := make(map[string]string)
	for key, value := range rs.repositoryServer.Credentials.ServerUserAccess.Data {
		userAccessMap[key] = string(value)
	}

	// if hostname is not provided, use the random hostname in the map as default
	for key, val := range userAccessMap {
		hostname = key
		userPassphrase = val
		break
	}
	// check if hostname is provided in the repository server
	if rs.hostName != "" {
		err := rs.checkHostnameExistsInUserAccessMap(userAccessMap)
		if err != nil {
			return "", "", err
		}
		hostname = rs.hostName
		userPassphrase = userAccessMap[hostname]
	}

	return hostname, string(userPassphrase), nil
}

func (rs *RepositoryServer) checkHostnameExistsInUserAccessMap(userAccessMap map[string]string) error {
	// check if hostname is provided in the repository server exists in the user access map
	if _, ok := userAccessMap[rs.hostName]; !ok {
		return errkit.New("hostname provided in the repository server does not exist in the user access map")
	}
	return nil
}

func NewRepositoryServerDataMover(repoServer *param.RepositoryServer, outputName, snapJSON, userHostname string) *RepositoryServer {
	return &RepositoryServer{
		outputName:       outputName,
		repositoryServer: repoServer,
		snapJSON:         snapJSON,
		hostName:         userHostname,
	}
}
