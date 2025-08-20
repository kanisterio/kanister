// Copyright 2024 The Kanister Authors.
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

package ephemeral

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/kanisterio/kanister/pkg/kube"
)

var (
	Container             ApplierList[corev1.Container]
	PersistentVolumeClaim ApplierList[corev1.PersistentVolumeClaim]
	Pod                   ApplierList[corev1.Pod]
	Service               ApplierList[corev1.Service]
	NetworkPolicy         ApplierList[networkingv1.NetworkPolicy]
	PodOptions            ApplierList[kube.PodOptions]
)

// ApplierSet is a group of Appliers, typically returned by a constructor.
type ApplierSet struct {
	Container             Applier[corev1.Container]
	PersistentVolumeClaim Applier[corev1.PersistentVolumeClaim]
	Pod                   Applier[corev1.Pod]
	Service               Applier[corev1.Service]
	NetworkPolicy         Applier[networkingv1.NetworkPolicy]
	PodOptions            Applier[kube.PodOptions]
}

// Register generically registers an Applier.
func Register[T Constraint](applier Applier[T]) {
	switch applier := any(applier).(type) {
	case Applier[corev1.Container]:
		Container.Register(applier)
	case Applier[corev1.PersistentVolumeClaim]:
		PersistentVolumeClaim.Register(applier)
	case Applier[corev1.Pod]:
		Pod.Register(applier)
	case Applier[corev1.Service]:
		Service.Register(applier)
	case Applier[networkingv1.NetworkPolicy]:
		NetworkPolicy.Register(applier)
	case Applier[kube.PodOptions]:
		PodOptions.Register(applier)
	default:
		panic("Unknown applier type")
	}
}

// RegisterSet registers each of the Appliers contained in the set.
func RegisterSet(set ApplierSet) {
	if set.Container != nil {
		Container.Register(set.Container)
	}

	if set.PersistentVolumeClaim != nil {
		PersistentVolumeClaim.Register(set.PersistentVolumeClaim)
	}

	if set.Pod != nil {
		Pod.Register(set.Pod)
	}

	if set.Service != nil {
		Service.Register(set.Service)
	}

	if set.NetworkPolicy != nil {
		NetworkPolicy.Register(set.NetworkPolicy)
	}

	if set.PodOptions != nil {
		PodOptions.Register(set.PodOptions)
	}
}

// Constraint provides the set of types allowed for appliers and filterers.
type Constraint interface {
	kube.PodOptions | corev1.Container | corev1.PersistentVolumeClaim | corev1.Pod | corev1.Service | networkingv1.NetworkPolicy
}

// Applier is the interface which applies a manipulation to the PodOption to be
// used to run ephemeral pdos.
type Applier[T Constraint] interface {
	Apply(*T) error
}

// ApplierFunc is a function which implements the Applier interface and can be
// used to generically manipulate the PodOptions.
type ApplierFunc[T Constraint] func(*T) error

func (f ApplierFunc[T]) Apply(options *T) error { return f(options) }

// ApplierList is an array of registered Appliers which will be applied on
// a PodOption.
type ApplierList[T Constraint] []Applier[T]

// Apply calls the Applier::Apply method on all registered appliers.
func (l ApplierList[T]) Apply(options *T) error {
	for _, applier := range l {
		if err := applier.Apply(options); err != nil {
			return err
		}
	}

	return nil
}

// Register adds the applier to the list of Appliers to be used when
// manipulating the PodOptions.
func (l *ApplierList[T]) Register(applier Applier[T]) {
	*l = append(*l, applier)
}

// Filterer is the interface which filters the use of registered appliers to
// only those PodOptions that match the filter criteria.
type Filterer[T Constraint] interface {
	Filter(*T) bool
}

// FiltererFunc is a function which implements the Filterer interface and can be
// used to generically filter PodOptions to manipulate using the ApplierList.
type FiltererFunc[T Constraint] func(*T) bool

func (f FiltererFunc[T]) Filter(options *T) bool {
	return f(options)
}

// Filter applies the Appliers if the Filterer criterion is met.
func Filter[T Constraint](filterer Filterer[T], appliers ...Applier[T]) Applier[T] {
	return ApplierFunc[T](func(options *T) error {
		if !filterer.Filter(options) {
			return nil
		}

		for _, applier := range appliers {
			if err := applier.Apply(options); err != nil {
				return err
			}
		}

		return nil
	})
}

// PodOptionsNameFilter is a Filterer that filters based on the PodOptions.Name
// which is the Pod name.
type PodOptionsNameFilter string

func (n PodOptionsNameFilter) Filter(options *kube.PodOptions) bool {
	return string(n) == options.Name
}

// ContainerNameFilter is a Filterer that filters based on the Container.Name.
type ContainerNameFilter string

func (n ContainerNameFilter) Filter(container *corev1.Container) bool {
	return string(n) == container.Name
}

// PVCNameFilter is a Filterer that filters based on the PersistentVolumeClaim.Name.
type PVCNameFilter string

func (n PVCNameFilter) Filter(pvc *corev1.PersistentVolumeClaim) bool {
	return string(n) == pvc.Name
}

// PodNameFilter is a Filterer that filters based on the Pod.Name.
type PodNameFilter string

func (n PodNameFilter) Filter(pod *corev1.Pod) bool {
	return string(n) == pod.Name
}

// ServiceNameFilter is a Filterer that filters based on the Service.Name.
type ServiceNameFilter string

func (n ServiceNameFilter) Filter(svc *corev1.Service) bool {
	return string(n) == svc.Name
}

// NetworkPolicyNameFilter is a Filterer that filters based on the NetworkPolicy.Name.
type NetworkPolicyNameFilter string

func (n NetworkPolicyNameFilter) Filter(np *networkingv1.NetworkPolicy) bool {
	return string(n) == np.Name
}
