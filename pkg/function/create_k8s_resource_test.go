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
	"fmt"
	"os"
	"time"

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

var _ = Suite(&CreateK8sResourceSuite{})

type CreateK8sResourceSuite struct {
	cli       kubernetes.Interface
	dynCli    dynamic.Interface
	namespace string
}

func (s *CreateK8sResourceSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	dynCli, err := kube.NewDynamicClient()
	c.Assert(err, IsNil)
	s.dynCli = dynCli

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterkubetasktest-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name
	os.Setenv("POD_NAMESPACE", cns.Name)
	os.Setenv("POD_SERVICE_ACCOUNT", "default")
}

func (s *CreateK8sResourceSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func createPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "create",
		Func: CreateK8sResourceFuncName,
		Args: map[string]interface{}{
			SpecsArg: fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: demo
  template:
    metadata:
      labels:
        app: demo
    spec:
      containers:
      - image: nginx:1.12
        imagePullPolicy: IfNotPresent
        name: web
        ports:
        - containerPort: 80
          name: http
          protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: test-deployment
  namespace: %s
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 80
  selector:
    app: demo
  type: ClusterIP`, namespace, namespace),
		},
	}
}

func newCreateResourceBlueprint(phases ...crv1alpha1.BlueprintPhase) crv1alpha1.Blueprint {
	return crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Phases: phases,
			},
		},
	}
}

func (s *CreateK8sResourceSuite) TestCreateK8sResource(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{}
	action := "test"
	type resourceRef struct {
		gvr       schema.GroupVersionResource
		name      string
		namespace string
	}
	for _, tc := range []struct {
		bp          crv1alpha1.Blueprint
		expResource []resourceRef
	}{
		{
			bp: newCreateResourceBlueprint(createPhase(s.namespace)),
			expResource: []resourceRef{
				{
					gvr:       schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
					name:      "test-deployment",
					namespace: s.namespace,
				},
				{
					gvr:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
					name:      "test-deployment",
					namespace: s.namespace,
				},
			},
		},
	} {
		phases, err := kanister.GetPhases(tc.bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			_, err := p.Exec(ctx, tc.bp, action, tp)
			c.Assert(err, IsNil, Commentf("Phase %s failed", p.Name()))
			for _, res := range tc.expResource {
				_, err = s.dynCli.Resource(res.gvr).Namespace(res.namespace).Get(context.TODO(), res.name, metav1.GetOptions{})
				c.Assert(err, IsNil)
			}
		}
	}
}
