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

// Code generated by client-gen. DO NOT EDIT.

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

// ProfilesGetter has a method to return a ProfileInterface.
// A group's client should implement this interface.
type ProfilesGetter interface {
	Profiles(namespace string) ProfileInterface
}

// ProfileInterface has methods to work with Profile resources.
type ProfileInterface interface {
	Create(ctx context.Context, profile *v1alpha1.Profile, opts v1.CreateOptions) (*v1alpha1.Profile, error)
	Update(ctx context.Context, profile *v1alpha1.Profile, opts v1.UpdateOptions) (*v1alpha1.Profile, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Profile, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ProfileList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Profile, err error)
	Apply(ctx context.Context, profile *crv1alpha1.ProfileApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Profile, err error)
	ProfileExpansion
}

// profiles implements ProfileInterface
type profiles struct {
	client rest.Interface
	ns     string
}

// newProfiles returns a Profiles
func newProfiles(c *CrV1alpha1Client, namespace string) *profiles {
	return &profiles{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the profile, and returns the corresponding profile object, and an error if there is any.
func (c *profiles) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Profile, err error) {
	result = &v1alpha1.Profile{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("profiles").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Profiles that match those selectors.
func (c *profiles) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ProfileList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ProfileList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("profiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested profiles.
func (c *profiles) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("profiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a profile and creates it.  Returns the server's representation of the profile, and an error, if there is any.
func (c *profiles) Create(ctx context.Context, profile *v1alpha1.Profile, opts v1.CreateOptions) (result *v1alpha1.Profile, err error) {
	result = &v1alpha1.Profile{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("profiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(profile).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a profile and updates it. Returns the server's representation of the profile, and an error, if there is any.
func (c *profiles) Update(ctx context.Context, profile *v1alpha1.Profile, opts v1.UpdateOptions) (result *v1alpha1.Profile, err error) {
	result = &v1alpha1.Profile{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("profiles").
		Name(profile.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(profile).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the profile and deletes it. Returns an error if one occurs.
func (c *profiles) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("profiles").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *profiles) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("profiles").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched profile.
func (c *profiles) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Profile, err error) {
	result = &v1alpha1.Profile{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("profiles").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied profile.
func (c *profiles) Apply(ctx context.Context, profile *crv1alpha1.ProfileApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Profile, err error) {
	if profile == nil {
		return nil, fmt.Errorf("profile provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}
	name := profile.Name
	if name == nil {
		return nil, fmt.Errorf("profile.Name must be provided to Apply")
	}
	result = &v1alpha1.Profile{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("profiles").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
