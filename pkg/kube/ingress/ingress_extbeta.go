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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type IngressExtBeta struct {
	kubeCli kubernetes.Interface
}

func NewIngressExtBeta(cli kubernetes.Interface) *IngressExtBeta {
	return &IngressExtBeta{
		kubeCli: cli,
	}
}

func (i *IngressExtBeta) List(ctx context.Context, ns string) (runtime.Object, error) {
	return i.kubeCli.ExtensionsV1beta1().Ingresses(ns).List(ctx, metav1.ListOptions{})
}

func (i *IngressExtBeta) Get(ctx context.Context, ns, name string) (runtime.Object, error) {
	return i.kubeCli.ExtensionsV1beta1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
}

func (i *IngressExtBeta) IngressPath(ctx context.Context, ns, releaseName string) (string, error) {
	obj, err := i.Get(ctx, ns, fmt.Sprintf("%s-ingress", releaseName))
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
		if path.Backend.ServiceName == "gateway" {
			ingressPath = path.Path
			break
		}
	}
	if ingressPath == "" {
		return "", errors.Wrapf(err, "No path was set for K10's gateway service")
	}

	return ingressPath, nil
}
