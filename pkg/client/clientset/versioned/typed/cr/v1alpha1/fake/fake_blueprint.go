package fake

import (
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

var blueprintsResource = schema.GroupVersionResource{Group: "cr", Version: "v1alpha1", Resource: "blueprints"}

var blueprintsKind = schema.GroupVersionKind{Group: "cr", Version: "v1alpha1", Kind: "Blueprint"}

// Get takes name of the blueprint, and returns the corresponding blueprint object, and an error if there is any.
func (c *FakeBlueprints) Get(name string, options v1.GetOptions) (result *v1alpha1.Blueprint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(blueprintsResource, c.ns, name), &v1alpha1.Blueprint{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Blueprint), err
}

// List takes label and field selectors, and returns the list of Blueprints that match those selectors.
func (c *FakeBlueprints) List(opts v1.ListOptions) (result *v1alpha1.BlueprintList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(blueprintsResource, blueprintsKind, c.ns, opts), &v1alpha1.BlueprintList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.BlueprintList{}
	for _, item := range obj.(*v1alpha1.BlueprintList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested blueprints.
func (c *FakeBlueprints) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(blueprintsResource, c.ns, opts))

}

// Create takes the representation of a blueprint and creates it.  Returns the server's representation of the blueprint, and an error, if there is any.
func (c *FakeBlueprints) Create(blueprint *v1alpha1.Blueprint) (result *v1alpha1.Blueprint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(blueprintsResource, c.ns, blueprint), &v1alpha1.Blueprint{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Blueprint), err
}

// Update takes the representation of a blueprint and updates it. Returns the server's representation of the blueprint, and an error, if there is any.
func (c *FakeBlueprints) Update(blueprint *v1alpha1.Blueprint) (result *v1alpha1.Blueprint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(blueprintsResource, c.ns, blueprint), &v1alpha1.Blueprint{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Blueprint), err
}

// Delete takes name of the blueprint and deletes it. Returns an error if one occurs.
func (c *FakeBlueprints) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(blueprintsResource, c.ns, name), &v1alpha1.Blueprint{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeBlueprints) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(blueprintsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.BlueprintList{})
	return err
}

// Patch applies the patch and returns the patched blueprint.
func (c *FakeBlueprints) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Blueprint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(blueprintsResource, c.ns, name, data, subresources...), &v1alpha1.Blueprint{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Blueprint), err
}
