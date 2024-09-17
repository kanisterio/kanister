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
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"

	. "gopkg.in/check.v1"
)

var _ = Suite(&KubeTaskParallelSuite{})

type KubeTaskParallelSuite struct {
	cli       kubernetes.Interface
	namespace string
}

func (s *KubeTaskParallelSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanister-kubetaskparalleltest-",
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

func (s *KubeTaskParallelSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func kubeTaskParallelPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testKubeTaskParallel",
		Func: KubeTaskParallelFuncName,
		Args: map[string]interface{}{
			KubeTaskParallelNamespaceArg:       namespace,
			KubeTaskParallelBackgroundImageArg: consts.LatestKanisterToolsImage,
			KubeTaskParallelBackgroundCommandArg: []string{
				"sh",
				"-c",
				"echo foo > /tmp/file",
			},
			KubeTaskParallelOutputImageArg: consts.LatestKanisterToolsImage,
			KubeTaskParallelOutputCommandArg: []string{
				"sh",
				"-c",
				"while [ ! -e /tmp/file  ]; do sleep 1; done; kando output value $(cat /tmp/file)",
			},
		},
	}
}

func (s *KubeTaskParallelSuite) TestKubeTaskParallel(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
		PodOverride: crv1alpha1.JSONMap{
			"containers": []map[string]interface{}{
				{
					"name":            "background",
					"imagePullPolicy": "Always",
				},
				{
					"name":            "output",
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
			bp: newTaskBlueprint(kubeTaskParallelPhase(s.namespace)),
			outs: []map[string]interface{}{
				{
					"value": "foo",
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

func kubeTaskParallelPhaseWithInit(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testKubeTaskParallel",
		Func: KubeTaskParallelFuncName,
		Args: map[string]interface{}{
			KubeTaskParallelNamespaceArg: namespace,
			KubeTaskParallelInitImageArg: consts.LatestKanisterToolsImage,
			KubeTaskParallelInitCommandArg: []string{
				"sh",
				"-c",
				"mkfifo /tmp/file",
			},
			KubeTaskParallelBackgroundImageArg: consts.LatestKanisterToolsImage,
			KubeTaskParallelBackgroundCommandArg: []string{
				"sh",
				"-c",
				"if [ ! -e /tmp/file  ]; then exit 1; fi; echo foo >> /tmp/file",
			},
			KubeTaskParallelOutputImageArg: consts.LatestKanisterToolsImage,
			KubeTaskParallelOutputCommandArg: []string{
				"sh",
				"-c",
				"if [ ! -e /tmp/file  ]; then exit 1; fi; kando output value $(cat /tmp/file)",
			},
		},
	}
}

func (s *KubeTaskParallelSuite) TestKubeTaskParallelWithInit(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
		PodOverride: crv1alpha1.JSONMap{
			"containers": []map[string]interface{}{
				{
					"name":            "background",
					"imagePullPolicy": "Always",
				},
				{
					"name":            "output",
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
			bp: newTaskBlueprint(kubeTaskParallelPhaseWithInit(s.namespace)),
			outs: []map[string]interface{}{
				{
					"value": "foo",
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
