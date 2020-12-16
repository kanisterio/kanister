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

package testutil

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
)

// NewTestPVC function returns a pointer to a new PVC test object
func NewTestPVC() *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}
}

// NewTestNamespace function returns a pointer to a new Namespace test object
func NewTestNamespace() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest-",
		},
	}
}

// NewTestDeployment function returns a pointer to a new Deployment test object
func NewTestDeployment(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-deployment-",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "fake-app"}},
			Template: newTestPodTemplateSpec(),
		},
	}
}

// NewTestStatefulSet function returns a pointer to a new StatefulSet test object
func NewTestStatefulSet(replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-statefulset-",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "fake-svc",
			Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app": "fake-app"}},
			Template:    newTestPodTemplateSpec(),
		},
	}
}

func newTestPodTemplateSpec() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "fake-app",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:    "test-container",
					Image:   "ghcr.io/kanisterio/kanister-tools:0.45.0",
					Command: []string{"tail"},
					Args:    []string{"-f", "/dev/null"},
				},
			},
		},
	}
}

const TestProfileName = "test-profile"

// NewTestProfileSecret function returns a pointer to a new Secret test object.
func NewTestProfileSecret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: map[string]string{
			"id":     "foo",
			"secret": "bar",
		},
	}
}

// NewTestProfileSecretWithRole function returns a pointer to a new Secret test object with role.
func NewTestProfileSecretWithRole(role string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		Data: map[string][]byte{
			"id":     []byte("foo"),
			"secret": []byte("bar"),
			"role":   []byte(role),
		},
	}
}

// NewTestProfile function returns a pointer to a new Profile test object that
// passes validation.
func NewTestProfile(namespace string, secretName string) *crv1alpha1.Profile {
	return &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestProfileName,
			Namespace: namespace,
		},
		Location: crv1alpha1.Location{
			Type: crv1alpha1.LocationTypeS3Compliant,
		},
		Credential: crv1alpha1.Credential{
			Type: crv1alpha1.CredentialTypeKeyPair,
			KeyPair: &crv1alpha1.KeyPair{
				Secret: crv1alpha1.ObjectReference{
					Name:      secretName,
					Namespace: namespace,
				},
				IDField:     "id",
				SecretField: "secret",
			},
		},
	}
}

// NewSecretProfileFromLocation figures out Provider type from the location type and
// returns a pointer to Secret and Profile
func NewSecretProfileFromLocation(location crv1alpha1.Location) (*v1.Secret, *crv1alpha1.Profile, error) {
	var key, val string
	data := make(map[string]string)

	switch location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		key = os.Getenv(awsconfig.AccessKeyID)
		val = os.Getenv(awsconfig.SecretAccessKey)
		if role := os.Getenv(awsconfig.ConfigRole); role != "" {
			data["role"] = role
		}
	case crv1alpha1.LocationTypeGCS:
		os.Getenv(blockstorage.GoogleCloudCreds)
		creds, err := google.FindDefaultCredentials(context.Background(), compute.ComputeScope)
		if err != nil {
			return nil, nil, err
		}
		key = creds.ProjectID
		val = string(creds.JSON)
	case crv1alpha1.LocationTypeAzure:
		key = os.Getenv(blockstorage.AzureStorageAccount)
		val = os.Getenv(blockstorage.AzureStorageKey)
	default:
		return nil, nil, fmt.Errorf("Invalid location type '%s'", location.Type)
	}
	data["access_key_id"] = key
	data["secret_access_key"] = val
	return NewProfileSecret(data), NewProfile(location), nil
}

// NewProfileSecret function returns a pointer to a new Secret test object.
func NewProfileSecret(data map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: data,
	}
}

// NewProfile function returns a pointer to a new Profile object that
// passes validation.
func NewProfile(location crv1alpha1.Location) *crv1alpha1.Profile {
	return &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-profile-",
		},
		Location: location,
		Credential: crv1alpha1.Credential{
			Type: crv1alpha1.CredentialTypeKeyPair,
			KeyPair: &crv1alpha1.KeyPair{
				IDField:     "access_key_id",
				SecretField: "secret_access_key",
			},
		},
	}
}

// NewTestActionSet function returns a pointer to a new ActionSet test object
func NewTestActionSet(namespace, blueprintName, poKind, poName, poNamespace, version string) *crv1alpha1.ActionSet {
	return &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
			Namespace:    namespace,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				crv1alpha1.ActionSpec{
					Blueprint: blueprintName,
					Name:      "myAction",
					Object: crv1alpha1.ObjectReference{
						Kind:      poKind,
						Name:      poName,
						Namespace: poNamespace,
					},
					Profile: &crv1alpha1.ObjectReference{
						Kind:      crv1alpha1.ProfileResourceName,
						Name:      TestProfileName,
						Namespace: namespace,
					},
					PreferredVersion: version,
				},
			},
		},
	}
}

// NewTestConfigMap function returns a pointer to a new ConfigMap test object
func NewTestConfigMap() *v1.ConfigMap {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-configmap-",
		},
		Data: map[string]string{"myKey": "myValue"},
	}
	return cm
}

const actionName = "myAction"

// NewTestBlueprint function returns a pointer to a new Blueprint test object
func NewTestBlueprint(poKind string, phaseFuncs ...string) *crv1alpha1.Blueprint {
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			actionName: &crv1alpha1.BlueprintAction{
				Kind:   "StatefulSet",
				Phases: make([]crv1alpha1.BlueprintPhase, 0, len(phaseFuncs)),
			},
		},
	}
	for _, bpa := range bp.Actions {
		for i, pf := range phaseFuncs {
			bpa.Phases = append(bpa.Phases, crv1alpha1.BlueprintPhase{
				Name: fmt.Sprintf("myPhase-%d", i),
				Func: pf,
			})
		}
	}
	return bp
}

// ActionSetWithConfigMap function returns a pointer to a new ActionSet test object with CongigMap
func ActionSetWithConfigMap(as *crv1alpha1.ActionSet, name string) *crv1alpha1.ActionSet {
	as.Spec.Actions[0].ConfigMaps = map[string]crv1alpha1.ObjectReference{
		"myCM": crv1alpha1.ObjectReference{
			Name:      name,
			Namespace: as.GetNamespace(),
		},
	}
	return as
}

// BlueprintWithConfigMap function returns a pointer to a new Blueprint test object with CongigMap
func BlueprintWithConfigMap(bp *crv1alpha1.Blueprint) *crv1alpha1.Blueprint {
	cmArgs := map[string]interface{}{"key": "{{ .ConfigMaps.myCM.Data.myKey  }}"}
	for i := range bp.Actions[actionName].Phases {
		bp.Actions[actionName].Phases[i].Args = cmArgs
	}
	return bp
}
