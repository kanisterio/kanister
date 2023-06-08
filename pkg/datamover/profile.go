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
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kopia/repository"

	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
)

type Profile struct {
	outputName string
	profile    *param.Profile
	snapJSON   string
}

func (p *Profile) Pull(ctx context.Context, sourcePath, destinationPath string) error {
	if p.profile.Location.Type == crv1alpha1.LocationTypeKopia {
		if p.snapJSON == "" {
			return errors.New("kopia snapshot information is required to pull data using kopia")
		}
		kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(p.snapJSON)
		if err != nil {
			return err
		}
		if err = p.connectToKopiaRepositoryServer(ctx); err != nil {
			return err
		}
		return kopiaLocationPull(ctx, kopiaSnap.ID, destinationPath, sourcePath, p.profile.Credential.KopiaServerSecret.Password)
	}
	target, err := targetWriter(sourcePath)
	if err != nil {
		return err
	}
	return locationPull(ctx, p.profile, destinationPath, target)
}

func (p *Profile) Push(ctx context.Context, sourcePath, destinationPath string) error {
	if p.profile.Location.Type == crv1alpha1.LocationTypeKopia {
		if err := p.connectToKopiaRepositoryServer(ctx); err != nil {
			return err
		}
		return kopiaLocationPush(ctx, destinationPath, p.outputName, sourcePath, p.profile.Credential.KopiaServerSecret.Password)
	}
	source, err := sourceReader(sourcePath)
	if err != nil {
		return err
	}
	return locationPush(ctx, p.profile, destinationPath, source)
}

func (p *Profile) Delete(ctx context.Context, destinationPath string) error {
	if p.profile.Location.Type == crv1alpha1.LocationTypeKopia {
		if p.snapJSON == "" {
			return errors.New("kopia snapshot information is required to delete data using kopia")
		}
		kopiaSnap, err := snapshot.UnmarshalKopiaSnapshot(p.snapJSON)
		if err != nil {
			return err
		}
		if err = p.connectToKopiaRepositoryServer(ctx); err != nil {
			return err
		}
		return kopiaLocationDelete(ctx, kopiaSnap.ID, destinationPath, p.profile.Credential.KopiaServerSecret.Password)
	}
	return locationDelete(ctx, p.profile, destinationPath)
}

func (p *Profile) connectToKopiaRepositoryServer(ctx context.Context) error {
	contentCacheSize := kopia.GetDataStoreGeneralContentCacheSize(p.profile.Credential.KopiaServerSecret.ConnectOptions)
	metadataCacheSize := kopia.GetDataStoreGeneralMetadataCacheSize(p.profile.Credential.KopiaServerSecret.ConnectOptions)
	return repository.ConnectToAPIServer(
		ctx,
		p.profile.Credential.KopiaServerSecret.Cert,
		p.profile.Credential.KopiaServerSecret.Password,
		p.profile.Credential.KopiaServerSecret.Hostname,
		p.profile.Location.Endpoint,
		p.profile.Credential.KopiaServerSecret.Username,
		contentCacheSize,
		metadataCacheSize,
	)
}

func NewProfileDataMover(profile *param.Profile, outputName, snapJson string) *Profile {
	return &Profile{
		outputName: outputName,
		profile:    profile,
		snapJSON:   snapJson,
	}
}
