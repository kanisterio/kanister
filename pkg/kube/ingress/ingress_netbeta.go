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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type IngressNetBeta struct {
	kubeCli kubernetes.Interface
}

func NewIngressNetBeta(cli kubernetes.Interface) *IngressNetBeta {
	return &IngressNetBeta{
		kubeCli: cli,
	}
}

func (i *IngressNetBeta) List(ctx context.Context, ns string) (runtime.Object, error) {
	return i.kubeCli.NetworkingV1beta1().Ingresses(ns).List(ctx, metav1.ListOptions{})
}

func (i *IngressNetBeta) Get(ctx context.Context, ns, name string) (runtime.Object, error) {
	return i.kubeCli.NetworkingV1beta1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
}
