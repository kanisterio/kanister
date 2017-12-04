package fake

import (
	v1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeCrV1alpha1 struct {
	*testing.Fake
}

func (c *FakeCrV1alpha1) ActionSets(namespace string) v1alpha1.ActionSetInterface {
	return &FakeActionSets{c, namespace}
}

func (c *FakeCrV1alpha1) Blueprints(namespace string) v1alpha1.BlueprintInterface {
	return &FakeBlueprints{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeCrV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
