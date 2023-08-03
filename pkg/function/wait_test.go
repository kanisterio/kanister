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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil"
)

var _ = Suite(&WaitSuite{})

type WaitSuite struct {
	cli         kubernetes.Interface
	namespace   string
	deploy      string
	statefulset string
}

func (s *WaitSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterwaittest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	d, err := s.cli.AppsV1().Deployments(cns.Name).Create(context.TODO(), testutil.NewTestDeployment(int32(1)), metav1.CreateOptions{})
	c.Assert(err, IsNil)
	sts, err := s.cli.AppsV1().StatefulSets(cns.Name).Create(context.TODO(), testutil.NewTestStatefulSet(int32(1)), metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name
	s.deploy = d.Name
	s.statefulset = sts.Name
}

func (s *WaitSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func waitNsPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitNsReady",
		Func: WaitFuncName,
		Args: map[string]interface{}{
			WaitTimeoutArg: "1m",
			WaitConditionsArg: map[string]interface{}{
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
		Func: WaitFuncName,
		Args: map[string]interface{}{
			WaitTimeoutArg: "10s",
			WaitConditionsArg: map[string]interface{}{
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

func waitDeployPhase(namespace, deploy string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitDeployReady",
		Func: WaitFuncName,
		Args: map[string]interface{}{
			WaitTimeoutArg: "5m",
			WaitConditionsArg: map[string]interface{}{
				"anyOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ if and (eq "{$.spec.replicas}" "{$.status.availableReplicas}" )
							(eq "{$.status.conditions[?(@.type == 'Available')].type}" "Available")
							(eq "{$.status.conditions[?(@.type == 'Available')].status}" "True") }}
							true
							{{ else }}
							false
							{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"group":      "apps",
							"resource":   "deployments",
							"name":       deploy,
							"namespace":  namespace,
						},
					},
				},
			},
		},
	}
}

func waitStatefulSetPhase(namespace, sts string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitStsReady",
		Func: WaitFuncName,
		Args: map[string]interface{}{
			WaitTimeoutArg: "5m",
			WaitConditionsArg: map[string]interface{}{
				"allOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ if (eq "{$.spec.replicas}" "{$.status.currentReplicas}" )}}
									true
								{{ else }}
									false
								{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"group":      "apps",
							"resource":   "statefulsets",
							"name":       sts,
							"namespace":  namespace,
						},
					},
					map[string]interface{}{
						"condition": `{{ if (eq "{$.spec.replicas}" "{$.status.readyReplicas}" )}}
									true
								{{ else }}
									false
								{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"group":      "apps",
							"resource":   "statefulsets",
							"name":       sts,
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
	tp := param.TemplateParams{
		Time: time.Now().String(),
	}
	action := "test"
	for _, tc := range []struct {
		bp      *crv1alpha1.Blueprint
		checker Checker
	}{
		{
			bp:      newWaitBlueprint(waitDeployPhase(s.namespace, s.deploy)),
			checker: IsNil,
		},
		{
			bp:      newWaitBlueprint(waitStatefulSetPhase(s.namespace, s.statefulset)),
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
			_, err := p.Exec(context.TODO(), *tc.bp, action, tp)
			c.Assert(err, tc.checker)
		}
	}
}
