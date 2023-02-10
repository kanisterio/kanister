/*
Copyright The Kubernetes Authors.

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

// RepositoryServerLister helps list RepositoryServers.
// All objects returned here must be treated as read-only.
type RepositoryServerLister interface {
	// List lists all RepositoryServers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.RepositoryServer, err error)
	// RepositoryServers returns an object that can list and get RepositoryServers.
	RepositoryServers(namespace string) RepositoryServerNamespaceLister
	RepositoryServerListerExpansion
}

// repositoryServerLister implements the RepositoryServerLister interface.
type repositoryServerLister struct {
	indexer cache.Indexer
}

// NewRepositoryServerLister returns a new RepositoryServerLister.
func NewRepositoryServerLister(indexer cache.Indexer) RepositoryServerLister {
	return &repositoryServerLister{indexer: indexer}
}

// List lists all RepositoryServers in the indexer.
func (s *repositoryServerLister) List(selector labels.Selector) (ret []*v1alpha1.RepositoryServer, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RepositoryServer))
	})
	return ret, err
}

// RepositoryServers returns an object that can list and get RepositoryServers.
func (s *repositoryServerLister) RepositoryServers(namespace string) RepositoryServerNamespaceLister {
	return repositoryServerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// RepositoryServerNamespaceLister helps list and get RepositoryServers.
// All objects returned here must be treated as read-only.
type RepositoryServerNamespaceLister interface {
	// List lists all RepositoryServers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.RepositoryServer, err error)
	// Get retrieves the RepositoryServer from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.RepositoryServer, error)
	RepositoryServerNamespaceListerExpansion
}

// repositoryServerNamespaceLister implements the RepositoryServerNamespaceLister
// interface.
type repositoryServerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all RepositoryServers in the indexer for a given namespace.
func (s repositoryServerNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.RepositoryServer, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RepositoryServer))
	})
	return ret, err
}

// Get retrieves the RepositoryServer from the indexer for a given namespace and name.
func (s repositoryServerNamespaceLister) Get(name string) (*v1alpha1.RepositoryServer, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("repositoryserver"), name)
	}
	return obj.(*v1alpha1.RepositoryServer), nil
}
