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

package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/kanisterio/kanister/pkg/kopialib/storage"
	"github.com/kopia/kopia/repo"
)

type Repository struct {
	st          storage.Storage
	password    string
	configFile  string
	storageType storage.StorageType
}

func (r *Repository) Create(opts *repo.NewRepositoryOptions) (err error) {
	storage, err := r.st.Connect()
	if err != nil {
		return err
	}
	return repo.Initialize(context.Background(), storage, opts, r.password)
}

func (r *Repository) Connect(opts *repo.ConnectOptions) (err error) {
	storage, err := r.st.Connect()
	if err != nil {
		return err
	}
	return repo.Connect(context.Background(), r.configFile, storage, r.password, opts)
}

func (r *Repository) ConnectUsingFile(opts *repo.ConnectOptions) error {
	repoConfig := repositoryConfigFileName(r.configFile)
	if _, err := os.Stat(repoConfig); os.IsNotExist(err) {
		return errors.New("failed find kopia configuration file")
	}

	_, err := repo.Open(context.Background(), repoConfig, r.password, &repo.Options{})
	return err
}

func repositoryConfigFileName(configFile string) string {
	if configFile != "" {
		return configFile
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "kopia", "repository.config")
}
