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

package validate

import (
	"context"
	"testing"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ValidateSuite struct{}

var _ = Suite(&ValidateSuite{})

func (s *ValidateSuite) TestActionSet(c *C) {
	for _, tc := range []struct {
		as      *crv1alpha1.ActionSet
		checker Checker
	}{
		{
			as:      &crv1alpha1.ActionSet{},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec:       &crv1alpha1.ActionSetSpec{},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
							ConfigMaps: map[string]crv1alpha1.ObjectReference{
								"testCM": {
									Namespace: "ns2",
								},
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
							ConfigMaps: map[string]crv1alpha1.ObjectReference{
								"testCM": {
									Namespace: "ns1",
								},
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
							Secrets: map[string]crv1alpha1.ObjectReference{
								"testSecrets": {
									Namespace: "ns2",
								},
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
							Secrets: map[string]crv1alpha1.ObjectReference{
								"testSecrets": {
									Namespace: "ns1",
								},
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec:   &crv1alpha1.ActionSetSpec{},
				Status: &crv1alpha1.ActionSetStatus{},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{},
				Status: &crv1alpha1.ActionSetStatus{
					State: crv1alpha1.StatePending,
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
						},
					},
				},
				Status: &crv1alpha1.ActionSetStatus{
					State: crv1alpha1.StatePending,
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
						},
					},
				},
				Status: &crv1alpha1.ActionSetStatus{
					State: crv1alpha1.StatePending,
					Actions: []crv1alpha1.ActionStatus{
						{},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSet{
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "ns1",
								Kind: param.NamespaceKind,
							},
						},
					},
				},
				Status: &crv1alpha1.ActionSetStatus{
					State: crv1alpha1.StatePending,
					Actions: []crv1alpha1.ActionStatus{
						{},
					},
				},
			},
			checker: IsNil,
		},
		// NamespaceKind
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: param.NamespaceKind,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		// StatefulSetKind
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: param.StatefulSetKind,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		// DeploymentKind
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: param.DeploymentKind,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		// PVCKind
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: param.PVCKind,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		// Generic K8s resource (apiversion, resource missing)
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name: "foo",
								Kind: "unknown",
							},
						},
					},
				},
			},
			checker: NotNil,
		},
		// Generic K8s resource
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{
							Object: crv1alpha1.ObjectReference{
								Name:       "foo",
								APIVersion: "v1",
								Resource:   "serviceaccount",
							},
						},
					},
				},
			},
			checker: IsNil,
		}, // No object specified
		{
			as: &crv1alpha1.ActionSet{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
				Spec: &crv1alpha1.ActionSetSpec{
					Actions: []crv1alpha1.ActionSpec{
						{},
					},
				},
			},
			checker: NotNil,
		},
	} {
		err := ActionSet(tc.as)
		c.Check(err, tc.checker)
	}
}

func (s *ValidateSuite) TestActionSetStatus(c *C) {
	for _, tc := range []struct {
		as      *crv1alpha1.ActionSetStatus
		checker Checker
	}{
		{
			as:      nil,
			checker: IsNil,
		},
		{
			as:      &crv1alpha1.ActionSetStatus{},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					{},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					{},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					{
						Phases: []crv1alpha1.Phase{},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					{
						Phases: []crv1alpha1.Phase{
							{},
						},
					},
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					{
						Phases: []crv1alpha1.Phase{
							{},
						},
					},
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StatePending,
				Actions: []crv1alpha1.ActionStatus{
					{
						Phases: []crv1alpha1.Phase{
							{
								State: crv1alpha1.StatePending,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StateFailed,
				Actions: []crv1alpha1.ActionStatus{
					{
						Phases: []crv1alpha1.Phase{
							{
								State: crv1alpha1.StatePending,
							},
						},
					},
				},
			},
			checker: IsNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StateComplete,
				Actions: []crv1alpha1.ActionStatus{
					{
						Phases: []crv1alpha1.Phase{
							{
								State: crv1alpha1.StatePending,
							},
						},
					},
				},
			},
			checker: NotNil,
		},
		{
			as: &crv1alpha1.ActionSetStatus{
				State: crv1alpha1.StateComplete,
				Actions: []crv1alpha1.ActionStatus{
					{
						Phases: []crv1alpha1.Phase{
							{
								State: crv1alpha1.StatePending,
							},
							{
								State: crv1alpha1.StateComplete,
							},
						},
					},
				},
			},
			checker: NotNil,
		},
	} {
		err := actionSetStatus(tc.as)
		c.Check(err, tc.checker)
	}
}

func (s *ValidateSuite) TestBlueprint(c *C) {
	err := Blueprint(nil)
	c.Assert(err, IsNil)
}

func (s *ValidateSuite) TestProfileSchema(c *C) {
	tcs := []struct {
		profile *crv1alpha1.Profile
		checker Checker
	}{
		{
			profile: &crv1alpha1.Profile{
				Location: crv1alpha1.Location{
					Type: crv1alpha1.LocationTypeS3Compliant,
				},
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeSecret,
					Secret: &crv1alpha1.ObjectReference{
						Name:      "secret-name",
						Namespace: "secret-namespace",
					},
				},
			},
			checker: IsNil,
		},
		{
			profile: &crv1alpha1.Profile{
				Location: crv1alpha1.Location{
					Type: crv1alpha1.LocationTypeS3Compliant,
				},
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeKeyPair,
					KeyPair: &crv1alpha1.KeyPair{
						IDField:     "id",
						SecretField: "secret",
						Secret: crv1alpha1.ObjectReference{
							Name:      "secret-name",
							Namespace: "secret-namespace",
						},
					},
				},
			},
			checker: IsNil,
		},
		// Missing secret namespace
		{
			profile: &crv1alpha1.Profile{
				Location: crv1alpha1.Location{
					Type: crv1alpha1.LocationTypeS3Compliant,
				},
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeSecret,
					Secret: &crv1alpha1.ObjectReference{
						Name: "secret-name",
					},
				},
			},
			checker: NotNil,
		},
		// Missing secret name
		{
			profile: &crv1alpha1.Profile{
				Location: crv1alpha1.Location{
					Type: crv1alpha1.LocationTypeS3Compliant,
				},
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeSecret,
					Secret: &crv1alpha1.ObjectReference{
						Namespace: "secret-namespace",
					},
				},
			},
			checker: NotNil,
		},
		// Missing secret field
		{
			profile: &crv1alpha1.Profile{
				Location: crv1alpha1.Location{
					Type: crv1alpha1.LocationTypeS3Compliant,
				},
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeKeyPair,
					KeyPair: &crv1alpha1.KeyPair{
						IDField: "id",
						Secret: crv1alpha1.ObjectReference{
							Name:      "secret-name",
							Namespace: "secret-namespace",
						},
					},
				},
			},
			checker: NotNil,
		},
		// Missing id field
		{
			profile: &crv1alpha1.Profile{
				Location: crv1alpha1.Location{
					Type: crv1alpha1.LocationTypeS3Compliant,
				},
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeKeyPair,
					KeyPair: &crv1alpha1.KeyPair{
						SecretField: "secret",
						Secret: crv1alpha1.ObjectReference{
							Name:      "secret-name",
							Namespace: "secret-namespace",
						},
					},
				},
			},
			checker: NotNil,
		},
	}

	for _, tc := range tcs {
		err := ProfileSchema(tc.profile)
		c.Check(err, tc.checker)
	}
}

func (s *ValidateSuite) TestOsSecretFromProfile(c *C) {
	ctx := context.Background()
	for i, tc := range []struct {
		pType      objectstore.ProviderType
		p          *crv1alpha1.Profile
		cli        kubernetes.Interface
		expected   *objectstore.Secret
		errChecker Checker
	}{
		{
			p: &crv1alpha1.Profile{
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeSecret,
					Secret: &crv1alpha1.ObjectReference{
						Name:      "secname",
						Namespace: "secnamespace",
					},
				},
			},
			pType: objectstore.ProviderTypeAzure,
			cli: fake.NewSimpleClientset(&corev1.Secret{
				Type: corev1.SecretType(secrets.AzureSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secname",
					Namespace: "secnamespace",
				},
				Data: map[string][]byte{
					secrets.AzureStorageAccountID:   []byte("said"),
					secrets.AzureStorageAccountKey:  []byte("sakey"),
					secrets.AzureStorageEnvironment: []byte("env"),
				},
			}),
			expected: &objectstore.Secret{
				Type: objectstore.SecretTypeAzStorageAccount,
				Azure: &objectstore.SecretAzure{
					StorageAccount:  "said",
					StorageKey:      "sakey",
					EnvironmentName: "env",
				},
			},
			errChecker: IsNil,
		},
		{
			p: &crv1alpha1.Profile{
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeKeyPair,
					KeyPair: &crv1alpha1.KeyPair{
						IDField:     secrets.AzureStorageAccountID,
						SecretField: secrets.AzureStorageAccountKey,
						Secret: crv1alpha1.ObjectReference{
							Name:      "secname",
							Namespace: "secnamespace",
						},
					},
				},
			},
			pType: objectstore.ProviderTypeAzure,
			cli: fake.NewSimpleClientset(&corev1.Secret{
				Type: corev1.SecretType(secrets.AzureSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secname",
					Namespace: "secnamespace",
				},
				Data: map[string][]byte{
					secrets.AzureStorageAccountID:   []byte("said"),
					secrets.AzureStorageAccountKey:  []byte("sakey"),
					secrets.AzureStorageEnvironment: []byte("env"),
				},
			}),
			expected: &objectstore.Secret{
				Type: objectstore.SecretTypeAzStorageAccount,
				Azure: &objectstore.SecretAzure{
					StorageAccount:  "said",
					StorageKey:      "sakey",
					EnvironmentName: "",
				},
			},
			errChecker: IsNil,
		},
		{ // bad secret field err
			p: &crv1alpha1.Profile{
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeKeyPair,
					KeyPair: &crv1alpha1.KeyPair{
						IDField:     secrets.AzureStorageAccountID,
						SecretField: secrets.AWSSecretAccessKey, // bad field
						Secret: crv1alpha1.ObjectReference{
							Name:      "secname",
							Namespace: "secnamespace",
						},
					},
				},
			},
			pType: objectstore.ProviderTypeAzure,
			cli: fake.NewSimpleClientset(&corev1.Secret{
				Type: corev1.SecretType(secrets.AzureSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secname",
					Namespace: "secnamespace",
				},
				Data: map[string][]byte{
					secrets.AzureStorageAccountID:   []byte("said"),
					secrets.AzureStorageAccountKey:  []byte("sakey"),
					secrets.AzureStorageEnvironment: []byte("env"),
				},
			}),
			expected:   nil,
			errChecker: NotNil,
		},
		{ // bad id field err
			p: &crv1alpha1.Profile{
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeKeyPair,
					KeyPair: &crv1alpha1.KeyPair{
						IDField:     "badidfield",
						SecretField: secrets.AzureStorageAccountKey,
						Secret: crv1alpha1.ObjectReference{
							Name:      "secname",
							Namespace: "secnamespace",
						},
					},
				},
			},
			pType: objectstore.ProviderTypeAzure,
			cli: fake.NewSimpleClientset(&corev1.Secret{
				Type: corev1.SecretType(secrets.AzureSecretType),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secname",
					Namespace: "secnamespace",
				},
				Data: map[string][]byte{
					secrets.AzureStorageAccountID:   []byte("said"),
					secrets.AzureStorageAccountKey:  []byte("sakey"),
					secrets.AzureStorageEnvironment: []byte("env"),
				},
			}),
			expected:   nil,
			errChecker: NotNil,
		},
		{ // missing secret
			p: &crv1alpha1.Profile{
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeKeyPair,
					KeyPair: &crv1alpha1.KeyPair{
						IDField:     "badidfield",
						SecretField: secrets.AzureStorageAccountKey,
						Secret: crv1alpha1.ObjectReference{
							Name:      "secname",
							Namespace: "secnamespace",
						},
					},
				},
			},
			pType:      objectstore.ProviderTypeAzure,
			cli:        fake.NewSimpleClientset(),
			expected:   nil,
			errChecker: NotNil,
		},
		{ // missing keypair
			p: &crv1alpha1.Profile{
				Credential: crv1alpha1.Credential{
					Type:    crv1alpha1.CredentialTypeKeyPair,
					KeyPair: nil,
				},
			},
			pType:      objectstore.ProviderTypeAzure,
			cli:        fake.NewSimpleClientset(),
			expected:   nil,
			errChecker: NotNil,
		},
		{ // missing secret
			p: &crv1alpha1.Profile{
				Credential: crv1alpha1.Credential{
					Type: crv1alpha1.CredentialTypeSecret,
					Secret: &crv1alpha1.ObjectReference{
						Name:      "secname",
						Namespace: "secnamespace",
					},
				},
			},
			pType:      objectstore.ProviderTypeAzure,
			cli:        fake.NewSimpleClientset(),
			expected:   nil,
			errChecker: NotNil,
		},
	} {
		secret, err := osSecretFromProfile(ctx, tc.pType, tc.p, tc.cli)
		c.Check(secret, DeepEquals, tc.expected, Commentf("test number: %d", i))
		c.Check(err, tc.errChecker)
	}
}
