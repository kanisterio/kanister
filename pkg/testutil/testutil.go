package testutil

import (
	"fmt"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// NewTestNamespace function returns a pointer to a new Namespace test object
func NewTestNamespace() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest-",
		},
	}
}

// NewTestDeployment function returns a pointer to a new Deployment test object
func NewTestDeployment() *v1beta1.Deployment {
	var replicas int32 = 1
	return &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-deployment-",
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: newTestPodTemplateSpec(),
		},
	}
}

// NewTestStatefulSet function returns a pointer to a new StatefulSet test object
func NewTestStatefulSet() *v1beta1.StatefulSet {
	var replicas int32 = 1
	return &v1beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-statefulset-",
		},
		Spec: v1beta1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "fake-svc",
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
					Image:   "alpine:3.6",
					Command: []string{"tail"},
					Args:    []string{"-f", "/dev/null"},
				},
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
	cmArgs := []string{"{{ .ConfigMaps.myCM.Data.myKey  }}"}
	for i := range bp.Actions[actionName].Phases {
		bp.Actions[actionName].Phases[i].Args = cmArgs
	}
	return bp
}
