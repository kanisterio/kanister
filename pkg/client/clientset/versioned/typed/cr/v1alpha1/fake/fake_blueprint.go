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

	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeBlueprints implements BlueprintInterface
type FakeBlueprints struct {
	Fake *FakeCrV1alpha1
	ns   string
}

var blueprintsResource = schema.GroupVersionResource{Group: "cr.kanister.io", Version: "v1alpha1", Resource: "blueprints"}

var blueprintsKind = schema.GroupVersionKind{Group: "cr.kanister.io", Version: "v1alpha1", Kind: "Blueprint"}

// Get takes name of the blueprint, and returns the corresponding blueprint object, and an error if there is any.
func (c *FakeBlueprints) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Blueprint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(blueprintsResource, c.ns, name), &v1alpha1.Blueprint{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Blueprint), err
}

// List takes label and field selectors, and returns the list of Blueprints that match those selectors.
func (c *FakeBlueprints) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.BlueprintList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(blueprintsResource, blueprintsKind, c.ns, opts), &v1alpha1.BlueprintList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.BlueprintList{ListMeta: obj.(*v1alpha1.BlueprintList).ListMeta}
	for _, item := range obj.(*v1alpha1.BlueprintList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested blueprints.
func (c *FakeBlueprints) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(blueprintsResource, c.ns, opts))

}

// Create takes the representation of a blueprint and creates it.  Returns the server's representation of the blueprint, and an error, if there is any.
func (c *FakeBlueprints) Create(ctx context.Context, blueprint *v1alpha1.Blueprint, opts v1.CreateOptions) (result *v1alpha1.Blueprint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(blueprintsResource, c.ns, blueprint), &v1alpha1.Blueprint{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Blueprint), err
}

// Update takes the representation of a blueprint and updates it. Returns the server's representation of the blueprint, and an error, if there is any.
func (c *FakeBlueprints) Update(ctx context.Context, blueprint *v1alpha1.Blueprint, opts v1.UpdateOptions) (result *v1alpha1.Blueprint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(blueprintsResource, c.ns, blueprint), &v1alpha1.Blueprint{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Blueprint), err
}

// Delete takes name of the blueprint and deletes it. Returns an error if one occurs.
func (c *FakeBlueprints) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(blueprintsResource, c.ns, name, opts), &v1alpha1.Blueprint{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeBlueprints) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(blueprintsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.BlueprintList{})
	return err
}

// Patch applies the patch and returns the patched blueprint.
func (c *FakeBlueprints) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Blueprint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(blueprintsResource, c.ns, name, pt, data, subresources...), &v1alpha1.Blueprint{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Blueprint), err
}
