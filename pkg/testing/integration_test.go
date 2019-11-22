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
	"context"
	"time"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type secretProfile struct {
	secret  *v1.Secret
	profile *crv1alpha1.Profile
}

type IntegrationSuite struct {
	name      string
	cli       kubernetes.Interface
	crCli     crclient.CrV1alpha1Interface
	app       app.App
	bp        app.Blueprinter
	profile   *secretProfile
	namespace string
	skip      bool
	cancel    context.CancelFunc
}

// INTEGRATION TEST APPLICATIONS

// rds-postgres app
var _ = Suite(&IntegrationSuite{
	name:      "rds-postgres",
	namespace: "rds-postgres-test",
	app:       app.NewRDSPostgresDB("rds-postgres"),
	bp:        app.NewBlueprint("rds-postgres"),
	profile:   newSecretProfile("", "", ""),
})

// pitr-postgresql app
var _ = Suite(&IntegrationSuite{
	name:      "pitr-postgres",
	namespace: "pitr-postgres-test",
	app:       app.NewPostgresDB("pitr-postgres"),
	bp:        app.NewPITRBlueprint("pitr-postgres"),
	profile:   newSecretProfile("infracloud.kanister.io", "", ""),
})

// postgres app
var _ = Suite(&IntegrationSuite{
	name:      "postgres",
	namespace: "postgres-test",
	app:       app.NewPostgresDB("postgres"),
	bp:        app.NewBlueprint("postgres"),
	profile:   newSecretProfile("infracloud.kanister.io", "", ""),
})

func newSecretProfile(bucket, endpoint, prefix string) *secretProfile {
	_, location := testutil.GetObjectstoreLocation()
	location.Bucket = bucket
	location.Endpoint = endpoint
	location.Prefix = prefix

	secret, profile, err := testutil.NewSecretProfileFromLocation(location)
	if err != nil {
		return nil
	}
	return &secretProfile{
		secret:  secret,
		profile: profile,
	}
}

func (s *IntegrationSuite) SetUpSuite(c *C) {
	ctx := context.Background()
	ctx, s.cancel = context.WithCancel(ctx)

	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	s.cli, err = kubernetes.NewForConfig(cfg)
	c.Assert(err, IsNil)
	s.crCli, err = crclient.NewForConfig(cfg)
	c.Assert(err, IsNil)

	// Start the controller
	err = resource.CreateCustomResources(ctx, cfg)
	c.Assert(err, IsNil)
	ctlr := controller.New(cfg)
	err = ctlr.StartWatch(ctx, s.namespace)
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
	log.Info().Print("Running e2e integration test.", field.M{"app": s.name})

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

	// Create profile
	if s.profile == nil {
		log.Info().Print("Skipping integration test. Could not create profile. Please check if required credentials are set.", field.M{"app": s.name})
		s.skip = true
		c.Skip("Could not create a Profile")
	}
	profileName := s.createProfile(c)

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
	_, err = s.crCli.Blueprints(s.namespace).Create(bp)
	c.Assert(err, IsNil)

	var configMaps, secrets map[string]crv1alpha1.ObjectReference
	testEntries := 3
	// Add test entries to DB
	if a, ok := s.app.(app.DatabaseApp); ok {
		err = a.Ping(ctx)
		c.Assert(err, IsNil)

		err = a.Reset(ctx)
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

	// Create ActionSet specs
	as := newActionSet(bp.GetName(), profileName, s.namespace, s.app.Object(), configMaps, secrets)
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

		count, err := a.Count(ctx)
		c.Assert(err, IsNil)
		c.Assert(count, Equals, 0)
	}

	// Restore backup
	pas, err := s.crCli.ActionSets(s.namespace).Get(backup, metav1.GetOptions{})
	c.Assert(err, IsNil)
	s.createActionset(ctx, c, pas, "restore", restoreOptions)

	// Verify data
	if a, ok := s.app.(app.DatabaseApp); ok {
		err = a.Ping(ctx)
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
				crv1alpha1.ActionSpec{
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

func (s *IntegrationSuite) createProfile(c *C) string {
	secret, err := s.cli.CoreV1().Secrets(s.namespace).Create(s.profile.secret)
	c.Assert(err, IsNil)

	// set secret ref in profile
	s.profile.profile.Credential.KeyPair.Secret = crv1alpha1.ObjectReference{
		Name:      secret.GetName(),
		Namespace: secret.GetNamespace(),
	}
	profile, err := s.crCli.Profiles(s.namespace).Create(s.profile.profile)
	c.Assert(err, IsNil)

	return profile.GetName()
}

func validateBlueprint(c *C, bp crv1alpha1.Blueprint, configMaps, secrets map[string]crv1alpha1.ObjectReference) {
	for _, action := range bp.Actions {
		// Validate BP action ConfigMapNames with the app.ConfigMaps references
		for _, bpc := range action.ConfigMapNames {
			validConfig := false
			for appc, _ := range configMaps {
				if appc == bpc {
					validConfig = true
				}
			}
			c.Assert(validConfig, Equals, true)
		}
		// Validate BP action SecretNames with the app.Secrets reference
		for _, bps := range action.SecretNames {
			validSecret := false
			for apps, _ := range secrets {
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
		as, err = s.crCli.ActionSets(s.namespace).Create(as)
		c.Assert(err, IsNil)
	case "restore", "delete":
		as, err = restoreActionSetSpecs(as, action)
		c.Assert(err, IsNil)
		as.Spec.Actions[0].Options = options
		as, err = s.crCli.ActionSets(s.namespace).Create(as)
		c.Assert(err, IsNil)
	default:
		c.Errorf("Invalid action %s while creating ActionSet", action)
	}

	// Wait for the ActionSet to complete.
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err = s.crCli.ActionSets(s.namespace).Get(as.GetName(), metav1.GetOptions{})
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
	_, err := cli.CoreV1().Namespaces().Create(ns)
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

	// Delete namespace
	s.cli.CoreV1().Namespaces().Delete(s.namespace, nil)
	if s.cancel != nil {
		s.cancel()
	}
}
