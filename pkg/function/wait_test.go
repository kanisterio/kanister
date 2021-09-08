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

package function

import (
	"context"
	"time"

	. "gopkg.in/check.v1"
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
	s.namespace = cns.Name
}

func (s *WaitSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
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
						"condition": `{{ if (eq $.status.phase "Active")}}true{{ else }}false{{ end }}`,
						"objectReference": map[string]interface{}{
							"apiVersion": "v1",
							"resource":   "namespaces",
							"namespace":  namespace,
							"name":       namespace,
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
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{}
	action := "test"
	for _, tc := range []struct {
		bp *crv1alpha1.Blueprint
	}{
		{
			bp: newWaitBlueprint(waitNsPhase(s.namespace)),
		},
	} {
		phases, err := kanister.GetPhases(*tc.bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			_, err := p.Exec(ctx, *tc.bp, action, tp)
			c.Assert(err, IsNil, Commentf("Phase %s failed", p.Name()))
		}
	}
}
