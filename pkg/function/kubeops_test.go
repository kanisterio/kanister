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
	"time"

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	deploySpec = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
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
          protocol: TCP`

	serviceSpec = `apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 80
  selector:
    app: demo
  type: %s`

	fooCRSpec = `apiVersion: samplecontroller.k8s.io/v1alpha1
kind: Foo
metadata:
  name: %s
  namespace: %s
spec:
  deploymentName: example-foo
  replicas: 1`

	testServiceName = "test-service"
)

var _ = check.Suite(&KubeOpsSuite{})

type KubeOpsSuite struct {
	kubeCli   kubernetes.Interface
	crdCli    crdclient.Interface
	dynCli    dynamic.Interface
	namespace string
}

func (s *KubeOpsSuite) SetUpSuite(c *check.C) {
	cli, err := kube.NewClient()
	c.Assert(err, check.IsNil)
	s.kubeCli = cli

	dynCli, err := kube.NewDynamicClient()
	c.Assert(err, check.IsNil)
	s.dynCli = dynCli

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterkubeopstest-",
		},
	}
	cns, err := s.kubeCli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = cns.Name
	// Create CRD
	crdCli, err := kube.NewCRDClient()
	c.Assert(err, check.IsNil)
	s.crdCli = crdCli
	_, err = s.crdCli.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), getSampleCRD(), metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return
	}
	c.Assert(err, check.IsNil)
}

func (s *KubeOpsSuite) TearDownSuite(c *check.C) {
	if s.namespace != "" {
		_ = s.kubeCli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
	_ = s.crdCli.ApiextensionsV1().CustomResourceDefinitions().Delete(context.TODO(), getSampleCRD().GetName(), metav1.DeleteOptions{})
}

func createPhase(namespace string, spec string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "createDeploy",
		Func: KubeOpsFuncName,
		Args: map[string]interface{}{
			KubeOpsOperationArg: "create",
			KubeOpsNamespaceArg: namespace,
			KubeOpsSpecArg:      spec,
		},
	}
}

func patchPhase(gvr schema.GroupVersionResource, name, namespace, spec string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "patchDeploy",
		Func: KubeOpsFuncName,
		Args: map[string]interface{}{
			KubeOpsOperationArg: "patch",
			KubeOpsNamespaceArg: namespace,
			KubeOpsObjectReferenceArg: map[string]interface{}{
				"apiVersion": gvr.Version,
				"group":      gvr.Group,
				"resource":   gvr.Resource,
				"name":       name,
				"namespace":  namespace,
			},
			KubeOpsSpecArg: spec,
		},
	}
}

func deletePhase(gvr schema.GroupVersionResource, name, namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "deleteDeploy",
		Func: KubeOpsFuncName,
		Args: map[string]interface{}{
			KubeOpsOperationArg: "delete",
			KubeOpsNamespaceArg: namespace,
			KubeOpsObjectReferenceArg: map[string]interface{}{
				"apiVersion": gvr.Version,
				"group":      gvr.Group,
				"resource":   gvr.Resource,
				"name":       name,
				"namespace":  namespace,
			},
		},
	}
}

func createInSpecsNsPhase(spec string, config []string) crv1alpha1.BlueprintPhase {
	params := make([]interface{}, len(config))
	for i, v := range config {
		params[i] = v
	}

	return crv1alpha1.BlueprintPhase{
		Name: "create-in-def-ns",
		Func: KubeOpsFuncName,
		Args: map[string]interface{}{
			KubeOpsOperationArg: "create",
			KubeOpsSpecArg:      fmt.Sprintf(spec, params...),
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

func (s *KubeOpsSuite) TestKubeOps(c *check.C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{}
	action := "test"
	type resourceRef struct {
		gvr       schema.GroupVersionResource
		namespace string
	}
	for _, tc := range []struct {
		name        string
		spec        string
		expResource resourceRef
		config      []string
	}{
		{
			name:   fmt.Sprintf("%s-%s", testServiceName, rand.String(8)),
			spec:   serviceSpec,
			config: []string{"ClusterIP"},
			expResource: resourceRef{
				gvr:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
				namespace: s.namespace,
			},
		},
		{
			name:   fmt.Sprintf("%s-%s", "example-foo", rand.String(8)),
			spec:   fooCRSpec,
			config: []string{},
			expResource: resourceRef{
				gvr:       schema.GroupVersionResource{Group: "samplecontroller.k8s.io", Version: "v1alpha1", Resource: "foos"},
				namespace: s.namespace,
			},
		},
	} {
		bp := newCreateResourceBlueprint(createInSpecsNsPhase(tc.spec, append([]string{tc.name, s.namespace}, tc.config...)))
		phases, err := kanister.GetPhases(bp, action, kanister.DefaultVersion, tp)
		c.Assert(err, check.IsNil)
		for _, p := range phases {
			out, err := p.Exec(ctx, bp, action, tp)
			c.Assert(err, check.IsNil, check.Commentf("Phase %s failed", p.Name()))
			_, err = s.dynCli.Resource(tc.expResource.gvr).Namespace(tc.expResource.namespace).Get(context.TODO(), tc.name, metav1.GetOptions{})
			c.Assert(err, check.IsNil)
			expOut := map[string]interface{}{
				"apiVersion": tc.expResource.gvr.Version,
				"group":      tc.expResource.gvr.Group,
				"resource":   tc.expResource.gvr.Resource,
				"kind":       "",
				"name":       tc.name,
				"namespace":  tc.expResource.namespace,
			}
			c.Assert(out, check.DeepEquals, expOut)
			err = s.dynCli.Resource(tc.expResource.gvr).Namespace(s.namespace).Delete(ctx, tc.name, metav1.DeleteOptions{})
			c.Assert(err, check.IsNil)
		}
	}
}

func (s *KubeOpsSuite) TestKubeOpsCreatePatchDeleteWithCoreResource(c *check.C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{}
	action := "test"
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	serviceName := fmt.Sprintf("%s-%s", testServiceName, rand.String(8))
	spec := fmt.Sprintf(serviceSpec, serviceName, s.namespace, "ClusterIP")
	patchedSpec := fmt.Sprintf(serviceSpec, serviceName, s.namespace, "NodePort")

	bp := newCreateResourceBlueprint(
		createPhase(s.namespace, spec),
		patchPhase(gvr, serviceName, s.namespace, patchedSpec),
		deletePhase(gvr, serviceName, s.namespace),
	)

	phases, err := kanister.GetPhases(bp, action, kanister.DefaultVersion, tp)
	c.Assert(err, check.IsNil)
	for _, p := range phases {
		out, err := p.Exec(ctx, bp, action, tp)
		c.Assert(err, check.IsNil, check.Commentf("Phase %s failed", p.Name()))
		us, err := s.dynCli.Resource(gvr).Namespace(s.namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if p.Name() == "deleteDeploy" {
			c.Assert(err, check.NotNil)
			c.Assert(apierrors.IsNotFound(err), check.Equals, true)
		} else {
			c.Assert(err, check.IsNil)
		}

		// Validation for patch phase,
		if p.Name() == "patchDeploy" {
			// Check if the namespace is patched
			svc := &corev1.Service{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(us.Object, svc)
			c.Assert(err, check.IsNil)
			c.Assert(svc.Spec, check.NotNil)
			c.Assert(svc.Spec.Type, check.Equals, corev1.ServiceTypeNodePort)
		}

		expOut := map[string]interface{}{
			"apiVersion": gvr.Version,
			"group":      gvr.Group,
			"resource":   gvr.Resource,
			"kind":       "",
			"name":       serviceName,
			"namespace":  s.namespace,
		}
		c.Assert(out, check.DeepEquals, expOut)
	}
}

func (s *KubeOpsSuite) TestKubeOpsCreateWaitDelete(c *check.C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{
		Time: time.Now().String(),
	}
	action := "test"
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	deployName := "test-deployment"

	bp := newCreateResourceBlueprint(createPhase(s.namespace, deploySpec),
		waitDeployPhase(s.namespace, deployName),
		deletePhase(gvr, deployName, s.namespace))
	phases, err := kanister.GetPhases(bp, action, kanister.DefaultVersion, tp)
	c.Assert(err, check.IsNil)
	for _, p := range phases {
		out, err := p.Exec(ctx, bp, action, tp)
		c.Assert(err, check.IsNil, check.Commentf("Phase %s failed", p.Name()))

		_, err = s.dynCli.Resource(gvr).Namespace(s.namespace).Get(context.TODO(), deployName, metav1.GetOptions{})
		if p.Name() == "deleteDeploy" {
			c.Assert(err, check.NotNil)
			c.Assert(apierrors.IsNotFound(err), check.Equals, true)
		} else {
			c.Assert(err, check.IsNil)
		}

		if p.Name() == "waitDeployReady" {
			continue
		}
		expOut := map[string]interface{}{
			"apiVersion": gvr.Version,
			"group":      gvr.Group,
			"resource":   gvr.Resource,
			"kind":       "",
			"name":       deployName,
			"namespace":  s.namespace,
		}
		c.Assert(out, check.DeepEquals, expOut)
	}
}

func getSampleCRD() *extensionsv1.CustomResourceDefinition {
	return &extensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "foos.samplecontroller.k8s.io",
			Annotations: map[string]string{
				"api-approved.kubernetes.io": "unapproved",
			},
		},
		Spec: extensionsv1.CustomResourceDefinitionSpec{
			Group: "samplecontroller.k8s.io",
			Names: extensionsv1.CustomResourceDefinitionNames{
				Plural: "foos",
				Kind:   "Foo",
			},
			Scope: extensionsv1.ResourceScope("Namespaced"),
			Versions: []extensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &extensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &extensionsv1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]extensionsv1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
}
