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

var _ = Suite(&PrepareDataSuite{})

const (
	deployment  = "Deployment"
	statefulset = "StatefulSet"
)

type PrepareDataSuite struct {
	cli       kubernetes.Interface
	namespace string
}

func (s *PrepareDataSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "preparedatatest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *PrepareDataSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func newPrepareDataBlueprint(kind, pvc string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "test1",
						Func: PrepareDataFuncName,
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: fmt.Sprintf("{{ .%s.Namespace }}", kind),
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"touch",
								"/mnt/data1/foo.txt",
							},
							PrepareDataVolumes: map[string]string{pvc: "/mnt/data1"},
						},
					},
					{
						Name: "test2",
						Func: PrepareDataFuncName,
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: fmt.Sprintf("{{ .%s.Namespace }}", kind),
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"ls",
								"-l",
								"/mnt/data1/foo.txt",
							},
							PrepareDataVolumes: map[string]string{pvc: "/mnt/data1"},
						},
					},
					{
						Name: "test3",
						Func: PrepareDataFuncName,
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: fmt.Sprintf("{{ .%s.Namespace }}", kind),
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"touch",
								fmt.Sprintf("/mnt/prepare_data/%s/foo.txt", pvc),
							},
						},
					},
				},
			},
		},
	}
}

func (s *PrepareDataSuite) TestPrepareData(c *C) {
	pvc := testutil.NewTestPVC()
	createdPVC, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	ctx := context.Background()
	for _, kind := range []string{deployment, statefulset} {
		tp := param.TemplateParams{}
		switch kind {
		case deployment:
			tp.Deployment = &param.DeploymentParams{
				Namespace: s.namespace,
				PersistentVolumeClaims: map[string]map[string]string{
					"pod-0": {
						createdPVC.Name: "/mnt/data",
					},
				},
			}
		case statefulset:
			tp.StatefulSet = &param.StatefulSetParams{
				Namespace: s.namespace,
				PersistentVolumeClaims: map[string]map[string]string{
					"pod-0": {
						createdPVC.Name: "/mnt/data",
					},
				},
			}
		}
		action := "test"
		bp := newPrepareDataBlueprint(kind, createdPVC.Name)
		phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			_, err = p.Exec(ctx, *bp, action, tp)
			c.Assert(err, IsNil)
		}
	}
}
