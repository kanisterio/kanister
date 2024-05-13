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

	"github.com/kanisterio/kanister/pkg/kube"
)

var (
	Container  ApplierList[corev1.Container]
	PodOptions ApplierList[kube.PodOptions]
)

// ApplierSet is a group of Appliers, typically returned by a constructor.
type ApplierSet struct {
	Container  Applier[corev1.Container]
	PodOptions Applier[kube.PodOptions]
}

// Register generically registers an Applier.
func Register[T Constraint](applier Applier[T]) {
	switch applier := any(applier).(type) {
	case Applier[corev1.Container]:
		Container.Register(applier)
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

	if set.PodOptions != nil {
		PodOptions.Register(set.PodOptions)
	}
}

// Constraint provides the set of types allowed for appliers and filterers.
type Constraint interface {
	kube.PodOptions | corev1.Container
}

// Applier is the interface which applies a manipulation to the PodOption to be
// used to run ephemeral pdos.
type Applier[T Constraint] interface {
	Apply(*T)
}

// ApplierFunc is a function which implements the Applier interface and can be
// used to generically manipulate the PodOptions.
type ApplierFunc[T Constraint] func(*T)

func (f ApplierFunc[T]) Apply(options *T) { f(options) }

// ApplierList is an array of registered Appliers which will be applied on
// a PodOption.
type ApplierList[T Constraint] []Applier[T]

// Apply calls the Applier::Apply method on all registered appliers.
func (l ApplierList[T]) Apply(options *T) {
	for _, applier := range l {
		applier.Apply(options)
	}
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
	return ApplierFunc[T](func(options *T) {
		if !filterer.Filter(options) {
			return
		}

		for _, applier := range appliers {
			applier.Apply(options)
		}
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
