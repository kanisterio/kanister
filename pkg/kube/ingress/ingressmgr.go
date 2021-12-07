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

	"github.com/pkg/errors"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	netv1 "k8s.io/api/networking/v1"
	netv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
)

const (
	ingressRes        = "ingresses"
	gatewaySvcName    = "gateway"
	ingressNameSuffix = "ingress"
	ingressNameFormat = "%s-%s"
)

// Manager is an abstraction over the behaviour of the ingress resources that
// depends on the APIVersion of the ingress resource
type Manager interface {
	// List can be used to list all the ingress resources from `ns` namespace
	List(ctx context.Context, ns string) (runtime.Object, error)
	// Get can be used to to get ingress resource with name `name` in `ns` namespace
	Get(ctx context.Context, ns, name string) (runtime.Object, error)
	// IngressPath can be used to get the backend path that is specified in the
	// ingress resource in `ns` namespace and name `releaseName-ingress`
	IngressPath(ctx context.Context, ns, releaseName string) (string, error)
	// Create accepts an ingress in as runtime.Object and creates on the cluster
	Create(ctx context.Context, ingress runtime.Object, opts metav1.CreateOptions) (runtime.Object, error)
}

// NewManager can be used to get the Manager based on the APIVersion of the ingress resources on the cluster
// so that, respecitve methods from that APIVersion can be called
func NewManager(ctx context.Context, kubeCli kubernetes.Interface) (Manager, error) {
	exists, err := kube.IsResAvailableInGroupVersion(ctx, kubeCli.Discovery(), netv1.GroupName, netv1.SchemeGroupVersion.Version, ingressRes)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewNetworkingV1(kubeCli), nil
	}

	exists, err = kube.IsResAvailableInGroupVersion(ctx, kubeCli.Discovery(), extensionsv1beta1.GroupName, extensionsv1beta1.SchemeGroupVersion.Version, ingressRes)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewExtensionsV1beta1(kubeCli), nil
	}

	exists, err = kube.IsResAvailableInGroupVersion(ctx, kubeCli.Discovery(), netv1beta1.GroupName, netv1beta1.SchemeGroupVersion.Version, ingressRes)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewNetworkingV1beta1(kubeCli), nil
	}
	return nil, errors.New("Ingress resources are not available")
}
