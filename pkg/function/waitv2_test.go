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

var _ = Suite(&WaitV2Suite{})

type WaitV2Suite struct {
	cli         kubernetes.Interface
	namespace   string
	deploy      string
	statefulset string
}

func (s *WaitV2Suite) SetUpSuite(c *C) {
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

func (s *WaitV2Suite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func waitV2NsPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitV2NsReady",
		Func: WaitV2FuncName,
		Args: map[string]interface{}{
			WaitV2TimeoutArg: "1m",
			WaitV2ConditionsArg: map[string]interface{}{
				"anyOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ if (eq .status.phase "Invalid")}}true{{ else }}false{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"resource":   "namespaces",
							"name":       namespace,
						},
					},
					map[string]interface{}{
						"condition": `{{ if (eq .status.phase "Active")}}true{{ else }}false{{ end }}`,
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

func waitV2NsTimeoutPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitV2NsReady",
		Func: WaitV2FuncName,
		Args: map[string]interface{}{
			WaitV2TimeoutArg: "10s",
			WaitV2ConditionsArg: map[string]interface{}{
				"allOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ if (eq .status.phase "Inactive")}}true{{ else }}false{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"resource":   "namespaces",
							"name":       namespace,
						},
					},
					map[string]interface{}{
						"condition": `{{ if (eq .status.phase "Invalid")}}true{{ else }}false{{ end }}`,
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

func waitV2DeployPhase(namespace, deploy string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitV2DeployReady",
		Func: WaitV2FuncName,
		Args: map[string]interface{}{
			WaitV2TimeoutArg: "5m",
			WaitV2ConditionsArg: map[string]interface{}{
				"anyOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ $available := false }}{{ range $condition := $.status.conditions }}{{ if and (eq .type "Available") (eq .status "True") }}{{ $available = true }}{{ end }}{{ end }}{{ $available }}`,
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

func waitV2StatefulSetPhase(namespace, sts string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "waitV2StsReady",
		Func: WaitV2FuncName,
		Args: map[string]interface{}{
			WaitV2TimeoutArg: "5m",
			WaitV2ConditionsArg: map[string]interface{}{
				"allOf": []interface{}{
					map[string]interface{}{
						"condition": `{{ if (eq .spec.replicas .status.currentReplicas )}}true{{ else }}false{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"group":      "apps",
							"resource":   "statefulsets",
							"name":       sts,
							"namespace":  namespace,
						},
					},
					map[string]interface{}{
						"condition": `{{ if (eq .spec.replicas .status.readyReplicas )}}true{{ else }}false{{ end }}`,
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

func newWaitV2Blueprint(phases ...crv1alpha1.BlueprintPhase) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Phases: phases,
			},
		},
	}
}

func (s *WaitV2Suite) TestWaitV2(c *C) {
	tp := param.TemplateParams{
		Time: time.Now().String(),
	}
	action := "test"
	for _, tc := range []struct {
		bp      *crv1alpha1.Blueprint
		checker Checker
	}{
		{
			bp:      newWaitV2Blueprint(waitV2DeployPhase(s.namespace, s.deploy)),
			checker: IsNil,
		},
		{
			bp:      newWaitV2Blueprint(waitV2StatefulSetPhase(s.namespace, s.statefulset)),
			checker: IsNil,
		},
		{
			bp:      newWaitV2Blueprint(waitV2NsPhase(s.namespace)),
			checker: IsNil,
		},
		{
			bp:      newWaitV2Blueprint(waitV2NsTimeoutPhase(s.namespace)),
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
