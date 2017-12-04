package v1alpha1

import (
	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	scheme "github.com/kanisterio/kanister/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// BlueprintsGetter has a method to return a BlueprintInterface.
// A group's client should implement this interface.
type BlueprintsGetter interface {
	Blueprints(namespace string) BlueprintInterface
}

// BlueprintInterface has methods to work with Blueprint resources.
type BlueprintInterface interface {
	Create(*v1alpha1.Blueprint) (*v1alpha1.Blueprint, error)
	Update(*v1alpha1.Blueprint) (*v1alpha1.Blueprint, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Blueprint, error)
	List(opts v1.ListOptions) (*v1alpha1.BlueprintList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Blueprint, err error)
	BlueprintExpansion
}

// blueprints implements BlueprintInterface
type blueprints struct {
	client rest.Interface
	ns     string
}

// newBlueprints returns a Blueprints
func newBlueprints(c *CrV1alpha1Client, namespace string) *blueprints {
	return &blueprints{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the blueprint, and returns the corresponding blueprint object, and an error if there is any.
func (c *blueprints) Get(name string, options v1.GetOptions) (result *v1alpha1.Blueprint, err error) {
	result = &v1alpha1.Blueprint{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("blueprints").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Blueprints that match those selectors.
func (c *blueprints) List(opts v1.ListOptions) (result *v1alpha1.BlueprintList, err error) {
	result = &v1alpha1.BlueprintList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("blueprints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested blueprints.
func (c *blueprints) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("blueprints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a blueprint and creates it.  Returns the server's representation of the blueprint, and an error, if there is any.
func (c *blueprints) Create(blueprint *v1alpha1.Blueprint) (result *v1alpha1.Blueprint, err error) {
	result = &v1alpha1.Blueprint{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("blueprints").
		Body(blueprint).
		Do().
		Into(result)
	return
}

// Update takes the representation of a blueprint and updates it. Returns the server's representation of the blueprint, and an error, if there is any.
func (c *blueprints) Update(blueprint *v1alpha1.Blueprint) (result *v1alpha1.Blueprint, err error) {
	result = &v1alpha1.Blueprint{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("blueprints").
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
		Resource("blueprints").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *blueprints) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("blueprints").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched blueprint.
func (c *blueprints) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Blueprint, err error) {
	result = &v1alpha1.Blueprint{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("blueprints").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
