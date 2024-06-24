/*
Copyright 2023 The Kanister Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// BlueprintLister helps list Blueprints.
// All objects returned here must be treated as read-only.
type BlueprintLister interface {
	// List lists all Blueprints in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Blueprint, err error)
	// Blueprints returns an object that can list and get Blueprints.
	Blueprints(namespace string) BlueprintNamespaceLister
	BlueprintListerExpansion
}

// blueprintLister implements the BlueprintLister interface.
type blueprintLister struct {
	indexer cache.Indexer
}

// NewBlueprintLister returns a new BlueprintLister.
func NewBlueprintLister(indexer cache.Indexer) BlueprintLister {
	return &blueprintLister{indexer: indexer}
}

// List lists all Blueprints in the indexer.
func (s *blueprintLister) List(selector labels.Selector) (ret []*v1alpha1.Blueprint, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Blueprint))
	})
	return ret, err
}

// Blueprints returns an object that can list and get Blueprints.
func (s *blueprintLister) Blueprints(namespace string) BlueprintNamespaceLister {
	return blueprintNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// BlueprintNamespaceLister helps list and get Blueprints.
// All objects returned here must be treated as read-only.
type BlueprintNamespaceLister interface {
	// List lists all Blueprints in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Blueprint, err error)
	// Get retrieves the Blueprint from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Blueprint, error)
	BlueprintNamespaceListerExpansion
}

// blueprintNamespaceLister implements the BlueprintNamespaceLister
// interface.
type blueprintNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Blueprints in the indexer for a given namespace.
func (s blueprintNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Blueprint, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Blueprint))
	})
	return ret, err
}

// Get retrieves the Blueprint from the indexer for a given namespace and name.
func (s blueprintNamespaceLister) Get(name string) (*v1alpha1.Blueprint, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("blueprint"), name)
	}
	return obj.(*v1alpha1.Blueprint), nil
}
