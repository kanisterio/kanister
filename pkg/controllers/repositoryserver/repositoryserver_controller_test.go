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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const (
	controllerPodName     = "test-pod"
	defaultServiceAccount = "default"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type RepoServerControllerSuite struct {
	crCli                         crclientv1alpha1.CrV1alpha1Interface
	kubeCli                       kubernetes.Interface
	repoServerControllerNamespace string
	repoServerSecrets             repositoryServerSecrets
	DefaultRepoServerReconciler   *RepositoryServerReconciler
	cancel                        context.CancelFunc
	k8sServerVersion              *version.Info
}

// patchStringValue specifies a patch operation for a string.
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

var _ = Suite(&RepoServerControllerSuite{})

func (s *RepoServerControllerSuite) SetUpSuite(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	c.Assert(err, IsNil)
	s.k8sServerVersion, err = discoveryClient.ServerVersion()
	c.Assert(err, IsNil)

	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)

	crCli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)
	ctx, cancel := context.WithCancel(context.TODO())
	// Make sure the CRDs exist.
	err = resource.CreateCustomResources(ctx, config)
	c.Assert(err, IsNil)

	err = resource.CreateRepoServerCustomResource(ctx, config)
	c.Assert(err, IsNil)

	s.kubeCli = cli
	s.crCli = crCli

	scheme := runtime.NewScheme()
	utilruntime.Must(k8sscheme.AddToScheme(scheme))
	utilruntime.Must(crv1alpha1.AddToScheme(scheme))

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "repositoryservercontrollertest-",
		},
	}
	cns, err := s.kubeCli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	s.repoServerControllerNamespace = cns.Name
	ws := webhook.NewServer(webhook.Options{Port: 9443})
	// Since we are not creating the controller in a pod
	// the repository server controller needs few env variables set explicitly
	err = os.Setenv("POD_NAMESPACE", s.repoServerControllerNamespace)
	c.Assert(err, IsNil)

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:         scheme,
		WebhookServer:  ws,
		Metrics:        server.Options{BindAddress: "0"},
		LeaderElection: false,
	})
	c.Assert(err, IsNil)

	repoReconciler := &RepositoryServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}

	err = repoReconciler.SetupWithManager(mgr)
	c.Assert(err, IsNil)

	// Since the manager is not started inside a pod,
	// the controller needs a pod reference to start successfully
	podSpec := getTestKanisterToolsPod(controllerPodName)
	_, err = cli.CoreV1().Pods(s.repoServerControllerNamespace).Create(ctx, podSpec, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = kube.WaitForPodReady(ctx, s.kubeCli, s.repoServerControllerNamespace, controllerPodName)
	c.Assert(err, IsNil)

	go func(ctx context.Context) {
		// Env setup required to start the controller service
		// We need to set this up since we are not creating controller in a pod
		err := os.Setenv("HOSTNAME", controllerPodName)
		c.Assert(err, IsNil)
		err = os.Setenv("POD_SERVICE_ACCOUNT", defaultServiceAccount)
		c.Assert(err, IsNil)
		// Set KANISTER_TOOLS env to override and use dev image
		err = os.Setenv(consts.KanisterToolsImageEnvName, consts.LatestKanisterToolsImage)
		c.Assert(err, IsNil)
		err = mgr.Start(ctx)
		c.Assert(err, IsNil)
	}(ctx)

	s.DefaultRepoServerReconciler = repoReconciler
	s.cancel = cancel
	s.createRepositoryServerSecrets(c)
}

func (s *RepoServerControllerSuite) createRepositoryServerSecrets(c *C) {
	kopiaTLSSecretData, err := testutil.GetKopiaTLSSecretData()
	c.Assert(err, IsNil)

	s.repoServerSecrets = repositoryServerSecrets{}
	s.repoServerSecrets.serverUserAccess, err = s.CreateRepositoryServerUserAccessSecret(testutil.GetRepoServerUserAccessSecretData("localhost", testutil.KopiaRepositoryServerAccessPassword))
	c.Assert(err, IsNil)

	s.repoServerSecrets.serverAdmin, err = s.CreateRepositoryServerAdminSecret(testutil.GetRepoServerAdminSecretData(testutil.KopiaRepositoryServerAdminUser, testutil.KopiaRepositoryServerAdminPassword))
	c.Assert(err, IsNil)

	s.repoServerSecrets.repositoryPassword, err = s.CreateRepositoryPasswordSecret(testutil.GetRepoPasswordSecretData(testutil.KopiaRepositoryPassword))
	c.Assert(err, IsNil)

	s.repoServerSecrets.serverTLS, err = s.CreateKopiaTLSSecret(kopiaTLSSecretData)
	c.Assert(err, IsNil)

	s.repoServerSecrets.storage, err = s.CreateStorageLocationSecret(testutil.GetDefaultS3CompliantStorageLocation())
	c.Assert(err, IsNil)

	s.repoServerSecrets.storageCredentials, err = s.CreateAWSStorageCredentialsSecret(testutil.GetDefaultS3StorageCreds(c))
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) CreateRepositoryServerAdminSecret(data map[string][]byte) (se *corev1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-server-admin-", repositoryserver.AdminCredentialsSecret, data)
}

func (s *RepoServerControllerSuite) CreateRepositoryServerUserAccessSecret(data map[string][]byte) (se *corev1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-server-user-access-", "", data)
}

func (s *RepoServerControllerSuite) CreateRepositoryPasswordSecret(data map[string][]byte) (se *corev1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-password-", repositoryserver.RepositoryPasswordSecret, data)
}

func (s *RepoServerControllerSuite) CreateKopiaTLSSecret(data map[string][]byte) (se *corev1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-kopia-tls-", corev1.SecretTypeTLS, data)
}

func (s *RepoServerControllerSuite) CreateStorageLocationSecret(data map[string][]byte) (se *corev1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-server-storage-", repositoryserver.Location, data)
}

func (s *RepoServerControllerSuite) CreateAWSStorageCredentialsSecret(data map[string][]byte) (se *corev1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-server-storage-creds-", corev1.SecretType(secrets.AWSSecretType), data)
}

func (s *RepoServerControllerSuite) CreateAzureStorageCredentialsSecret(data map[string][]byte) (se *corev1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-server-storage-creds-", corev1.SecretType(secrets.AzureSecretType), data)
}

func (s *RepoServerControllerSuite) CreateGCPStorageCredentialsSecret(data map[string][]byte) (se *corev1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-server-storage-creds-", corev1.SecretType(secrets.GCPSecretType), data)
}

func (s *RepoServerControllerSuite) TestRepositoryServerImmutability(c *C) {
	minorVersion, err := strconv.Atoi(s.k8sServerVersion.Minor)
	c.Assert(err, IsNil)

	if s.k8sServerVersion.Major == "1" && minorVersion < 25 {
		c.Skip("skipping the test since CRD validation rules feature is enabled only after k8s version 1.25")
	}

	ctx := context.Background()

	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repoServerCR)

	// Create a repository server CR
	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Update the repository server CR's Immutable field.
	patch := []patchStringValue{{
		Op:    "replace",
		Path:  "/spec/repository/rootPath",
		Value: "/updated-test-path/",
	}}
	patchBytes, _ := json.Marshal(patch)
	_, err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Patch(ctx, repoServerCRCreated.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	c.Assert(err, NotNil)

	// Check Error Message
	expectedErrorMessage := fmt.Sprintf("RepositoryServer.cr.kanister.io \"%s\" is invalid: spec.repository.rootPath: Invalid value: \"string\": Value is immutable", repoServerCRCreated.GetName())
	c.Assert(err.Error(), Equals, expectedErrorMessage)

	// Delete the repository server CR.
	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(ctx, repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

// TestRepositoryServerStatusIsServerReady creates a CR with correct configurations and
// tests that the CR gets into created/ready state
func (s *RepoServerControllerSuite) TestRepositoryServerStatusIsServerReady(c *C) {
	ctx := context.Background()
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repoServerCR)

	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForRepoServerInfoUpdateInCR(repoServerCRCreated.Name)
	c.Assert(err, IsNil)

	// Get repository server CR with the updated server information
	repoServerCRCreated, err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Get(ctx, repoServerCRCreated.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)

	err = kube.WaitForPodReady(ctx, s.kubeCli, s.repoServerControllerNamespace, repoServerCRCreated.Status.ServerInfo.PodName)
	c.Assert(err, IsNil)

	err = testutil.CreateTestKopiaRepository(ctx, s.kubeCli, repoServerCRCreated, testutil.GetDefaultS3CompliantStorageLocation())
	c.Assert(err, IsNil)

	_, err = s.waitOnRepositoryServerState(c, repoServerCRCreated.Name)
	c.Assert(err, IsNil)

	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(context.Background(), repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

// TestRepositoryServerCRStateWithoutSecrets checks if server creation is failed
// when no storage secrets are set
func (s *RepoServerControllerSuite) TestRepositoryServerCRStateWithoutSecrets(c *C) {
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	ctx := context.Background()
	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	state, err := s.waitOnRepositoryServerState(c, repoServerCRCreated.Name)
	c.Assert(err, NotNil)
	c.Assert(state, Equals, crv1alpha1.Failed)

	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(context.Background(), repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

// TestCreationOfOwnedResources checks if pod and service for repository server
// is created successfully
func (s *RepoServerControllerSuite) TestCreationOfOwnedResources(c *C) {
	ctx := context.Background()

	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repoServerCR)

	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForRepoServerInfoUpdateInCR(repoServerCRCreated.Name)
	c.Assert(err, IsNil)

	// Get repository server CR with the updated server information
	repoServerCRCreated, err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Get(ctx, repoServerCRCreated.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)

	pod, err := s.kubeCli.CoreV1().Pods(s.repoServerControllerNamespace).Get(ctx, repoServerCRCreated.Status.ServerInfo.PodName, metav1.GetOptions{})
	c.Assert(err, IsNil)
	c.Assert(len(pod.OwnerReferences), Equals, 1)
	c.Assert(pod.OwnerReferences[0].UID, Equals, repoServerCRCreated.UID)

	service, err := s.kubeCli.CoreV1().Services(s.repoServerControllerNamespace).Get(ctx, repoServerCRCreated.Status.ServerInfo.ServiceName, metav1.GetOptions{})
	c.Assert(err, IsNil)
	c.Assert(len(service.OwnerReferences), Equals, 1)
	c.Assert(service.OwnerReferences[0].UID, Equals, repoServerCRCreated.UID)

	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(context.Background(), repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) TestInvalidRepositoryPassword(c *C) {
	ctx := context.Background()
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repoServerCR)

	InvalidRepositoryPassword, err := s.CreateRepositoryPasswordSecret(testutil.GetRepoPasswordSecretData("invalidPassword"))
	c.Assert(err, IsNil)

	repoServerCR.Spec.Repository.PasswordSecretRef.Name = InvalidRepositoryPassword.Name
	repoServerCR.Spec.Repository.PasswordSecretRef.Namespace = InvalidRepositoryPassword.Namespace

	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	state, err := s.waitOnRepositoryServerState(c, repoServerCRCreated.Name)
	c.Assert(err, NotNil)
	c.Assert(state, Equals, crv1alpha1.Failed)

	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(context.Background(), repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) TestInvalidStorageLocation(c *C) {
	ctx := context.Background()
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repoServerCR)

	storageLocationData := testutil.GetDefaultS3CompliantStorageLocation()
	storageLocationData[repositoryserver.BucketKey] = []byte("invalidbucket")

	InvalidStorageLocationSecret, err := s.CreateStorageLocationSecret(storageLocationData)
	c.Assert(err, IsNil)

	repoServerCR.Spec.Storage.SecretRef.Name = InvalidStorageLocationSecret.Name
	repoServerCR.Spec.Storage.SecretRef.Namespace = InvalidStorageLocationSecret.Namespace

	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	state, err := s.waitOnRepositoryServerState(c, repoServerCRCreated.Name)
	c.Assert(err, NotNil)
	c.Assert(state, Equals, crv1alpha1.Failed)

	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(context.Background(), repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) TestInvalidStorageLocationCredentials(c *C) {
	ctx := context.Background()
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repoServerCR)

	storageLocationCredsData := testutil.GetDefaultS3StorageCreds(c)
	storageLocationCredsData[secrets.AWSAccessKeyID] = []byte("testaccesskey")

	InvalidStorageLocationCrdesSecret, err := s.CreateAWSStorageCredentialsSecret(storageLocationCredsData)
	c.Assert(err, IsNil)

	repoServerCR.Spec.Storage.CredentialSecretRef.Name = InvalidStorageLocationCrdesSecret.Name
	repoServerCR.Spec.Storage.CredentialSecretRef.Namespace = InvalidStorageLocationCrdesSecret.Namespace

	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	state, err := s.waitOnRepositoryServerState(c, repoServerCRCreated.Name)
	c.Assert(err, NotNil)
	c.Assert(state, Equals, crv1alpha1.Failed)

	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(context.Background(), repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) TestFilestoreLocationVolumeMountOnRepoServerPod(c *C) {
	var err error
	ctx := context.Background()
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repoServerCR)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-pvc-",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): k8sresource.MustParse("1Gi"),
				},
			},
		},
	}
	pvc, err = s.kubeCli.CoreV1().PersistentVolumeClaims(s.repoServerControllerNamespace).Create(ctx, pvc, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	storageSecret, err := s.CreateStorageLocationSecret(testutil.GetFileStoreLocationSecretData(pvc.Name))
	c.Assert(err, IsNil)

	repoServerCR.Spec.Storage.SecretRef.Name = storageSecret.Name

	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForRepoServerInfoUpdateInCR(repoServerCRCreated.Name)
	c.Assert(err, IsNil)

	// Get repository server CR with the updated server information
	repoServerCRCreated, err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Get(ctx, repoServerCRCreated.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)

	pod, err := s.kubeCli.CoreV1().Pods(s.repoServerControllerNamespace).Get(ctx, repoServerCRCreated.Status.ServerInfo.PodName, metav1.GetOptions{})
	c.Assert(err, IsNil)

	c.Assert(len(pod.Spec.Volumes), Equals, 3)

	var volumeattached bool
	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil && vol.PersistentVolumeClaim.ClaimName == pvc.Name {
			volumeattached = true
		}
	}
	c.Assert(volumeattached, Equals, true)

	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(context.Background(), repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) waitForRepoServerInfoUpdateInCR(repoServerName string) error {
	ctxTimeout := 25 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()
	var serverInfoUpdated bool
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		repoServerCR, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Get(ctx, repoServerName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if repoServerCR.Status.ServerInfo.PodName == "" || repoServerCR.Status.ServerInfo.ServiceName == "" {
			return false, nil
		}
		serverInfoUpdated = true
		return true, nil
	})

	if !serverInfoUpdated && err == nil {
		err = errors.New("pod name or service name is not set on repository server CR")
	}

	if err != nil {
		return errors.Wrapf(err, "failed waiting for RepoServer Info updates in the CR")
	}
	return err
}

func (s *RepoServerControllerSuite) waitOnRepositoryServerState(c *C, reposerverName string) (crv1alpha1.RepositoryServerProgress, error) {
	ctxTimeout := 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()
	var repoServerState crv1alpha1.RepositoryServerProgress
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		repoServerCR, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Get(ctx, reposerverName, metav1.GetOptions{})
		if err != nil {
			repoServerState = ""
			return false, err
		}
		repoServerState = repoServerCR.Status.Progress
		if repoServerCR.Status.Progress == "" || repoServerCR.Status.Progress == crv1alpha1.Pending {
			return false, nil
		}
		if repoServerCR.Status.Progress == crv1alpha1.Failed {
			return false, errors.New(fmt.Sprintf(" There is failure in staring the repository server, server is in %s state, please check logs", repoServerCR.Status.Progress))
		}
		if repoServerCR.Status.Progress == crv1alpha1.Ready {
			return true, nil
		}
		return false, errors.New(fmt.Sprintf("Unexpected Repository server state: %s", repoServerCR.Status.Progress))
	})
	return repoServerState, err
}

func setRepositoryServerSecretsInCR(secrets *repositoryServerSecrets, repoServerCR *crv1alpha1.RepositoryServer) {
	if secrets != nil {
		if secrets.serverAdmin != nil {
			repoServerCR.Spec.Server.AdminSecretRef.Name = secrets.serverAdmin.Name
		}
		if secrets.repositoryPassword != nil {
			repoServerCR.Spec.Repository.PasswordSecretRef.Name = secrets.repositoryPassword.Name
		}

		if secrets.serverUserAccess != nil {
			repoServerCR.Spec.Server.UserAccess.UserAccessSecretRef.Name = secrets.serverUserAccess.Name
		}
		if secrets.serverTLS != nil {
			repoServerCR.Spec.Server.TLSSecretRef.Name = secrets.serverTLS.Name
		}
		if secrets.storage != nil {
			repoServerCR.Spec.Storage.SecretRef.Name = secrets.storage.Name
		}
		if secrets.storageCredentials != nil {
			repoServerCR.Spec.Storage.CredentialSecretRef.Name = secrets.storageCredentials.Name
		}
	}
}

func getTestKanisterToolsPod(podName string) (pod *corev1.Pod) {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "kanister-tools",
					Image: consts.LatestKanisterToolsImage,
				},
			},
		},
	}
}

func (s *RepoServerControllerSuite) TearDownSuite(c *C) {
	err := os.Unsetenv(consts.KanisterToolsImageEnvName)
	c.Assert(err, IsNil)
	if s.repoServerControllerNamespace != "" {
		err := s.kubeCli.CoreV1().Namespaces().Delete(context.TODO(), s.repoServerControllerNamespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
	if s.cancel != nil {
		s.cancel()
	}
}
