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

// BlueprintsGetter has a method to return a BlueprintInterface.
// A group's client should implement this interface.
type BlueprintsGetter interface {
	Blueprints(namespace string) BlueprintInterface
}

// BlueprintInterface has methods to work with Blueprint resources.
type BlueprintInterface interface {
	Create(*Blueprint) (*Blueprint, error)
	Get(name string, options v1.GetOptions) (*Blueprint, error)
	Update(*Blueprint) (*Blueprint, error)
	Delete(name string, options *v1.DeleteOptions) error
	List(opts v1.ListOptions) (*BlueprintList, error)
}

// blueprints implements BlueprintInterface
type blueprints struct {
	client         rest.Interface
	ns             string
	parameterCodec runtime.ParameterCodec
}

// newBlueprints returns a Blueprints
func newBlueprints(c *CRV1alpha1Client, namespace string, parameterCodec runtime.ParameterCodec) *blueprints {
	return &blueprints{
		client:         c.RESTClient(),
		ns:             namespace,
		parameterCodec: parameterCodec,
	}
}

// Create takes the representation of a blueprint and creates it.  Returns the server's representation of the blueprint, and an error, if there is any.
func (c *blueprints) Create(blueprint *Blueprint) (result *Blueprint, err error) {
	result = &Blueprint{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource(BlueprintResourceNamePlural).
		Body(blueprint).
		Do().
		Into(result)
	return
}

// Get takes name of the blueprint, and returns the corresponding blueprint object, and an error if there is any.
func (c *blueprints) Get(name string, options v1.GetOptions) (result *Blueprint, err error) {
	result = &Blueprint{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(BlueprintResourceNamePlural).
		Name(name).
		VersionedParams(&options, c.parameterCodec).
		Do().
		Into(result)
	return
}

// Update takes the representation of a blueprint and updates it. Returns the server's representation of the blueprint, and an error, if there is any.
func (c *blueprints) Update(blueprint *Blueprint) (result *Blueprint, err error) {
	result = &Blueprint{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource(BlueprintResourceNamePlural).
		Name(blueprint.Name).
		Body(blueprint).
		Do().
		Into(result)
	return
}

// Delete takes name of the blueprint and deletes it. Returns an error if one occurs.
func (c *blueprints) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource(BlueprintResourceNamePlural).
		Name(name).
		Body(options).
		Do().
		Error()
}

// List takes label and field selectors, and returns the list of Blueprints that match those selectors.
func (c *blueprints) List(opts v1.ListOptions) (result *BlueprintList, err error) {
	result = &BlueprintList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(BlueprintResourceNamePlural).
		VersionedParams(&opts, c.parameterCodec).
		Do().
		Into(result)
	return
}
