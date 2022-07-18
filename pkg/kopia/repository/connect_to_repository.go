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
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/cmd"
	"github.com/kanisterio/kanister/pkg/kube"
)

// ConnectToKopiaRepository connects to an already existing kopia repository
func ConnectToKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	prof kopia.Profile,
	pointInTimeConnection strfmt.DateTime,
) error {
	cmd, err := kopiacmd.RepositoryConnect(
		prof,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		pointInTimeConnection,
	)
	if err != nil {
		return errors.Wrap(err, "Failed to generate repository connect command")
	}

	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(stderr, kopia.ErrInvalidPassword.Error()):
		err = errors.WithMessage(err, kopia.ErrInvalidPassword.Error())
	case err != nil && strings.Contains(err.Error(), kopia.ErrCodeOutOfMemory.Error()):
		err = errors.WithMessage(err, kopia.ErrOutOfMemory.Error())
	case strings.Contains(stderr, kopia.ErrAccessDenied.Error()):
		err = errors.WithMessage(err, kopia.ErrAccessDenied.Error())
	case kopia.RepoNotInitialized(stderr):
		err = errors.WithMessage(err, kopia.ErrRepoNotFound.Error())
	}
	return errors.Wrap(err, "Failed to connect to the backup repository")
}
