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

package ingress

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// ExtensionsV1beta1 implements ingress.Manager interface
type ExtensionsV1beta1 struct {
	kubeCli kubernetes.Interface
}

func NewExtensionsV1beta1(cli kubernetes.Interface) *ExtensionsV1beta1 {
	return &ExtensionsV1beta1{
		kubeCli: cli,
	}
}

// List can be used to list all the ingress resources from `ns` namespace
func (i *ExtensionsV1beta1) List(ctx context.Context, ns string) (runtime.Object, error) {
	return i.kubeCli.ExtensionsV1beta1().Ingresses(ns).List(ctx, metav1.ListOptions{})
}

// Get can be used to to get ingress resource with name `name` in `ns` namespace
func (i *ExtensionsV1beta1) Get(ctx context.Context, ns, name string) (runtime.Object, error) {
	return i.kubeCli.ExtensionsV1beta1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
}

// Create can be used to create an ingress resource in extensions v1beta1 apiVersion
func (i *ExtensionsV1beta1) Create(ctx context.Context, ingress *unstructured.Unstructured, opts metav1.CreateOptions) (runtime.Object, error) {
	var ing extensionsv1beta1.Ingress
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(ingress.UnstructuredContent(), &ing)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed converting runtime.Object to extensions/v1beta1 ingress")
	}
	return i.kubeCli.ExtensionsV1beta1().Ingresses(ing.Namespace).Create(ctx, &ing, opts)
}

// Update can be used to update an ingress resource in extensions v1beta1 apiVersion
func (i *ExtensionsV1beta1) Update(ctx context.Context, ingress *unstructured.Unstructured, opts metav1.UpdateOptions) (runtime.Object, error) {
	var ing extensionsv1beta1.Ingress
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(ingress.UnstructuredContent(), &ing)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed converting runtime.Object to extensions/v1beta1 ingress")
	}
	return i.kubeCli.ExtensionsV1beta1().Ingresses(ing.Namespace).Update(ctx, &ing, opts)
}

// IngressPath can be used to get the backend path that is specified in the
// ingress resource in `ns` namespace and name `releaseName-ingress`
func (i *ExtensionsV1beta1) IngressPath(ctx context.Context, ns, releaseName string) (string, error) {
	obj, err := i.Get(ctx, ns, fmt.Sprintf(ingressNameFormat, releaseName, ingressNameSuffix))
	if apierrors.IsNotFound(err) {
		// Try the release name if the ingress does not exist.
		// This is possible if the user setup OIDC using the localhost IP
		// and has port forwarding turned on to access K10.
		return releaseName, nil
	}
	if err != nil {
		return "", err
	}

	ingress := obj.(*extensionsv1beta1.Ingress)
	if len(ingress.Spec.Rules) == 0 {
		return "", errors.Wrapf(err, "No ingress rules were found")
	}
	ingressHTTPRule := ingress.Spec.Rules[0].IngressRuleValue.HTTP
	if ingressHTTPRule == nil {
		return "", errors.Wrapf(err, "A HTTP ingress rule value is missing")
	}
	ingressPaths := ingressHTTPRule.Paths
	if len(ingressPaths) == 0 {
		return "", errors.Wrapf(err, "Failed to find HTTP paths in the ingress")
	}
	ingressPath := ""
	for _, path := range ingressPaths {
		if path.Backend.ServiceName == gatewaySvcName {
			ingressPath = path.Path
			break
		}
	}
	if ingressPath == "" {
		return "", errors.Wrapf(err, "No path was set for K10's gateway service")
	}

	return ingressPath, nil
}
