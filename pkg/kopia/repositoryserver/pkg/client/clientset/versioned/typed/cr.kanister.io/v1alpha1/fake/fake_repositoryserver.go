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

	v1alpha1 "github.com/kanisterio/kanister/pkg/kopia/repositoryserver/pkg/apis/cr.kanister.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRepositoryServers implements RepositoryServerInterface
type FakeRepositoryServers struct {
	Fake *FakeCrV1alpha1
	ns   string
}

var repositoryserversResource = schema.GroupVersionResource{Group: "cr.kanister.io", Version: "v1alpha1", Resource: "repositoryservers"}

var repositoryserversKind = schema.GroupVersionKind{Group: "cr.kanister.io", Version: "v1alpha1", Kind: "RepositoryServer"}

// Get takes name of the repositoryServer, and returns the corresponding repositoryServer object, and an error if there is any.
func (c *FakeRepositoryServers) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.RepositoryServer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(repositoryserversResource, c.ns, name), &v1alpha1.RepositoryServer{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RepositoryServer), err
}

// List takes label and field selectors, and returns the list of RepositoryServers that match those selectors.
func (c *FakeRepositoryServers) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.RepositoryServerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(repositoryserversResource, repositoryserversKind, c.ns, opts), &v1alpha1.RepositoryServerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.RepositoryServerList{ListMeta: obj.(*v1alpha1.RepositoryServerList).ListMeta}
	for _, item := range obj.(*v1alpha1.RepositoryServerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested repositoryServers.
func (c *FakeRepositoryServers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(repositoryserversResource, c.ns, opts))

}

// Create takes the representation of a repositoryServer and creates it.  Returns the server's representation of the repositoryServer, and an error, if there is any.
func (c *FakeRepositoryServers) Create(ctx context.Context, repositoryServer *v1alpha1.RepositoryServer, opts v1.CreateOptions) (result *v1alpha1.RepositoryServer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(repositoryserversResource, c.ns, repositoryServer), &v1alpha1.RepositoryServer{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RepositoryServer), err
}

// Update takes the representation of a repositoryServer and updates it. Returns the server's representation of the repositoryServer, and an error, if there is any.
func (c *FakeRepositoryServers) Update(ctx context.Context, repositoryServer *v1alpha1.RepositoryServer, opts v1.UpdateOptions) (result *v1alpha1.RepositoryServer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(repositoryserversResource, c.ns, repositoryServer), &v1alpha1.RepositoryServer{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RepositoryServer), err
}

// Delete takes name of the repositoryServer and deletes it. Returns an error if one occurs.
func (c *FakeRepositoryServers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(repositoryserversResource, c.ns, name, opts), &v1alpha1.RepositoryServer{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRepositoryServers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(repositoryserversResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.RepositoryServerList{})
	return err
}

// Patch applies the patch and returns the patched repositoryServer.
func (c *FakeRepositoryServers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.RepositoryServer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(repositoryserversResource, c.ns, name, pt, data, subresources...), &v1alpha1.RepositoryServer{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RepositoryServer), err
}
