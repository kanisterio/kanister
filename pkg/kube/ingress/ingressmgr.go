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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
)

const (
	ingressRes = "ingresses"
)

type IngressMgr interface {
	List(ctx context.Context, ns string) (runtime.Object, error)
	Get(ctx context.Context, ns, name string) (runtime.Object, error)
}

func NewIngressMgr(ctx context.Context, kubeCli kubernetes.Interface) (IngressMgr, error) {
	exists, err := kube.IsResAvailableInGroupVersion(ctx, kubeCli.Discovery(), netv1.GroupName, netv1.SchemeGroupVersion.Version, ingressRes)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewIngressNetV1(kubeCli), nil
	}

	exists, err = kube.IsResAvailableInGroupVersion(ctx, kubeCli.Discovery(), extensionsv1beta1.GroupName, extensionsv1beta1.SchemeGroupVersion.Version, ingressRes)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewIngressExtBeta(kubeCli), nil
	}

	exists, err = kube.IsResAvailableInGroupVersion(ctx, kubeCli.Discovery(), netv1beta1.GroupName, netv1beta1.SchemeGroupVersion.Version, ingressRes)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewIngressNetBeta(kubeCli), nil
	}
	return nil, errors.New("Ingress resources are not available")
}
