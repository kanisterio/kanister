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

package app

import (
	"fmt"
	"strings"
	"testing"

	. "gopkg.in/check.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/function"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type BlueprintSuite struct{}

var _ = Suite(&BlueprintSuite{})

func (bs *BlueprintSuite) TestUpdateImageTags(c *C) {
	for _, bp := range []*crv1alpha1.Blueprint{
		// BP with no phase with image arg
		&crv1alpha1.Blueprint{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-blueprint-",
			},
			Actions: map[string]*crv1alpha1.BlueprintAction{
				"test": &crv1alpha1.BlueprintAction{
					Kind: "Deployment",
					Phases: []crv1alpha1.BlueprintPhase{
						crv1alpha1.BlueprintPhase{
							Func: function.KubeExecFuncName,
							Name: "test-kube-exec",
							Args: map[string]interface{}{
								"namespace": "{{ .Deployment.Namespace }}",
								"pod":       "{{ index .Deployment.Pods 0 }}",
								"container": "test-container",
								"command":   []string{"echo", "hello"},
							},
						},
					},
				},
			},
		},

		// BP with multiple phases with image arg
		&crv1alpha1.Blueprint{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-blueprint-",
			},
			Actions: map[string]*crv1alpha1.BlueprintAction{
				"test": &crv1alpha1.BlueprintAction{
					Kind: "Deployment",
					Phases: []crv1alpha1.BlueprintPhase{
						crv1alpha1.BlueprintPhase{
							Func: function.KubeTaskFuncName,
							Name: "test-kube-task",
							Args: map[string]interface{}{
								"namespace": "{{ .Deployment.Namespace }}",
								"image":     "ghcr.io/image:v0.50.0",
								"command":   []string{"echo", "hello"},
							},
						},
						crv1alpha1.BlueprintPhase{
							Func: function.KubeTaskFuncName,
							Name: "test-kube-task2",
							Args: map[string]interface{}{
								"namespace": "{{ .Deployment.Namespace }}",
								"image":     "ghcr.io/image2:v0.50.0",
							},
						},
					},
				},
			},
		},

		// BP with multiple actions
		&crv1alpha1.Blueprint{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-blueprint-",
			},
			Actions: map[string]*crv1alpha1.BlueprintAction{
				"test": &crv1alpha1.BlueprintAction{
					Kind: "Deployment",
					Phases: []crv1alpha1.BlueprintPhase{
						crv1alpha1.BlueprintPhase{
							Func: function.KubeTaskFuncName,
							Name: "test-kube-task",
							Args: map[string]interface{}{
								"namespace": "{{ .Deployment.Namespace }}",
								"image":     "ghcr.io/image:v0.50.0",
								"command":   []string{"echo", "hello"},
							},
						},
						crv1alpha1.BlueprintPhase{
							Func: function.KubeTaskFuncName,
							Name: "test-kube-task2",
							Args: map[string]interface{}{
								"namespace": "{{ .Deployment.Namespace }}",
								"image":     "ghcr.io/image2:v0.50.0",
							},
						},
					},
				},
				"test2": &crv1alpha1.BlueprintAction{
					Phases: []crv1alpha1.BlueprintPhase{
						crv1alpha1.BlueprintPhase{
							Func: function.PrepareDataFuncName,
							Name: "test-prepare-data",
							Args: map[string]interface{}{
								"namespace": "{{ .Deployment.Namespace }}",
								"image":     "ghcr.io/image/tools:v0.40.0",
							},
						},
					},
				},
			},
		},
	} {
		updateImageTags(bp)
		validateImageTags(c, bp)
	}
}

func validateImageTags(c *C, bp *crv1alpha1.Blueprint) {
	podOverride := crv1alpha1.JSONMap{
		"containers": []map[string]interface{}{
			{
				"name":            "container",
				"imagePullPolicy": "Always",
			},
		},
	}
	for _, a := range bp.Actions {
		for _, phase := range a.Phases {
			image, ok := phase.Args["image"]
			if !ok {
				continue
			}
			// Verify if the tag is "latest"
			c.Log(fmt.Sprintf("phase:%s, image:%s", phase.Name, image.(string)))
			c.Assert(strings.Split(image.(string), ":")[1], Equals, "latest")
			c.Assert(phase.Args["podOverride"], DeepEquals, podOverride)
		}
	}
}
