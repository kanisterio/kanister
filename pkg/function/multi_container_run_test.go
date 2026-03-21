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

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

var _ = check.Suite(&MultiContainerRunSuite{})

type MultiContainerRunSuite struct {
	cli       kubernetes.Interface
	namespace string
}

func (s *MultiContainerRunSuite) SetUpSuite(c *check.C) {
	cli, err := kube.NewClient()
	c.Assert(err, check.IsNil)
	s.cli = cli

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanister-multicontainerruntest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = cns.Name
	err = os.Setenv("POD_NAMESPACE", cns.Name)
	c.Assert(err, check.IsNil)
	err = os.Setenv("POD_SERVICE_ACCOUNT", "default")
	c.Assert(err, check.IsNil)
}

func (s *MultiContainerRunSuite) TearDownSuite(c *check.C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func multiContainerRunPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testMultiContainerRun",
		Func: MultiContainerRunFuncName,
		Args: map[string]interface{}{
			MultiContainerRunNamespaceArg:       namespace,
			MultiContainerRunBackgroundImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunBackgroundCommandArg: []string{
				"sh",
				"-c",
				"echo foo > /tmp/file",
			},
			MultiContainerRunOutputImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunOutputCommandArg: []string{
				"sh",
				"-c",
				"while [ ! -e /tmp/file  ]; do sleep 1; done; kando output value $(cat /tmp/file)",
			},
		},
	}
}

func (s *MultiContainerRunSuite) TestMultiContainerRun(c *check.C) {
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
			bp: newTaskBlueprint(multiContainerRunPhase(s.namespace)),
			outs: []map[string]interface{}{
				{
					"value": "foo",
				},
			},
		},
	} {
		phases, err := kanister.GetPhases(*tc.bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, check.IsNil)
		c.Assert(phases, check.HasLen, len(tc.outs))
		for i, p := range phases {
			out, err := p.Exec(ctx, *tc.bp, action, tp)
			c.Assert(err, check.IsNil, check.Commentf("Phase %s failed", p.Name()))
			c.Assert(out, check.DeepEquals, tc.outs[i])
		}
	}
}

func multiContainerRunPhaseWithInit(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testMultiContainerRun",
		Func: MultiContainerRunFuncName,
		Args: map[string]interface{}{
			MultiContainerRunNamespaceArg: namespace,
			MultiContainerRunInitImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunInitCommandArg: []string{
				"sh",
				"-c",
				"mkfifo /tmp/file",
			},
			MultiContainerRunBackgroundImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunBackgroundCommandArg: []string{
				"sh",
				"-c",
				"if [ ! -e /tmp/file  ]; then exit 1; fi; echo foo >> /tmp/file",
			},
			MultiContainerRunOutputImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunOutputCommandArg: []string{
				"sh",
				"-c",
				"if [ ! -e /tmp/file  ]; then exit 1; fi; kando output value $(cat /tmp/file)",
			},
		},
	}
}

func (s *MultiContainerRunSuite) TestMultiContainerRunWithInit(c *check.C) {
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
			bp: newTaskBlueprint(multiContainerRunPhaseWithInit(s.namespace)),
			outs: []map[string]interface{}{
				{
					"value": "foo",
				},
			},
		},
	} {
		phases, err := kanister.GetPhases(*tc.bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, check.IsNil)
		c.Assert(phases, check.HasLen, len(tc.outs))
		for i, p := range phases {
			out, err := p.Exec(ctx, *tc.bp, action, tp)
			c.Assert(err, check.IsNil, check.Commentf("Phase %s failed", p.Name()))
			c.Assert(out, check.DeepEquals, tc.outs[i])
		}
	}
}

func multiContainerRunPhaseWithoutNamespace() crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testMultiContainerRun",
		Func: MultiContainerRunFuncName,
		Args: map[string]interface{}{
			MultiContainerRunBackgroundImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunBackgroundCommandArg: []string{
				"sh",
				"-c",
				"echo foo > /tmp/file",
			},
			MultiContainerRunOutputImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunOutputCommandArg: []string{
				"sh",
				"-c",
				"while [ ! -e /tmp/file  ]; do sleep 1; done; kando output value $(cat /tmp/file)",
			},
		},
	}
}

func (s *MultiContainerRunSuite) TestMultiContainerRunWithoutNamespace(c *check.C) {
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
			bp: newTaskBlueprint(multiContainerRunPhaseWithoutNamespace()),
			outs: []map[string]interface{}{
				{
					"value": "foo",
				},
			},
		},
	} {
		phases, err := kanister.GetPhases(*tc.bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, check.IsNil)
		c.Assert(phases, check.HasLen, len(tc.outs))
		for i, p := range phases {
			out, err := p.Exec(ctx, *tc.bp, action, tp)
			c.Assert(err, check.IsNil, check.Commentf("Phase %s failed", p.Name()))
			c.Assert(out, check.DeepEquals, tc.outs[i])
		}
	}
}

func (s *MultiContainerRunSuite) TestPrepareActionSetPodSpecOverride(c *check.C) {
	testCases := []struct {
		name     string
		input    crv1alpha1.JSONMap
		expected crv1alpha1.JSONMap
	}{
		{
			name: "explicit_background_and_output_pass_through_unchanged",
			input: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name":            "background",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
					map[string]interface{}{
						"name":            "output",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
				},
			},
			expected: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name":            "background",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
					map[string]interface{}{
						"name":            "output",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "unknown_container_name_pass_through_unchanged",
			input: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name":            "other_container",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
				},
			},
			expected: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name":            "other_container",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "container_alias_expands_to_init_background_output",
			input: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name":            "container",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
					map[string]interface{}{
						"name":            "other_container",
						"imagePullPolicy": "Never",
						"resources":       map[string]interface{}{},
					},
				},
			},
			expected: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name":            "other_container",
						"imagePullPolicy": "Never",
						"resources":       map[string]interface{}{},
					},
					map[string]interface{}{
						"name":            "background",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
					map[string]interface{}{
						"name":            "output",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
				},
				"initContainers": []interface{}{
					map[string]interface{}{
						"name":            "init",
						"imagePullPolicy": "Always",
						"resources":       map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "container_alias_with_resources_expands_to_all",
			input: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name": "container",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
				},
			},
			expected: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name": "background",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
					map[string]interface{}{
						"name": "output",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
				},
				"initContainers": []interface{}{
					map[string]interface{}{
						"name": "init",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
				},
			},
		},
		{
			name: "container_alias_with_explicit_background_keeps_explicit",
			input: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name": "container",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
					map[string]interface{}{
						"name": "background",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "256Mi",
							},
						},
					},
				},
			},
			expected: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name": "background",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "256Mi",
							},
						},
					},
					map[string]interface{}{
						"name": "output",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
				},
				"initContainers": []interface{}{
					map[string]interface{}{
						"name": "init",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
				},
			},
		},
		{
			name: "explicit_init_override_preserved_not_duplicated",
			input: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name": "container",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
				},
				"initContainers": []interface{}{
					map[string]interface{}{
						"name": "init",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "512Mi",
							},
						},
					},
				},
			},
			expected: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name": "background",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
					map[string]interface{}{
						"name": "output",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
				},
				"initContainers": []interface{}{
					map[string]interface{}{
						"name": "init",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "512Mi",
							},
						},
					},
				},
			},
		},
		{
			name:     "nil_input_returns_nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty_map_returns_empty",
			input:    crv1alpha1.JSONMap{},
			expected: crv1alpha1.JSONMap{},
		},
		{
			name: "no_containers_key_pass_through",
			input: crv1alpha1.JSONMap{
				"serviceAccountName": "custom-sa",
			},
			expected: crv1alpha1.JSONMap{
				"serviceAccountName": "custom-sa",
			},
		},
		{
			name: "container_alias_with_explicit_output_only",
			input: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name": "container",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
					map[string]interface{}{
						"name": "output",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "512Mi",
							},
						},
					},
				},
			},
			expected: crv1alpha1.JSONMap{
				"containers": []interface{}{
					map[string]interface{}{
						"name": "background",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
					map[string]interface{}{
						"name": "output",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "512Mi",
							},
						},
					},
				},
				"initContainers": []interface{}{
					map[string]interface{}{
						"name": "init",
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"memory": "128Mi",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		c.Log("Running test case:", tc.name)

		result, err := prepareActionSetPodSpecOverride(tc.input)

		c.Assert(err, check.IsNil, check.Commentf("Test case %q: unexpected error", tc.name))
		c.Assert(result, check.DeepEquals, tc.expected, check.Commentf("Test case %q: result mismatch", tc.name))
	}
}

func (s *MultiContainerRunSuite) TestInitContainerGenericOverride(c *check.C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Define a phase with an init container that checks for the user ID
	phase := crv1alpha1.BlueprintPhase{
		Name: "testInitContainerGenericOverride",
		Func: MultiContainerRunFuncName,
		Args: map[string]interface{}{
			MultiContainerRunNamespaceArg: s.namespace,
			MultiContainerRunInitImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunInitCommandArg: []string{
				"sh",
				"-c",
				"if [ \"$(id -u)\" != \"1000\" ]; then echo 'User ID is not 1000'; exit 1; fi",
			},
			MultiContainerRunBackgroundImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunBackgroundCommandArg: []string{
				"sh",
				"-c",
				"while [ ! -f /tmp/done ]; do sleep 1; done",
			},
			MultiContainerRunOutputImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunOutputCommandArg: []string{
				"sh",
				"-c",
				"echo 'success' && touch /tmp/done",
			},
		},
	}

	// Define a generic override that sets the securityContext on "container"
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
		PodOverride: crv1alpha1.JSONMap{
			"containers": []map[string]interface{}{
				{
					"name": "container",
					"securityContext": map[string]interface{}{
						"runAsUser": 1000,
					},
				},
			},
		},
	}

	bp := newTaskBlueprint(phase)
	action := "test"

	phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, tp)
	c.Assert(err, check.IsNil)
	c.Assert(phases, check.HasLen, 1)

	for _, p := range phases {
		_, err := p.Exec(ctx, *bp, action, tp)
		c.Assert(err, check.IsNil, check.Commentf("Phase %s failed", p.Name()))
	}
}

func (s *MultiContainerRunSuite) TestInitContainerGenericOverrideWithExtraInit(c *check.C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Define a phase with an init container that checks for the user ID
	phase := crv1alpha1.BlueprintPhase{
		Name: "testInitContainerGenericOverrideWithExtraInit",
		Func: MultiContainerRunFuncName,
		Args: map[string]interface{}{
			MultiContainerRunNamespaceArg: s.namespace,
			MultiContainerRunInitImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunInitCommandArg: []string{
				"sh",
				"-c",
				"if [ \"$(id -u)\" != \"1000\" ]; then echo 'User ID is not 1000'; exit 1; fi",
			},
			MultiContainerRunBackgroundImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunBackgroundCommandArg: []string{
				"sh",
				"-c",
				"while [ ! -f /tmp/done ]; do sleep 0.1; done",
			},
			MultiContainerRunOutputImageArg: consts.LatestKanisterToolsImage,
			MultiContainerRunOutputCommandArg: []string{
				"sh",
				"-c",
				"echo 'success' && touch /tmp/done",
			},
		},
	}

	// Define a generic override that sets the securityContext on "container"
	// AND adds an extra init container
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
		PodOverride: crv1alpha1.JSONMap{
			"containers": []map[string]interface{}{
				{
					"name": "container",
					"securityContext": map[string]interface{}{
						"runAsUser": 1000,
					},
				},
			},
			"initContainers": []map[string]interface{}{
				{
					"name":    "extra-init",
					"image":   consts.LatestKanisterToolsImage,
					"command": []string{"sh", "-c", "echo extra"},
				},
			},
		},
	}

	bp := newTaskBlueprint(phase)
	action := "test"

	phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, tp)
	c.Assert(err, check.IsNil)
	c.Assert(phases, check.HasLen, 1)

	for _, p := range phases {
		_, err := p.Exec(ctx, *bp, action, tp)
		c.Assert(err, check.IsNil, check.Commentf("Phase %s failed", p.Name()))
	}
}
