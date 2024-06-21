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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/client/applyconfiguration/cr/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeProfiles implements ProfileInterface
type FakeProfiles struct {
	Fake *FakeCrV1alpha1
	ns   string
}

var profilesResource = v1alpha1.SchemeGroupVersion.WithResource("profiles")

var profilesKind = v1alpha1.SchemeGroupVersion.WithKind("Profile")

// Get takes name of the profile, and returns the corresponding profile object, and an error if there is any.
func (c *FakeProfiles) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Profile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(profilesResource, c.ns, name), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}

// List takes label and field selectors, and returns the list of Profiles that match those selectors.
func (c *FakeProfiles) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ProfileList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(profilesResource, profilesKind, c.ns, opts), &v1alpha1.ProfileList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ProfileList{ListMeta: obj.(*v1alpha1.ProfileList).ListMeta}
	for _, item := range obj.(*v1alpha1.ProfileList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested profiles.
func (c *FakeProfiles) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(profilesResource, c.ns, opts))

}

// Create takes the representation of a profile and creates it.  Returns the server's representation of the profile, and an error, if there is any.
func (c *FakeProfiles) Create(ctx context.Context, profile *v1alpha1.Profile, opts v1.CreateOptions) (result *v1alpha1.Profile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(profilesResource, c.ns, profile), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}

// Update takes the representation of a profile and updates it. Returns the server's representation of the profile, and an error, if there is any.
func (c *FakeProfiles) Update(ctx context.Context, profile *v1alpha1.Profile, opts v1.UpdateOptions) (result *v1alpha1.Profile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(profilesResource, c.ns, profile), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}

// Delete takes name of the profile and deletes it. Returns an error if one occurs.
func (c *FakeProfiles) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(profilesResource, c.ns, name, opts), &v1alpha1.Profile{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeProfiles) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(profilesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ProfileList{})
	return err
}

// Patch applies the patch and returns the patched profile.
func (c *FakeProfiles) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Profile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(profilesResource, c.ns, name, pt, data, subresources...), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied profile.
func (c *FakeProfiles) Apply(ctx context.Context, profile *crv1alpha1.ProfileApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Profile, err error) {
	if profile == nil {
		return nil, fmt.Errorf("profile provided to Apply must not be nil")
	}
	data, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}
	name := profile.Name
	if name == nil {
		return nil, fmt.Errorf("profile.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(profilesResource, c.ns, *name, types.ApplyPatchType, data), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}
