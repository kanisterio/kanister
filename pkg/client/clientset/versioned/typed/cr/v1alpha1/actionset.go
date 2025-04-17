/*
Copyright 2024 The Kanister Authors.

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
// Code generated by client-gen. DO NOT EDIT.


// Package v1alpha1 provides the client implementation for interacting with
// resources in the Kanister custom resource API.
package v1alpha1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/client/applyconfiguration/cr/v1alpha1"
	scheme "github.com/kanisterio/kanister/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ActionSetsGetter has a method to return a ActionSetInterface.
// A group's client should implement this interface.
type ActionSetsGetter interface {
	ActionSets(namespace string) ActionSetInterface
}

// ActionSetInterface has methods to work with ActionSet resources.
type ActionSetInterface interface {
	Create(ctx context.Context, actionSet *v1alpha1.ActionSet, opts v1.CreateOptions) (*v1alpha1.ActionSet, error)
	Update(ctx context.Context, actionSet *v1alpha1.ActionSet, opts v1.UpdateOptions) (*v1alpha1.ActionSet, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ActionSet, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ActionSetList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ActionSet, err error)
	Apply(ctx context.Context, actionSet *crv1alpha1.ActionSetApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.ActionSet, err error)
	ActionSetExpansion
}

// actionSets implements ActionSetInterface
type actionSets struct {
	client rest.Interface
	ns     string
}

// newActionSets returns a ActionSets
func newActionSets(c *CrV1alpha1Client, namespace string) *actionSets {
	return &actionSets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the actionSet, and returns the corresponding actionSet object, and an error if there is any.
func (c *actionSets) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ActionSet, err error) {
	result = &v1alpha1.ActionSet{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("actionsets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ActionSets that match those selectors.
func (c *actionSets) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ActionSetList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ActionSetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("actionsets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested actionSets.
func (c *actionSets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("actionsets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a actionSet and creates it.  Returns the server's representation of the actionSet, and an error, if there is any.
func (c *actionSets) Create(ctx context.Context, actionSet *v1alpha1.ActionSet, opts v1.CreateOptions) (result *v1alpha1.ActionSet, err error) {
	result = &v1alpha1.ActionSet{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("actionsets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(actionSet).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a actionSet and updates it. Returns the server's representation of the actionSet, and an error, if there is any.
func (c *actionSets) Update(ctx context.Context, actionSet *v1alpha1.ActionSet, opts v1.UpdateOptions) (result *v1alpha1.ActionSet, err error) {
	result = &v1alpha1.ActionSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("actionsets").
		Name(actionSet.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(actionSet).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the actionSet and deletes it. Returns an error if one occurs.
func (c *actionSets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("actionsets").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *actionSets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("actionsets").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched actionSet.
func (c *actionSets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ActionSet, err error) {
	result = &v1alpha1.ActionSet{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("actionsets").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied actionSet.
func (c *actionSets) Apply(ctx context.Context, actionSet *crv1alpha1.ActionSetApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.ActionSet, err error) {
	if actionSet == nil {
		return nil, fmt.Errorf("actionSet provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(actionSet)
	if err != nil {
		return nil, err
	}
	name := actionSet.Name
	if name == nil {
		return nil, fmt.Errorf("actionSet.Name must be provided to Apply")
	}
	result = &v1alpha1.ActionSet{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("actionsets").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
