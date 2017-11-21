/*
Copyright 2017 The Kubernetes Authors.

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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// ActionSetsGetter has a method to return a ActionSetInterface.
// A group's client should implement this interface.
type ActionSetsGetter interface {
	ActionSets(namespace string) ActionSetInterface
}

// ActionSetInterface has methods to work with ActionSet resources.
type ActionSetInterface interface {
	Create(*ActionSet) (*ActionSet, error)
	Get(name string, options v1.GetOptions) (*ActionSet, error)
	Update(*ActionSet) (*ActionSet, error)
	Delete(name string, options *v1.DeleteOptions) error
	List(opts v1.ListOptions) (*ActionSetList, error)
}

// actionsets implements ActionSetInterface
type actionsets struct {
	client         rest.Interface
	ns             string
	parameterCodec runtime.ParameterCodec
}

// newActionSets returns a ActionSets
func newActionSets(c *CRV1alpha1Client, namespace string, parameterCodec runtime.ParameterCodec) *actionsets {
	return &actionsets{
		client:         c.RESTClient(),
		ns:             namespace,
		parameterCodec: parameterCodec,
	}
}

// Create takes the representation of a actionset and creates it.  Returns the server's representation of the actionset, and an error, if there is any.
func (c *actionsets) Create(actionset *ActionSet) (result *ActionSet, err error) {
	result = &ActionSet{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource(ActionSetResourceNamePlural).
		Body(actionset).
		Do().
		Into(result)
	return
}

// Get takes name of the actionset, and returns the corresponding actionset object, and an error if there is any.
func (c *actionsets) Get(name string, options v1.GetOptions) (result *ActionSet, err error) {
	result = &ActionSet{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(ActionSetResourceNamePlural).
		Name(name).
		VersionedParams(&options, c.parameterCodec).
		Do().
		Into(result)
	return
}

// Update takes the representation of a actionset and updates it. Returns the server's representation of the actionset, and an error, if there is any.
func (c *actionsets) Update(actionset *ActionSet) (result *ActionSet, err error) {
	result = &ActionSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource(ActionSetResourceNamePlural).
		Name(actionset.Name).
		Body(actionset).
		Do().
		Into(result)
	return
}

// Delete takes name of the actionset and deletes it. Returns an error if one occurs.
func (c *actionsets) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource(ActionSetResourceNamePlural).
		Name(name).
		Body(options).
		Do().
		Error()
}

// List takes label and field selectors, and returns the list of ActionSets that match those selectors.
func (c *actionsets) List(opts v1.ListOptions) (result *ActionSetList, err error) {
	result = &ActionSetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(ActionSetResourceNamePlural).
		VersionedParams(&opts, c.parameterCodec).
		Do().
		Into(result)
	return
}
