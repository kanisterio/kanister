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

package v1alpha1

import (
	"testing"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const repoServerSpec = `
apiVersion: cr.kanister.io/v1alpha1
kind: RepositoryServer
metadata:
  name: test-kopia-repo-server
  namespace: kanister
spec:
  storage:
    secretRef:
      name: test-s3-location
      namespace: kanister
    credentialSecretRef:
      name: test-s3-creds
      namespace: kanister
  repository:
    rootPath: /test-repo-controller/
    passwordSecretRef:
      name: test-repo-pass 
      namespace: kanister
    username: test-repository-user
    hostname: localhost
  server:
    adminSecretRef:
      name: test-repository-admin-user
      namespace: kanister
    tlsSecretRef:
      name: test-repository-server-tls-cert
      namespace: kanister
    userAccess:
      userAccessSecretRef:
        name: test-repository-server-user-access
        namespace: kanister
      username: test-kanister-user
`

func TestRepositoryServer(t *testing.T) { check.TestingT(t) }

func (s *TypesSuite) TestRepositoryServerDecode(c *check.C) {
	rs, err := getRepositoryServerFromSpec([]byte(repoServerSpec))
	c.Assert(err, check.IsNil)
	c.Assert(rs, check.NotNil)
	c.Assert(rs.Spec.Storage.SecretRef.Name, check.Equals, "test-s3-location")
	c.Assert(rs.Spec.Storage.SecretRef.Namespace, check.Equals, "kanister")
	c.Assert(rs.Spec.Storage.CredentialSecretRef.Name, check.Equals, "test-s3-creds")
	c.Assert(rs.Spec.Storage.CredentialSecretRef.Namespace, check.Equals, "kanister")
	c.Assert(rs.Spec.Repository.RootPath, check.Equals, "/test-repo-controller/")
	c.Assert(rs.Spec.Repository.PasswordSecretRef.Name, check.Equals, "test-repo-pass")
	c.Assert(rs.Spec.Repository.PasswordSecretRef.Namespace, check.Equals, "kanister")
	c.Assert(rs.Spec.Repository.Username, check.Equals, "test-repository-user")
	c.Assert(rs.Spec.Repository.Hostname, check.Equals, "localhost")
	c.Assert(rs.Spec.Server.AdminSecretRef.Name, check.Equals, "test-repository-admin-user")
	c.Assert(rs.Spec.Server.AdminSecretRef.Namespace, check.Equals, "kanister")
	c.Assert(rs.Spec.Server.TLSSecretRef.Name, check.Equals, "test-repository-server-tls-cert")
	c.Assert(rs.Spec.Server.TLSSecretRef.Namespace, check.Equals, "kanister")
	c.Assert(rs.Spec.Server.UserAccess.UserAccessSecretRef.Name, check.Equals, "test-repository-server-user-access")
	c.Assert(rs.Spec.Server.UserAccess.UserAccessSecretRef.Namespace, check.Equals, "kanister")
	c.Assert(rs.Spec.Server.UserAccess.Username, check.Equals, "test-kanister-user")
}

func getRepositoryServerFromSpec(spec []byte) (*RepositoryServer, error) {
	repositoryServer := &RepositoryServer{}
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, repositoryServer); err != nil {
		return nil, errkit.Wrap(err, "Failed to decode RepositoryServer")
	}
	return repositoryServer, nil
}
