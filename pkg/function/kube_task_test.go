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
	"fmt"
	"os"
	"strings"
	"time"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

var _ = Suite(&KubeTaskSuite{})

type KubeTaskSuite struct {
	cli       kubernetes.Interface
	namespace string
}

func (s *KubeTaskSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterkubetasktest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name
	err = os.Setenv("POD_NAMESPACE", cns.Name)
	c.Assert(err, IsNil)
	err = os.Setenv("POD_SERVICE_ACCOUNT", "default")
	c.Assert(err, IsNil)
}

func (s *KubeTaskSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func bigOutputPhase(namespace string) crv1alpha1.BlueprintPhase {
	longstring := strings.Repeat("a", 100000)
	return crv1alpha1.BlueprintPhase{
		Name: "testOutput",
		Func: KubeTaskFuncName,
		Args: map[string]interface{}{
			KubeTaskNamespaceArg: namespace,
			KubeTaskImageArg:     consts.LatestKanisterToolsImage,
			KubeTaskCommandArg: []string{
				"sh",
				"-c",
				// We output a line for log only, and a line with output at the tail
				fmt.Sprintf("echo -n %s > tmpfile; cat tmpfile; echo; cat tmpfile; kando output longstring $(cat tmpfile)", longstring),
			},
		},
	}
}

func outputPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testOutput",
		Func: KubeTaskFuncName,
		Args: map[string]interface{}{
			KubeTaskNamespaceArg: namespace,
			KubeTaskImageArg:     consts.LatestKanisterToolsImage,
			KubeTaskCommandArg: []string{
				"sh",
				"-c",
				"kando output version 0.110.0",
			},
		},
	}
}

func sleepPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testSleep",
		Func: KubeTaskFuncName,
		Args: map[string]interface{}{
			KubeTaskNamespaceArg: namespace,
			KubeTaskImageArg:     "ubuntu:latest",
			KubeTaskCommandArg: []string{
				"sleep",
				"2",
			},
		},
	}
}

func tickPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testTick",
		Func: KubeTaskFuncName,
		Args: map[string]interface{}{
			KubeTaskNamespaceArg: namespace,
			KubeTaskImageArg:     "alpine:3.10",
			KubeTaskCommandArg: []string{
				"sh",
				"-c",
				`for i in $(seq 3); do echo Tick: "${i}"; sleep 1; done`,
			},
		},
	}
}

func newTaskBlueprint(phases ...crv1alpha1.BlueprintPhase) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Kind:   "StatefulSet",
				Phases: phases,
			},
		},
	}
}

func (s *KubeTaskSuite) TestKubeTask(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
		PodOverride: crv1alpha1.JSONMap{
			"containers": []map[string]interface{}{
				{
					"name":            "container",
					"imagePullPolicy": "Always",
				},
			},
		},
	}
	action := "test"
	for _, tc := range []struct {
		bp   *crv1alpha1.Blueprint
		outs []map[string]interface{}
	}{
		{
			bp: newTaskBlueprint(outputPhase(s.namespace), sleepPhase(s.namespace), tickPhase(s.namespace)),
			outs: []map[string]interface{}{
				{
					"version": "0.110.0",
				},
				{},
				{},
			},
		},
	} {
		phases, err := kanister.GetPhases(*tc.bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, IsNil)
		c.Assert(phases, HasLen, len(tc.outs))
		for i, p := range phases {
			out, err := p.Exec(ctx, *tc.bp, action, tp)
			c.Assert(err, IsNil, Commentf("Phase %s failed", p.Name()))
			c.Assert(out, DeepEquals, tc.outs[i])
		}
	}
}

func (s *KubeTaskSuite) TestKubeTaskWithBigOutput(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
		PodOverride: crv1alpha1.JSONMap{
			"containers": []map[string]interface{}{
				{
					"name":            "container",
					"imagePullPolicy": "Always",
				},
			},
		},
	}
	expectedOut := strings.Repeat("a", 100000)
	action := "test"
	for _, tc := range []struct {
		bp   *crv1alpha1.Blueprint
		outs []map[string]interface{}
	}{
		{
			bp: newTaskBlueprint(bigOutputPhase(s.namespace)),
			outs: []map[string]interface{}{
				{
					"longstring": expectedOut,
				},
			},
		},
	} {
		phases, err := kanister.GetPhases(*tc.bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, IsNil)
		c.Assert(phases, HasLen, len(tc.outs))
		for i, p := range phases {
			out, err := p.Exec(ctx, *tc.bp, action, tp)
			c.Assert(err, IsNil, Commentf("Phase %s failed", p.Name()))
			c.Assert(out, DeepEquals, tc.outs[i])
		}
	}
}
