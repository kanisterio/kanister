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
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
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
}

var _ = Suite(&RepositoryServerSuite{})

func (rss *RepositoryServerSuite) TestRepositoryServerImplementsDataMover(c *C) {
	rs := repositoryServer{}
	var dm interface{} = rs
	_, ok := dm.(DataMover)
	c.Assert(ok, Equals, false)

	dm = &rs
	_, ok = dm.(DataMover)
	c.Assert(ok, Equals, true)
}

func (rss *RepositoryServerSuite) SetUpSuite(c *C) {
	// Set Context as Background
	rss.ctx = context.Background()

	// Get Kubernetes Client
	cli, err := newTestClient()
	c.Assert(err, IsNil)
	rss.cli = cli

	// Create Test Namespace
	rss.namespace, err = createRepositoryServerTestNamespace(rss.ctx, rss.cli)
	c.Assert(err, IsNil)

	// Configure and Create Test S3 Credential and Location Secrets
	rss.s3Creds, rss.s3Location, err = testS3CredsLocationSecret(rss.ctx, rss.cli, rss.namespace.GetName())
	c.Assert(err, IsNil)

	// Configure and Create Test User Access Secret for Kopia Repository Server
	rss.userAccessSecret, err = testKopiaRepositoryServerUserAccess(rss.ctx, rss.cli, rss.namespace.GetName())
	c.Assert(err, IsNil)

	// Configure and Create Test TLS Certificate Secret
	rss.tlsSecret, err = testKopiaTLSCertificate(rss.ctx, rss.cli, rss.namespace.GetName())
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
	err = addTestUserInKopiaRepositoryServer(rss.cli, rss.namespace.GetName(), rss.pod)
	c.Assert(err, IsNil)

	// Refresh Kopia Repository Server
	err = refreshTestKopiaRepositoryServer(rss.ctx, rss.cli, rss.namespace.GetName(), rss.pod, rss.tlsSecret)
	c.Assert(err, IsNil)

	// Wait for Kopia Repository Server To Get Ready
	err = waitForServerReady(rss.ctx, rss.cli, rss.namespace.GetName(), rss.pod, rss.tlsSecret)
	c.Assert(err, IsNil)

	// Set Kopia Repo Server Template Param
	contentCacheMB, metadataCacheMB := kopiacmd.GetGeneralCacheSizeSettings()
	rss.repoServer = &param.RepositoryServer{
		Name:      testRepoServerName,
		Namespace: rss.namespace.GetName(),
		ServerInfo: crv1alpha1.ServerInfo{
			PodName:     rss.pod.GetName(),
			ServiceName: rss.service.GetName(),
		},
		Username: testKopiaRepoServerUsername,
		Credentials: param.RepositoryServerCredentials{
			ServerTLS:        *rss.tlsSecret,
			ServerUserAccess: *rss.userAccessSecret,
		},
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
		Address:         fmt.Sprintf("https://%s.%s.%s:%d", rss.service.GetName(), rss.namespace.GetName(), clusterLocalDomain, rss.service.Spec.Ports[0].Port),
	}
}
