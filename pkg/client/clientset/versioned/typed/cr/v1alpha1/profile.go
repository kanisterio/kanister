package v1alpha1

import (
	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
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
	Create(*v1alpha1.Profile) (*v1alpha1.Profile, error)
	Update(*v1alpha1.Profile) (*v1alpha1.Profile, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Profile, error)
	List(opts v1.ListOptions) (*v1alpha1.ProfileList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Profile, err error)
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
func (c *profiles) Get(name string, options v1.GetOptions) (result *v1alpha1.Profile, err error) {
	result = &v1alpha1.Profile{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("profiles").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Profiles that match those selectors.
func (c *profiles) List(opts v1.ListOptions) (result *v1alpha1.ProfileList, err error) {
	result = &v1alpha1.ProfileList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("profiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested profiles.
func (c *profiles) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("profiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a profile and creates it.  Returns the server's representation of the profile, and an error, if there is any.
func (c *profiles) Create(profile *v1alpha1.Profile) (result *v1alpha1.Profile, err error) {
	result = &v1alpha1.Profile{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("profiles").
		Body(profile).
		Do().
		Into(result)
	return
}

// Update takes the representation of a profile and updates it. Returns the server's representation of the profile, and an error, if there is any.
func (c *profiles) Update(profile *v1alpha1.Profile) (result *v1alpha1.Profile, err error) {
	result = &v1alpha1.Profile{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("profiles").
		Name(profile.Name).
		Body(profile).
		Do().
		Into(result)
	return
}

// Delete takes name of the profile and deletes it. Returns an error if one occurs.
func (c *profiles) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("profiles").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *profiles) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("profiles").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched profile.
func (c *profiles) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Profile, err error) {
	result = &v1alpha1.Profile{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("profiles").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
