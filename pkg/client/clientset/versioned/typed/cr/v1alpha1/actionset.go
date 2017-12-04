package v1alpha1

import (
	v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
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
	Create(*v1alpha1.ActionSet) (*v1alpha1.ActionSet, error)
	Update(*v1alpha1.ActionSet) (*v1alpha1.ActionSet, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ActionSet, error)
	List(opts v1.ListOptions) (*v1alpha1.ActionSetList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ActionSet, err error)
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
func (c *actionSets) Get(name string, options v1.GetOptions) (result *v1alpha1.ActionSet, err error) {
	result = &v1alpha1.ActionSet{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("actionsets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ActionSets that match those selectors.
func (c *actionSets) List(opts v1.ListOptions) (result *v1alpha1.ActionSetList, err error) {
	result = &v1alpha1.ActionSetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("actionsets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested actionSets.
func (c *actionSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("actionsets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a actionSet and creates it.  Returns the server's representation of the actionSet, and an error, if there is any.
func (c *actionSets) Create(actionSet *v1alpha1.ActionSet) (result *v1alpha1.ActionSet, err error) {
	result = &v1alpha1.ActionSet{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("actionsets").
		Body(actionSet).
		Do().
		Into(result)
	return
}

// Update takes the representation of a actionSet and updates it. Returns the server's representation of the actionSet, and an error, if there is any.
func (c *actionSets) Update(actionSet *v1alpha1.ActionSet) (result *v1alpha1.ActionSet, err error) {
	result = &v1alpha1.ActionSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("actionsets").
		Name(actionSet.Name).
		Body(actionSet).
		Do().
		Into(result)
	return
}

// Delete takes name of the actionSet and deletes it. Returns an error if one occurs.
func (c *actionSets) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("actionsets").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *actionSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("actionsets").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched actionSet.
func (c *actionSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ActionSet, err error) {
	result = &v1alpha1.ActionSet{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("actionsets").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
