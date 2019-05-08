package testutil

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
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
					v1.ResourceStorage: *resource.NewQuantity(1, resource.BinarySI),
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
func NewTestDeployment() *appsv1.Deployment {
	var replicas int32 = 1
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
func NewTestStatefulSet() *appsv1.StatefulSet {
	var replicas int32 = 1
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
					Image:   "kanisterio/kanister-tools:0.19.0",
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

// NewTestActionSet function returns a pointer to a new ActionSet test object
func NewTestActionSet(namespace, blueprintName, poKind, poName, poNamespace string) *crv1alpha1.ActionSet {
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
