// Copyright 2019 The Kanister Authors.
// Copyright 2016 The Rook Authors. All rights reserved.
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

package customresource

import (
	contextpkg "context"
	"fmt"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	_ "gopkg.in/check.v1" // importing go check to bypass the testing flags
)

const serverVersionV170 = "v1.7.0"

// CustomResource is for creating a Kubernetes TPR/CRD
type CustomResource struct {
	// Name of the custom resource
	Name string

	// Plural of the custom resource in plural
	Plural string

	// Group the custom resource belongs to
	Group string

	// Version which should be defined in a const above
	Version string

	// Scope of the CRD. Namespaced or cluster
	Scope apiextensionsv1.ResourceScope

	// Kind is the serialized interface of the resource.
	Kind string
}

// Context hold the clientsets used for creating and watching custom resources
type Context struct {
	Clientset             kubernetes.Interface
	APIExtensionClientset apiextensionsclient.Interface
	Interval              time.Duration
	Timeout               time.Duration
	Context               contextpkg.Context
}

// CreateCustomResources creates the given custom resources and waits for them to initialize
// The resource is of kind CRD if the Kubernetes server is 1.7.0 and above.
// The resource is of kind TPR if the Kubernetes server is below 1.7.0.
func CreateCustomResources(context Context, resources []CustomResource) error {
	// CRD is available on v1.7.0 and above. TPR became deprecated on v1.7.0
	serverVersion, err := context.Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("Error getting server version: %v", err)
	}
	kubeVersion := semver.MustParse(serverVersion.GitVersion)

	if kubeVersion.LessThan(semver.MustParse(serverVersionV170)) {
		return fmt.Errorf("Kubernetes versions less than 1.7.0 not supported")
	}
	var lastErr error
	for _, resource := range resources {
		err = createCRD(context, resource)
		if err != nil {
			lastErr = err
		}
	}

	for _, resource := range resources {
		if err := waitForCRDInit(context, resource); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func getCRDFromSpec(spec []byte) (*apiextensionsv1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := decodeSpecIntoObject(spec, crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func decodeSpecIntoObject(spec []byte, intoObj runtime.Object) error {
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode(spec, nil, intoObj); err != nil {
		return fmt.Errorf("Failed to decode spec into object: %s; spec %s\n", err.Error(), spec)
	}
	return nil
}

func createCRD(context Context, resource CustomResource) error {
	c, err := rawCRDFromFile(fmt.Sprintf("%s.yaml", resource.Name))
	if err != nil {
		return errors.Wrap(err, "Getting raw CRD from CRD manifests")
	}

	crd, err := getCRDFromSpec(c)
	if err != nil {
		return errors.Wrap(err, "Getting CRD object from CRD bytes")
	}

	_, err = context.APIExtensionClientset.ApiextensionsV1().CustomResourceDefinitions().Create(context.Context, crd, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Errorf("Failed to create %s CRD. %+v", resource.Name, err)
		}

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// if CRD already exists, get the resource version and create the CRD with that resource version
			c, err := context.APIExtensionClientset.ApiextensionsV1().CustomResourceDefinitions().Get(context.Context, crd.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			crd.ResourceVersion = c.ResourceVersion
			_, err = context.APIExtensionClientset.ApiextensionsV1().CustomResourceDefinitions().Update(context.Context, crd, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Failed to update existing CRD: %s\n", err.Error())
		}
	}
	return nil
}

func rawCRDFromFile(path string) ([]byte, error) {
	// yamls is the variable that has embedded custom resource manifest. More at `embed.go`
	return yamls.ReadFile(path)
}

func waitForCRDInit(context Context, resource CustomResource) error {
	crdName := fmt.Sprintf("%s.%s", resource.Plural, resource.Group)
	return wait.PollUntilContextTimeout(context.Context, context.Interval, context.Timeout, false, func(contextpkg.Context) (bool, error) {
		crd, err := context.APIExtensionClientset.ApiextensionsV1().CustomResourceDefinitions().Get(context.Context, crdName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1.Established:
				if cond.Status == apiextensionsv1.ConditionTrue {
					return true, nil
				}
			case apiextensionsv1.NamesAccepted:
				if cond.Status == apiextensionsv1.ConditionFalse {
					return false, fmt.Errorf("Name conflict: %v ", cond.Reason)
				}
			}
		}
		return false, nil
	})
}
