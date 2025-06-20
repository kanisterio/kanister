// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repositoryserver

import (
	corev1 "k8s.io/api/core/v1"
)

var _ Secret = &FileStore{}

type FileStore struct {
	storageLocation *corev1.Secret
}

func NewFileStoreLocation(secret *corev1.Secret) *FileStore {
	return &FileStore{
		storageLocation: secret,
	}
}

func (l *FileStore) Validate() error {
	// Currently empty since all the fields are optional
	return nil
}
