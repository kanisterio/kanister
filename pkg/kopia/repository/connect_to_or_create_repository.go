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

package repository

import (
	"github.com/go-openapi/strfmt"
	"k8s.io/client-go/kubernetes"

	kerrors "github.com/kanisterio/kanister/pkg/errors"
	"github.com/kanisterio/kanister/pkg/kopia"
)

// ConnectToOrCreateKopiaRepository connects to a kopia repository if present or creates if not already present
func ConnectToOrCreateKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	repoPathPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	prof kopia.Profile,
) error {
	err := ConnectToKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		repoPathPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		prof,
		strfmt.DateTime{},
	)
	switch {
	case err == nil:
		// If repository connect was successful, we're done!
		return nil
	case kopia.IsInvalidPasswordError(err):
		// If connect failed due to invalid password, no need to attempt creation
		return err
	}

	// Create a new repository
	err = CreateKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		repoPathPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		prof,
	)

	if err == nil {
		// Successfully created repository, we're done!
		return nil
	}

	// Creation failed. Repository may already exist.
	// Attempt connecting to it.
	// Multiple workers attempting to back up volumes from the
	// same app may race when trying to create the repository if
	// it doesn't yet exist. If this thread initially fails to connect
	// to the repo, then also fails to create a repo, it can try to
	// connect again under the assumption that the repo may have been
	// created by a parallel worker. No harm done if the connect still fails.
	connectErr := ConnectToKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		repoPathPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		prof,
		strfmt.DateTime{},
	)

	// Connected successfully after all
	if connectErr == nil {
		return nil
	}

	err = kerrors.Append(err, connectErr)
	return err
}
