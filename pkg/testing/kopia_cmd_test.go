//go:build kopia
// +build kopia

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

package testing

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const (
	testPodName = "kopia-cmd-"
)

type KopiaCmdSuite struct {
	cli       kubernetes.Interface
	namespace string
	locType   storage.LocType
}

var _ = check.Suite(&KopiaCmdSuite{locType: storage.LocTypeS3})
var _ = check.Suite(&KopiaCmdSuite{locType: storage.LocTypeFilestore})

func (s *KopiaCmdSuite) SetUpSuite(c *check.C) {
	s.skipIfEnvNotSet(c)
	config, err := kube.LoadConfig()
	c.Assert(err, check.IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, check.IsNil)
	s.cli = cli

	ctx := context.Background()
	ns := testutil.NewTestNamespace()
	ns.GenerateName = "kanister-datatest-"

	cns, err := s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = cns.GetName()
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-sa-",
			Namespace:    s.namespace,
		},
	}

	sa, err = cli.CoreV1().ServiceAccounts(s.namespace).Create(ctx, sa, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	os.Setenv("POD_NAMESPACE", s.namespace)
	os.Setenv("POD_SERVICE_ACCOUNT", sa.Name)
}

func (s *KopiaCmdSuite) TearDownSuite(c *check.C) {
	os.Unsetenv("POD_NAMESPACE")
	os.Unsetenv("POD_SERVICE_ACCOUNT")
	ctx := context.Background()
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(ctx, s.namespace, metav1.DeleteOptions{})
	}
}

func (s *KopiaCmdSuite) TestRepositoryCreateConnect(c *check.C) {
	locSecret, credSecret := s.createLocationAndCredSecrets(c)
	pod := s.startKanisterToolsPod(c, locSecret, credSecret)
	repoPath := fmt.Sprintf("test-path/test-%v", time.Now().Unix())
	repoPassword := "test-pass123"
	hostname := "test-hostname"
	username := "test-username"
	err := repository.CreateKopiaRepository(s.cli, s.namespace, pod.Name, pod.Spec.Containers[0].Name, command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			ConfigFilePath: "/tmp/config",
			LogDirectory:   "/tmp/logs",
		},
		RepoPassword:    repoPassword,
		Hostname:        hostname,
		Username:        username,
		RepoPathPrefix:  repoPath,
		ContentCacheMB:  0,
		MetadataCacheMB: 0,
		CacheDirectory:  "/tmp/cache",
		Location:        locSecret.Data,
	})
	c.Assert(err, check.IsNil)

	err = repository.ConnectToKopiaRepository(s.cli, s.namespace, pod.Name, pod.Spec.Containers[0].Name, command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			ConfigFilePath: "/tmp/config-new",
			LogDirectory:   "/tmp/logs",
		},
		RepoPassword:    repoPassword,
		Hostname:        hostname,
		Username:        username,
		RepoPathPrefix:  repoPath,
		ContentCacheMB:  0,
		MetadataCacheMB: 0,
		CacheDirectory:  "/tmp/cache",
		Location:        locSecret.Data,
	})
	c.Assert(err, check.IsNil)
}

func (s *KopiaCmdSuite) startKanisterToolsPod(c *check.C, locSecret, credSecret *v1.Secret) *v1.Pod {
	var (
		envVars []v1.EnvVar
		err     error
	)
	if credSecret != nil {
		envVars, err = storage.GenerateEnvSpecFromCredentialSecret(credSecret, time.Duration(30*time.Minute))
		c.Assert(err, check.IsNil)
	}
	options := &kube.PodOptions{
		Namespace:            s.namespace,
		GenerateName:         testPodName,
		Image:                consts.LatestKanisterToolsImage,
		Command:              []string{"bash", "-c", "tail -f /dev/null"},
		EnvironmentVariables: envVars,
	}
	pod, err := kube.CreatePod(context.Background(), s.cli, options)
	c.Assert(err, check.IsNil)
	err = kube.WaitForPodReady(context.Background(), s.cli, pod.Namespace, pod.Name)
	c.Assert(err, check.IsNil)
	return pod
}

func (s *KopiaCmdSuite) createLocationAndCredSecrets(c *check.C) (locSecret, credSecret *v1.Secret) {
	switch s.locType {
	case storage.LocTypeFilestore:
		locSecret = s.createFileStoreSecrets(c)
	case storage.LocTypeS3:
		locSecret, credSecret = s.createS3Secrets(c)
	default:
		c.Log("Unsupported test location type")
		c.Fail()
	}
	return
}

func (s *KopiaCmdSuite) createFileStoreSecrets(c *check.C) *v1.Secret {
	ls := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "location-secret-",
		},
		Data: storage.GetMapForLocationValues(s.locType, "test-prefix", "", "", "", ""),
	}
	locSecret, err := s.cli.CoreV1().Secrets(s.namespace).Create(context.Background(), ls, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	return locSecret
}

func (s *KopiaCmdSuite) createS3Secrets(c *check.C) (locSecret, credSecret *v1.Secret) {
	locSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "location-secret-",
		},
		Data: storage.GetMapForLocationValues(s.locType, "test-prefix", "us-west-2", "tests.kanister.io", "http://minio.minio.svc.cluster.local:9000", "true"),
	}
	var err error
	locSecret, err = s.cli.CoreV1().Secrets(s.namespace).Create(context.Background(), locSecret, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	accessKeyID := os.Getenv(aws.AccessKeyID)
	if accessKeyID == "" {
		c.Log(aws.AccessKeyID, " not set")
		c.Fail()
	}
	secretAccessKey := os.Getenv(aws.SecretAccessKey)
	if secretAccessKey == "" {
		c.Log(aws.SecretAccessKey, " not set")
		c.Fail()
	}
	credSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "creds-secret-",
		},
		Type: v1.SecretType(secrets.AWSSecretType),
		Data: map[string][]byte{
			secrets.AWSAccessKeyID:     []byte(accessKeyID),
			secrets.AWSSecretAccessKey: []byte(secretAccessKey),
		},
	}
	credSecret, err = s.cli.CoreV1().Secrets(s.namespace).Create(context.Background(), credSecret, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	return locSecret, credSecret
}

func (s *KopiaCmdSuite) skipIfEnvNotSet(c *check.C) {
	switch s.locType {
	case storage.LocTypeS3:
		getEnvOrSkip(aws.AccessKeyID, c)
		getEnvOrSkip(aws.SecretAccessKey, c)
	}
}

func getEnvOrSkip(env string, c *check.C) {
	if os.Getenv(env) == "" {
		c.Skip(fmt.Sprint("Env not set: ", env))
	}
}
