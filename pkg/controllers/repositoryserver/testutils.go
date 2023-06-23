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

package repositoryserver

import (
	"os"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/secrets"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const (
	defaultKopiaRepositoryPath                 = "kopia-repo-controller-test"
	defaulKopiaRepositoryServerAdminUser       = "admin@test"
	defaultKopiaRepositoryServerAdminPassword  = "admin1234"
	defaultKopiaRepositoryServerHost           = "localhost"
	defaultKopiaRepositoryPassword             = "test1234"
	defaultKopiaRepositoryUser                 = "repository-user"
	defaultKopiaRepositoryServerAccessUser     = "kanister-user"
	defaultKopiaRepositoryServerAccessPassword = "test1234"
	defaultKanisterNamespace                   = "kanister"
	defaultKopiaRepositoryServerContainer      = "repo-server-container"
	pathKey                                    = "path"
)

func getDefaultS3StorageCreds() map[string][]byte {
	key := os.Getenv(awsconfig.AccessKeyID)
	val := os.Getenv(awsconfig.SecretAccessKey)

	return map[string][]byte{
		secrets.AWSAccessKeyID:     []byte(key),
		secrets.AWSSecretAccessKey: []byte(val),
	}
}

func getDefaultS3CompliantStorageLocation() map[string][]byte {
	return map[string][]byte{
		reposerver.TypeKey:     []byte(crv1alpha1.LocationTypeS3Compliant),
		reposerver.BucketKey:   []byte(testutil.TestS3BucketName),
		pathKey:                []byte(defaultKopiaRepositoryPath),
		reposerver.RegionKey:   []byte(testutil.TestS3Region),
		reposerver.EndpointKey: []byte(os.Getenv("LOCATION_ENDPOINT")),
	}
}
