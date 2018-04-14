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

// FakeProfiles implements ProfileInterface
type FakeProfiles struct {
	Fake *FakeCrV1alpha1
	ns   string
}

var profilesResource = schema.GroupVersionResource{Group: "cr", Version: "v1alpha1", Resource: "profiles"}

var profilesKind = schema.GroupVersionKind{Group: "cr", Version: "v1alpha1", Kind: "Profile"}

// Get takes name of the profile, and returns the corresponding profile object, and an error if there is any.
func (c *FakeProfiles) Get(name string, options v1.GetOptions) (result *v1alpha1.Profile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(profilesResource, c.ns, name), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}

// List takes label and field selectors, and returns the list of Profiles that match those selectors.
func (c *FakeProfiles) List(opts v1.ListOptions) (result *v1alpha1.ProfileList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(profilesResource, profilesKind, c.ns, opts), &v1alpha1.ProfileList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ProfileList{}
	for _, item := range obj.(*v1alpha1.ProfileList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested profiles.
func (c *FakeProfiles) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(profilesResource, c.ns, opts))

}

// Create takes the representation of a profile and creates it.  Returns the server's representation of the profile, and an error, if there is any.
func (c *FakeProfiles) Create(profile *v1alpha1.Profile) (result *v1alpha1.Profile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(profilesResource, c.ns, profile), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}

// Update takes the representation of a profile and updates it. Returns the server's representation of the profile, and an error, if there is any.
func (c *FakeProfiles) Update(profile *v1alpha1.Profile) (result *v1alpha1.Profile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(profilesResource, c.ns, profile), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}

// Delete takes name of the profile and deletes it. Returns an error if one occurs.
func (c *FakeProfiles) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(profilesResource, c.ns, name), &v1alpha1.Profile{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeProfiles) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(profilesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ProfileList{})
	return err
}

// Patch applies the patch and returns the patched profile.
func (c *FakeProfiles) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Profile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(profilesResource, c.ns, name, data, subresources...), &v1alpha1.Profile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Profile), err
}
