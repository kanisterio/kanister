//go:build integration
// +build integration

// Copyright 2019 The Kanister Authors.
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
	context "context"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"os"
	test "testing"
	"time"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/app"
	crclient "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/controller"
	"github.com/kanisterio/kanister/pkg/field"
	_ "github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/kanctl"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

// Hook up gocheck into the "go test" runner for integration builds
func Test(t *test.T) {
	integrationSetup(t)
	TestingT(t)
	integrationCleanup(t)
}

// Global variables shared across Suite instances
type kanisterKontroller struct {
	namespace          string
	context            context.Context
	cancel             context.CancelFunc
	kubeCli            *kubernetes.Clientset
	serviceAccount     *v1.ServiceAccount
	clusterRole        *rbacv1.ClusterRole
	clusterRoleBinding *rbacv1.ClusterRoleBinding
}

var kontroller kanisterKontroller

func integrationSetup(t *test.T) {
	ns := "integration-test-controller-" + rand.String(5)
	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := kube.LoadConfig()
	if err != nil {
		t.Fatalf("Integration test setup failure: Error loading kube.Config; err=%v", err)
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Integration test setup failure: Error creating kubeCli; err=%v", err)
	}
	if err = createNamespace(cli, ns); err != nil {
		t.Fatalf("Integration test setup failure: Error creating namespace; err=%v", err)
	}
	sa, err := cli.CoreV1().ServiceAccounts(ns).Create(ctx, getServiceAccount(ns, controllerSA), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Integration test setup failure: Error creating service account; err=%v", err)
	}
	clusterRole, err := cli.RbacV1().ClusterRoles().Create(ctx, getClusterRole(ns), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Integration test setup failure: Error creating clusterrole; err=%v", err)
	}
	crb, err := cli.RbacV1().ClusterRoleBindings().Create(ctx, getClusterRoleBinding(sa, clusterRole), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Integration test setup failure: Error creating clusterRoleBinding; err=%v", err)
	}
	// Set Controller namespace and service account
	os.Setenv(kube.PodNSEnvVar, ns)
	os.Setenv(kube.PodSAEnvVar, controllerSA)

	if err = resource.CreateCustomResources(ctx, cfg); err != nil {
		t.Fatalf("Integration test setup failure: Error createing custom resources; err=%v", err)
	}
	ctlr := controller.New(cfg)
	if err = ctlr.StartWatch(ctx, ns); err != nil {
		t.Fatalf("Integration test setup failure: Error starting controller; err=%v", err)
	}

	kontroller.namespace = ns
	kontroller.context = ctx
	kontroller.cancel = cancel
	kontroller.kubeCli = cli
	kontroller.serviceAccount = sa
	kontroller.clusterRole = clusterRole
	kontroller.clusterRoleBinding = crb
}

func integrationCleanup(t *test.T) {
	ctx, cancel := context.WithTimeout(context.Background(), contextWaitTimeout)
	defer cancel()

	if kontroller.cancel != nil {
		kontroller.cancel()
	}
	if kontroller.namespace != "" {
		kontroller.kubeCli.CoreV1().Namespaces().Delete(ctx, kontroller.namespace, metav1.DeleteOptions{})
	}
	if kontroller.clusterRoleBinding != nil && kontroller.clusterRoleBinding.Name != "" {
		kontroller.kubeCli.RbacV1().ClusterRoleBindings().Delete(ctx, kontroller.clusterRoleBinding.Name, metav1.DeleteOptions{})
	}
	if kontroller.clusterRole != nil && kontroller.clusterRole.Name != "" {
		kontroller.kubeCli.RbacV1().ClusterRoles().Delete(ctx, kontroller.clusterRole.Name, metav1.DeleteOptions{})
	}
}

const (
	// appWaitTimeout decides the time we are going to wait for app to be ready
	appWaitTimeout     = 3 * time.Minute
	controllerSA       = "kanister-sa"
	contextWaitTimeout = 10 * time.Minute
)

type secretProfile struct {
	secret  *v1.Secret
	profile *crv1alpha1.Profile
}

type secretRepositoryServer struct {
	s3Location         *v1.Secret
	s3Creds            *v1.Secret
	repositoryPassword *v1.Secret
	serverAdminUser    *v1.Secret
	tls                *v1.Secret
	userAccess         *v1.Secret
	repositoryServer   *crv1alpha1.RepositoryServer
}

type IntegrationSuite struct {
	name             string
	cli              kubernetes.Interface
	crCli            crclient.CrV1alpha1Interface
	app              app.App
	bp               app.Blueprinter
	profile          *secretProfile
	repositoryServer *secretRepositoryServer
	namespace        string
	skip             bool
	cancel           context.CancelFunc
}

func newSecretProfile() *secretProfile {
	_, location := testutil.GetObjectstoreLocation()
	secret, profile, err := testutil.NewSecretProfileFromLocation(location)
	if err != nil {
		return nil
	}
	return &secretProfile{
		secret:  secret,
		profile: profile,
	}
}

func newSecretRepositoryServer() *secretRepositoryServer {
	s3Creds, s3Location := testutil.S3CredsLocationSecret()
	tls, err := testutil.KopiaTLSCertificate()
	if err != nil {
		return nil
	}
	return &secretRepositoryServer{
		s3Creds:            s3Creds,
		s3Location:         s3Location,
		repositoryPassword: testutil.KopiaRepositoryPassword(),
		serverAdminUser:    testutil.KopiaRepositoryServerAdminUser(),
		userAccess:         testutil.KopiaRepositoryServerUserAccess(),
		tls:                tls,
		repositoryServer:   testutil.NewKopiaRepositoryServer(),
	}
}

func (s *IntegrationSuite) SetUpSuite(c *C) {
	ctx := context.Background()
	_, s.cancel = context.WithCancel(ctx)

	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	s.cli, err = kubernetes.NewForConfig(cfg)
	c.Assert(err, IsNil)
	s.crCli, err = crclient.NewForConfig(cfg)
	c.Assert(err, IsNil)
}

// TestRun executes e2e workflow on the app
// 1. Install DB app
// 2. Add data
// 3. Create Kanister Profile and Blueprint
// 4. Take Backup
// 5. Delete DB data
// 6. Restore data from backup
// 7. Uninstall DB app
func (s *IntegrationSuite) TestRun(c *C) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Execute e2e workflow
	log.Info().Print("Running e2e integration test.", field.M{"app": s.name, "testName": c.TestName()})

	// Check config
	err := s.app.Init(ctx)
	if err != nil {
		log.Info().Print("Skipping integration test.", field.M{"app": s.name, "reason": err.Error()})
		s.skip = true
		c.Skip(err.Error())
	}

	// Create namespace
	err = createNamespace(s.cli, s.namespace)
	c.Assert(err, IsNil)

	// Install db
	err = s.app.Install(ctx, s.namespace)
	c.Assert(err, IsNil)

	// Check if ready
	ok, err := s.app.IsReady(ctx)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// Create blueprint
	bp := s.bp.Blueprint()
	c.Assert(bp, NotNil)
	_, err = s.crCli.Blueprints(kontroller.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	var configMaps, secrets map[string]crv1alpha1.ObjectReference
	testEntries := 3
	// Add test entries to DB
	if a, ok := s.app.(app.DatabaseApp); ok {
		// wait for application to be actually ready
		err = pingAppAndWait(ctx, a)
		c.Assert(err, IsNil)

		err = a.Reset(ctx)
		c.Assert(err, IsNil)

		err = a.Initialize(ctx)
		c.Assert(err, IsNil)

		// Add few entries
		for i := 0; i < testEntries; i++ {
			c.Assert(a.Insert(ctx), IsNil)
		}

		count, err := a.Count(ctx)
		c.Assert(err, IsNil)
		c.Assert(count, Equals, testEntries)
	}

	// Get Secret and ConfigMap object references
	if a, ok := s.app.(app.ConfigApp); ok {
		configMaps = a.ConfigMaps()
		secrets = a.Secrets()
	}

	// Validate Blueprint
	validateBlueprint(c, *bp, configMaps, secrets)

	var as *crv1alpha1.ActionSet
	log.Print("--------- ENV [KOPIA_INTEGRATION_TEST]-----------")
	log.Print(os.Getenv("KOPIA_INTEGRATION_TEST"))
	if os.Getenv("KOPIA_INTEGRATION_TEST") != "" {
		if s.repositoryServer == nil {
			log.Info().Print("Skipping integration test. Could not create repository server. Please check if required credentials are set.", field.M{"app": s.name})
			s.skip = true
			c.Skip("Could not create a RepositoryServer")
		}
		log.Info().Print("----- Creating Repository Server ----")
		repositoryServerName := s.createRepositoryServer(c, ctx)
		as = newActionSetWithRepoServer(bp.GetName(), repositoryServerName, "kanister", s.app.Object(), configMaps, secrets)
		log.Print("----- ActionSet -----", field.M{
			"Actionset": as,
		})
	} else {
		if s.profile == nil {
			log.Info().Print("Skipping integration test. Could not create profile. Please check if required credentials are set.", field.M{"app": s.name})
			s.skip = true
			c.Skip("Could not create a Profile")
		}
		// Create profile
		profileName := s.createProfile(c, ctx)
		// Create ActionSet specs
		as = newActionSet(bp.GetName(), profileName, kontroller.namespace, s.app.Object(), configMaps, secrets)
	}

	// Take backup
	backup := s.createActionset(ctx, c, as, "backup", nil)
	c.Assert(len(backup), Not(Equals), 0)

	// Save timestamp for PITR
	var restoreOptions map[string]string
	if b, ok := s.bp.(app.PITRBlueprinter); ok {
		pitr := b.FormatPITR(time.Now())
		log.Info().Print("Saving timestamp for PITR", field.M{"pitr": pitr})
		restoreOptions = map[string]string{
			"pitr": pitr,
		}
		// Add few more entries with timestamp > pitr
		time.Sleep(time.Second)
		if a, ok := s.app.(app.DatabaseApp); ok {
			c.Assert(a.Insert(ctx), IsNil)
			c.Assert(a.Insert(ctx), IsNil)

			count, err := a.Count(ctx)
			c.Assert(err, IsNil)
			c.Assert(count, Equals, testEntries+2)
		}
	}

	// Reset DB
	if a, ok := s.app.(app.DatabaseApp); ok {
		err = a.Reset(ctx)
		c.Assert(err, IsNil)
	}

	// Restore backup
	pas, err := s.crCli.ActionSets(kontroller.namespace).Get(ctx, backup, metav1.GetOptions{})
	c.Assert(err, IsNil)
	s.createActionset(ctx, c, pas, "restore", restoreOptions)

	// Verify data
	if a, ok := s.app.(app.DatabaseApp); ok {
		// wait for application to be actually ready
		err = pingAppAndWait(ctx, a)
		c.Assert(err, IsNil)

		count, err := a.Count(ctx)
		c.Assert(err, IsNil)
		c.Assert(count, Equals, testEntries)
	}

	// Delete snapshots
	s.createActionset(ctx, c, pas, "delete", nil)
}

func newActionSet(bpName, profile, profileNs string, object crv1alpha1.ObjectReference, configMaps, secrets map[string]crv1alpha1.ObjectReference) *crv1alpha1.ActionSet {
	return &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Name:      "backup",
					Object:    object,
					Blueprint: bpName,
					Profile: &crv1alpha1.ObjectReference{
						Name:      profile,
						Namespace: profileNs,
					},
					ConfigMaps: configMaps,
					Secrets:    secrets,
				},
			},
		},
	}
}

func newActionSetWithRepoServer(bpName, repositoryServer, repositoryServerNs string, object crv1alpha1.ObjectReference, configMaps, secrets map[string]crv1alpha1.ObjectReference) *crv1alpha1.ActionSet {
	return &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Name:      "backup",
					Object:    object,
					Blueprint: bpName,
					RepositoryServer: &crv1alpha1.ObjectReference{
						Name:      repositoryServer,
						Namespace: repositoryServerNs,
					},
					ConfigMaps: configMaps,
					Secrets:    secrets,
				},
			},
		},
	}
}

func (s *IntegrationSuite) createProfile(c *C, ctx context.Context) string {
	secret, err := s.cli.CoreV1().Secrets(kontroller.namespace).Create(ctx, s.profile.secret, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// set secret ref in profile
	s.profile.profile.Credential.KeyPair.Secret = crv1alpha1.ObjectReference{
		Name:      secret.GetName(),
		Namespace: secret.GetNamespace(),
	}
	profile, err := s.crCli.Profiles(kontroller.namespace).Create(ctx, s.profile.profile, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	return profile.GetName()
}

func (s *IntegrationSuite) createRepositoryServer(c *C, ctx context.Context) string {
	// Create Secrets required for setting up RepositoryServer
	log.Info().Print("----- Create Secrets required for setting up RepositoryServer -----")
	s3Location, err := s.cli.CoreV1().Secrets(testutil.DefaultKanisterNamespace).Create(ctx, s.repositoryServer.s3Location, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s3Creds, err := s.cli.CoreV1().Secrets(testutil.DefaultKanisterNamespace).Create(ctx, s.repositoryServer.s3Creds, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	repositoryPassword, err := s.cli.CoreV1().Secrets(testutil.DefaultKanisterNamespace).Create(ctx, s.repositoryServer.repositoryPassword, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	serverAdminUser, err := s.cli.CoreV1().Secrets(testutil.DefaultKanisterNamespace).Create(ctx, s.repositoryServer.serverAdminUser, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	tls, err := s.cli.CoreV1().Secrets(testutil.DefaultKanisterNamespace).Create(ctx, s.repositoryServer.tls, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	userAccess, err := s.cli.CoreV1().Secrets(testutil.DefaultKanisterNamespace).Create(ctx, s.repositoryServer.userAccess, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Setting Up the SecretRefs in RepositoryServer
	s.repositoryServer.repositoryServer.Spec.Storage.SecretRef = v1.SecretReference{
		Name:      s3Location.GetName(),
		Namespace: s3Location.GetNamespace(),
	}
	s.repositoryServer.repositoryServer.Spec.Storage.CredentialSecretRef = v1.SecretReference{
		Name:      s3Creds.GetName(),
		Namespace: s3Creds.GetNamespace(),
	}
	s.repositoryServer.repositoryServer.Spec.Repository.PasswordSecretRef = v1.SecretReference{
		Name:      repositoryPassword.GetName(),
		Namespace: repositoryPassword.GetNamespace(),
	}
	s.repositoryServer.repositoryServer.Spec.Server.UserAccess.UserAccessSecretRef = v1.SecretReference{
		Name:      userAccess.GetName(),
		Namespace: userAccess.GetNamespace(),
	}
	s.repositoryServer.repositoryServer.Spec.Server.AdminSecretRef = v1.SecretReference{
		Name:      serverAdminUser.GetName(),
		Namespace: serverAdminUser.GetNamespace(),
	}
	s.repositoryServer.repositoryServer.Spec.Server.TLSSecretRef = v1.SecretReference{
		Name:      tls.GetName(),
		Namespace: tls.GetNamespace(),
	}

	// Create RepositoryServer CR
	log.Info().Print("----- Create Repository Server CR -----")
	repositoryServer, err := s.crCli.RepositoryServers(testutil.DefaultKanisterNamespace).Create(ctx, s.repositoryServer.repositoryServer, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	time.Sleep(30 * time.Second)
	repositoryServer, err = s.crCli.RepositoryServers(testutil.DefaultKanisterNamespace).Get(ctx, repositoryServer.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)
	log.Print("---- Repo Server Pod Name ----", field.M{
		"Repo Server":     repositoryServer,
		"Repo Server Pod": repositoryServer.Status.ServerInfo.PodName,
	})
	time.Sleep(30 * time.Second)
	// Create Kopia Repository
	contentCacheMB, metadataCacheMB := kopiacmd.GetGeneralCacheSizeSettings()
	commandArgs := kopiacmd.RepositoryCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			RepoPassword:   testutil.DefaultRepositoryPassword,
			ConfigFilePath: kopiacmd.DefaultConfigFilePath,
			LogDirectory:   kopiacmd.DefaultLogDirectory,
		},
		CacheDirectory:  kopiacmd.DefaultCacheDirectory,
		Hostname:        testutil.DefaultRepositoryServerHost,
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
		Username:        testutil.DefaultKanisterAdminUser,
		RepoPathPrefix:  testutil.DefaultRepositoryPath,
		Location:        s3Location.Data,
	}
	log.Info().Print("----- Creating Kopia Repository ----")
	err = repository.ConnectToOrCreateKopiaRepository(
		s.cli,
		testutil.DefaultKanisterNamespace,
		repositoryServer.Status.ServerInfo.PodName,
		"repo-server-container",
		commandArgs,
	)
	if err != nil {
		log.Print(err.Error())
		return ""
	}
	log.Info().Print("----- Leaving Function ----")
	time.Sleep(20 * time.Minute)
	return repositoryServer.GetName()
}

func validateBlueprint(c *C, bp crv1alpha1.Blueprint, configMaps, secrets map[string]crv1alpha1.ObjectReference) {
	for _, action := range bp.Actions {
		// Validate BP action ConfigMapNames with the app.ConfigMaps references
		for _, bpc := range action.ConfigMapNames {
			validConfig := false
			for appc := range configMaps {
				if appc == bpc {
					validConfig = true
				}
			}
			c.Assert(validConfig, Equals, true)
		}
		// Validate BP action SecretNames with the app.Secrets reference
		for _, bps := range action.SecretNames {
			validSecret := false
			for apps := range secrets {
				if apps == bps {
					validSecret = true
				}
			}
			c.Assert(validSecret, Equals, true)
		}
	}
}

// createActionset creates and wait for actionset to complete
func (s *IntegrationSuite) createActionset(ctx context.Context, c *C, as *crv1alpha1.ActionSet, action string, options map[string]string) string {
	var err error
	switch action {
	case "backup":
		as.Spec.Actions[0].Options = options
		as, err = s.crCli.ActionSets(kontroller.namespace).Create(ctx, as, metav1.CreateOptions{})
		c.Assert(err, IsNil)
	case "restore", "delete":
		as, err = restoreActionSetSpecs(as, action)
		c.Assert(err, IsNil)
		as.Spec.Actions[0].Options = options
		if action == "delete" {
			// object of delete is always namespace of actionset
			as.Spec.Actions[0].Object = crv1alpha1.ObjectReference{
				APIVersion: "v1",
				Group:      "",
				Resource:   "namespaces",
				Kind:       "namespace",
				Name:       kontroller.namespace,
				Namespace:  "",
			}
		}
		as, err = s.crCli.ActionSets(kontroller.namespace).Create(ctx, as, metav1.CreateOptions{})
		c.Assert(err, IsNil)
	default:
		c.Errorf("Invalid action %s while creating ActionSet", action)
	}

	// Wait for the ActionSet to complete.
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err = s.crCli.ActionSets(kontroller.namespace).Get(ctx, as.GetName(), metav1.GetOptions{})
		switch {
		case err != nil, as.Status == nil:
			return false, err
		case as.Status.State == crv1alpha1.StateFailed:
			return true, errors.Errorf("Actionset failed: %#v", as.Status)
		case as.Status.State == crv1alpha1.StateComplete:
			return true, nil
		}
		return false, nil
	})
	c.Assert(err, IsNil)
	return as.GetName()
}

// restoreActionSetSpecs generates restore actionset specs from backup name
func restoreActionSetSpecs(from *crv1alpha1.ActionSet, action string) (*crv1alpha1.ActionSet, error) {
	params := kanctl.PerformParams{
		ActionName: action,
		ParentName: from.GetName(),
	}
	return kanctl.ChildActionSet(from, &params)
}

func createNamespace(cli kubernetes.Interface, name string) error {
	// Create Namespace
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (s *IntegrationSuite) TearDownSuite(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Uninstall app
	if !s.skip {
		err := s.app.Uninstall(ctx)
		c.Assert(err, IsNil)
	}

	// Uninstall implementation of the apps doesn't delete namespace
	// Delete the namespace separately
	err := s.cli.CoreV1().Namespaces().Delete(ctx, s.namespace, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func pingAppAndWait(ctx context.Context, a app.DatabaseApp) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, appWaitTimeout)
	defer waitCancel()
	err := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		err := a.Ping(ctx)
		return err == nil, nil
	})
	return err
}

func getServiceAccount(namespace, name string) *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func getClusterRole(namespace string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace + "-pod-reader",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "create"},
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/exec"},
			},
		},
	}
}

func getClusterRoleBinding(sa *v1.ServiceAccount, role *rbacv1.ClusterRole) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: sa.Namespace + "-global-pod-reader",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role.Name,
		},
	}
}
