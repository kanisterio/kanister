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
	"fmt"
	"os/exec"
	"path/filepath"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type RepositoryServerSuite struct {
	namespace        *corev1.Namespace
	pod              *corev1.Pod
	service          *corev1.Service
	ctx              context.Context
	cli              kubernetes.Interface
	s3Creds          *corev1.Secret
	s3Location       *corev1.Secret
	tlsSecret        *corev1.Secret
	userAccessSecret *corev1.Secret
	repoServer       *param.RepositoryServer
	user             string
}

var _ = Suite(&RepositoryServerSuite{})

func (rss *RepositoryServerSuite) SetUpSuite(c *C) {
	// Set Context as Background
	rss.ctx = context.Background()

	// Set Repository Server Test User
	rss.user = fmt.Sprintf("%s%s", testKopiaRepoServerUsername, rand.String(5))

	// Get Kubernetes Client
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	rss.cli = cli

	// Create Test Namespace
	rss.namespace, err = createRepositoryServerTestNamespace(rss.ctx, rss.cli)
	c.Assert(err, IsNil)

	// Configure and Create Test S3 Credential and Location Secrets
	storageCredentialsSecretData := testutil.GetDefaultS3StorageCreds()
	storageStorageLocationSecretData := testutil.GetDefaultS3CompliantStorageLocation()
	rss.s3Creds, err = testutil.CreateSecret(rss.cli, rss.namespace.GetName(), "test-s3-creds-", corev1.SecretType(secrets.AWSSecretType), storageCredentialsSecretData)
	c.Assert(err, IsNil)
	rss.s3Location, err = testutil.CreateSecret(rss.cli, rss.namespace.GetName(), "test-s3-location-", repositoryserver.Location, storageStorageLocationSecretData)
	c.Assert(err, IsNil)

	// Configure and Create Test User Access Secret for Kopia Repository Server
	userAccessSecretData := testutil.GetRepoServerUserAccessSecretData(defaultKopiaRepositoryHost, testKopiaRepoServerAdminPassword)
	rss.userAccessSecret, err = testutil.CreateSecret(rss.cli, rss.namespace.GetName(), "test-repository-server-user-access-", "", userAccessSecretData)
	c.Assert(err, IsNil)

	// Configure and Create Test TLS Certificate Secret
	kopiaTLSSecretata, err := testutil.GetKopiaTLSSecretData()
	c.Assert(err, IsNil)
	rss.tlsSecret, err = testutil.CreateSecret(rss.cli, rss.namespace.GetName(), "test-repository-server-user-access-", corev1.SecretTypeTLS, kopiaTLSSecretata)
	c.Assert(err, IsNil)

	// Create Test Pod
	rss.pod, err = createRepositoryServerTestPod(rss.ctx, rss.cli, rss.namespace.GetName(), rss.tlsSecret)
	c.Assert(err, IsNil)

	// Wait for Test Pod to get Ready
	c.Assert(kube.WaitForPodReady(rss.ctx, rss.cli, rss.namespace.GetName(), rss.pod.Name), IsNil)

	// Create Test Service
	rss.service, err = createRepositoryServerTestService(rss.ctx, rss.cli, rss.namespace.GetName())
	c.Assert(err, IsNil)

	// Configure and Create Kopia Repository
	err = createTestKopiaRepository(rss.s3Location, rss.cli, rss.namespace.GetName(), rss.pod)
	c.Assert(err, IsNil)

	// Start Kopia Repository Server
	err = startTestKopiaRepositoryServer(rss.cli, rss.namespace.GetName(), rss.pod)
	c.Assert(err, IsNil)

	// Wait for Kopia Repository Server To Get Ready
	err = waitForServerReady(rss.ctx, rss.cli, rss.namespace.GetName(), rss.pod, rss.tlsSecret)
	c.Assert(err, IsNil)

	// Add Test Client User in Kopia Repository
	err = addTestUserInKopiaRepositoryServer(rss.cli, rss.namespace.GetName(), rss.pod, rss.user)
	c.Assert(err, IsNil)

	// Refresh Kopia Repository Server
	err = refreshTestKopiaRepositoryServer(rss.ctx, rss.cli, rss.namespace.GetName(), rss.pod, rss.tlsSecret)
	c.Assert(err, IsNil)

	// Wait for Kopia Repository Server To Get Ready
	err = waitForServerReady(rss.ctx, rss.cli, rss.namespace.GetName(), rss.pod, rss.tlsSecret)
	c.Assert(err, IsNil)

	// Set Kopia Repo Server Template Param
	contentCacheMB, metadataCacheMB := kopiacmd.GetGeneralCacheSizeSettings()
	rss.pod, err = rss.cli.CoreV1().Pods(rss.namespace.GetName()).Get(rss.ctx, rss.pod.GetName(), metav1.GetOptions{})
	c.Assert(err, IsNil)
	rss.repoServer = &param.RepositoryServer{
		Name:      testRepoServerName,
		Namespace: rss.namespace.GetName(),
		ServerInfo: crv1alpha1.ServerInfo{
			PodName:     rss.pod.GetName(),
			ServiceName: rss.service.GetName(),
		},
		Username: rss.user,
		Credentials: param.RepositoryServerCredentials{
			ServerTLS:        *rss.tlsSecret,
			ServerUserAccess: *rss.userAccessSecret,
		},
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
		Address:         fmt.Sprintf("https://%s:%d", rss.pod.Status.HostIP, rss.service.Spec.Ports[0].NodePort),
	}
}

func (rss *RepositoryServerSuite) connectWithTestKopiaRepositoryServer() error {
	return repository.ConnectToAPIServer(
		rss.ctx,
		string(rss.tlsSecret.Data[kopia.TLSCertificateKey]),
		testKopiaRepoServerAdminPassword,
		defaultKopiaRepositoryHost,
		rss.repoServer.Address,
		rss.repoServer.Username,
		rss.repoServer.ContentCacheMB,
		rss.repoServer.MetadataCacheMB,
	)
}

func (rss *RepositoryServerSuite) TestLocationOperationsForRepositoryServerDataMover(c *C) {
	// Setup Test Data
	sourceDir := c.MkDir()
	filePath := filepath.Join(sourceDir, "test.txt")

	cmd := exec.Command("touch", filePath)
	_, err := cmd.Output()
	c.Assert(err, IsNil)

	targetDir := c.MkDir()

	// Connect With Test Kopia RepositoryServer
	err = rss.connectWithTestKopiaRepositoryServer()
	c.Assert(err, IsNil)

	// Test Kopia Repository Server Location Push
	snapInfo, err := kopiaLocationPush(rss.ctx, defaultKopiaRepositoryPath, "kandoOutput", sourceDir, testKopiaRepoServerAdminPassword)
	c.Assert(err, IsNil)

	// Test Kopia Repository Server Location Pull
	err = kopiaLocationPull(rss.ctx, snapInfo.ID, defaultKopiaRepositoryPath, targetDir, testKopiaRepoServerAdminPassword)
	c.Assert(err, IsNil)

	// Test Kopia Repository Location Delete
	err = kopiaLocationDelete(rss.ctx, snapInfo.ID, defaultKopiaRepositoryPath, testKopiaRepoServerAdminPassword)
	c.Assert(err, IsNil)
}

func (rss *RepositoryServerSuite) TearDownSuite(c *C) {
	// Delete Secrets
	err := rss.cli.CoreV1().Secrets(rss.namespace.GetName()).Delete(rss.ctx, rss.tlsSecret.GetName(), metav1.DeleteOptions{})
	c.Assert(err, IsNil)

	err = rss.cli.CoreV1().Secrets(rss.namespace.GetName()).Delete(rss.ctx, rss.userAccessSecret.GetName(), metav1.DeleteOptions{})
	c.Assert(err, IsNil)

	err = rss.cli.CoreV1().Secrets(rss.namespace.GetName()).Delete(rss.ctx, rss.s3Creds.GetName(), metav1.DeleteOptions{})
	c.Assert(err, IsNil)

	err = rss.cli.CoreV1().Secrets(rss.namespace.GetName()).Delete(rss.ctx, rss.s3Location.GetName(), metav1.DeleteOptions{})
	c.Assert(err, IsNil)

	// Delete Service
	err = rss.cli.CoreV1().Services(rss.namespace.GetName()).Delete(rss.ctx, rss.service.GetName(), metav1.DeleteOptions{})
	c.Assert(err, IsNil)

	// Delete Test Pod
	err = rss.cli.CoreV1().Pods(rss.namespace.GetName()).Delete(rss.ctx, rss.pod.GetName(), metav1.DeleteOptions{})
	c.Assert(err, IsNil)

	// Delete Namespace
	err = rss.cli.CoreV1().Namespaces().Delete(rss.ctx, rss.namespace.GetName(), metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}
