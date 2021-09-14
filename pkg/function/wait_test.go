// Copyright 2021 The Kanister Authors.
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

package function

import (
	"context"
	"time"

	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/function/wait"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

var _ = Suite(&WaitSuite{})

type WaitSuite struct {
	cli       kubernetes.Interface
	namespace string
}

func (s *WaitSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterwaittest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	_, err = s.cli.AppsV1().Deployments(cns.Name).Create(context.TODO(), getDeploy(), metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *WaitSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func getDeploy() *appsv1.Deployment {
	replica := int32(2)
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replica,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []v1.ContainerPort{
								v1.ContainerPort{
									Name:          "http",
									HostPort:      0,
									ContainerPort: 80,
									Protocol:      v1.Protocol("TCP"),
								},
							},
							Resources:       v1.ResourceRequirements{},
							ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
						},
					},
				},
			},
		},
	}
}

func waitNsPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitNsReady",
		Func: wait.FuncName,
		Args: map[string]interface{}{
			wait.TimeoutArg: "1m",
			wait.ConditionsArg: map[string]interface{}{
				"anyOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ if (eq "{ $.status.phase }" "Invalid")}}true{{ else }}false{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"resource":   "namespaces",
							"name":       namespace,
						},
					},
					map[string]interface{}{
						"condition": `{{ if (eq "{ $.status.phase }" "Active")}}true{{ else }}false{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"resource":   "namespaces",
							"name":       namespace,
						},
					},
				},
			},
		},
	}
}

func waitNsTimeoutPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitNsReady",
		Func: wait.FuncName,
		Args: map[string]interface{}{
			wait.TimeoutArg: "10s",
			wait.ConditionsArg: map[string]interface{}{
				"allOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ if (eq "{$.status.phase}" "Inactive")}}true{{ else }}false{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"resource":   "namespaces",
							"name":       namespace,
						},
					},
					map[string]interface{}{
						"condition": `{{ if (eq "{$.status.phase}" "Invalid")}}true{{ else }}false{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"resource":   "namespaces",
							"name":       namespace,
						},
					},
				},
			},
		},
	}
}

func waitDeployPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitDeployReady",
		Func: wait.FuncName,
		Args: map[string]interface{}{
			wait.TimeoutArg: "1m",
			wait.ConditionsArg: map[string]interface{}{
				"anyOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ if and (eq {$.spec.replicas} {$.status.availableReplicas} )
									(and (eq "{$.status.conditions[?(@.type == "Available")].type}" "Available")
									(eq "{$.status.conditions[?(@.type == "Available")].status}" "True"))}}
									true
								{{ else }}
									false
								{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"group":      "apps",
							"resource":   "deployments",
							"name":       getDeploy().GetName(),
							"namespace":  namespace,
						},
					},
				},
			},
		},
	}
}

func newWaitBlueprint(phases ...crv1alpha1.BlueprintPhase) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Phases: phases,
			},
		},
	}
}

func (s *WaitSuite) TestWait(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tp := param.TemplateParams{}
	action := "test"
	for _, tc := range []struct {
		bp      *crv1alpha1.Blueprint
		checker Checker
	}{
		{
			bp:      newWaitBlueprint(waitDeployPhase(s.namespace)),
			checker: IsNil,
		},
		{
			bp:      newWaitBlueprint(waitNsPhase(s.namespace)),
			checker: IsNil,
		},
		{
			bp:      newWaitBlueprint(waitNsTimeoutPhase(s.namespace)),
			checker: NotNil,
		},
	} {
		phases, err := kanister.GetPhases(*tc.bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			_, err := p.Exec(ctx, *tc.bp, action, tp)
			c.Assert(err, tc.checker)
		}
	}
}
