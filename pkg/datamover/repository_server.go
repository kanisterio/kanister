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

	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
)

type RepositoryServer struct {
	OutputName       string
	RepositoryServer *param.RepositoryServer
	SnapJSON         string
}

func (rs *RepositoryServer) Pull(sourcePath, destinationPath string) error {
	ctx := context.Background()
	if rs.SnapJSON == "" {
		return errors.New("kopia snapshot information is required to pull data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(rs.SnapJSON)
	if err != nil {
		return err
	}
	password, err := connectToKopiaRepositoryServer(ctx, rs.RepositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationPull(ctx, kopiaSnap.ID, destinationPath, sourcePath, password)
}

func (rs *RepositoryServer) Push(sourcePath, destinationPath string) error {
	ctx := context.Background()
	password, err := connectToKopiaRepositoryServer(ctx, rs.RepositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationPush(ctx, destinationPath, rs.OutputName, sourcePath, password)
}

func (rs *RepositoryServer) Delete(destinationPath string) error {
	ctx := context.Background()
	if rs.SnapJSON == "" {
		return errors.New("kopia snapshot information is required to pull data using kopia")
	}
	kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(rs.SnapJSON)
	if err != nil {
		return err
	}
	password, err := connectToKopiaRepositoryServer(ctx, rs.RepositoryServer)
	if err != nil {
		return err
	}
	return kopiaLocationDelete(ctx, kopiaSnap.ID, destinationPath, password)
}
