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

// FakeActionSets implements ActionSetInterface
type FakeActionSets struct {
	Fake *FakeCrV1alpha1
	ns   string
}

var actionsetsResource = schema.GroupVersionResource{Group: "cr", Version: "v1alpha1", Resource: "actionsets"}

var actionsetsKind = schema.GroupVersionKind{Group: "cr", Version: "v1alpha1", Kind: "ActionSet"}

// Get takes name of the actionSet, and returns the corresponding actionSet object, and an error if there is any.
func (c *FakeActionSets) Get(name string, options v1.GetOptions) (result *v1alpha1.ActionSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(actionsetsResource, c.ns, name), &v1alpha1.ActionSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ActionSet), err
}

// List takes label and field selectors, and returns the list of ActionSets that match those selectors.
func (c *FakeActionSets) List(opts v1.ListOptions) (result *v1alpha1.ActionSetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(actionsetsResource, actionsetsKind, c.ns, opts), &v1alpha1.ActionSetList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ActionSetList{}
	for _, item := range obj.(*v1alpha1.ActionSetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested actionSets.
func (c *FakeActionSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(actionsetsResource, c.ns, opts))

}

// Create takes the representation of a actionSet and creates it.  Returns the server's representation of the actionSet, and an error, if there is any.
func (c *FakeActionSets) Create(actionSet *v1alpha1.ActionSet) (result *v1alpha1.ActionSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(actionsetsResource, c.ns, actionSet), &v1alpha1.ActionSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ActionSet), err
}

// Update takes the representation of a actionSet and updates it. Returns the server's representation of the actionSet, and an error, if there is any.
func (c *FakeActionSets) Update(actionSet *v1alpha1.ActionSet) (result *v1alpha1.ActionSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(actionsetsResource, c.ns, actionSet), &v1alpha1.ActionSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ActionSet), err
}

// Delete takes name of the actionSet and deletes it. Returns an error if one occurs.
func (c *FakeActionSets) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(actionsetsResource, c.ns, name), &v1alpha1.ActionSet{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeActionSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(actionsetsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ActionSetList{})
	return err
}

// Patch applies the patch and returns the patched actionSet.
func (c *FakeActionSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ActionSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(actionsetsResource, c.ns, name, data, subresources...), &v1alpha1.ActionSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ActionSet), err
}
