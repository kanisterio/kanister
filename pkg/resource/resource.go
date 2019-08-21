package resource

import (
	"context"
	"time"

	"github.com/pkg/errors"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	customresource "github.com/kanisterio/kanister/pkg/customresource"
)

// CreateCustomResources creates the given custom resources and waits for them to initialize
func CreateCustomResources(ctx context.Context, config *rest.Config) error {
	crCTX, err := newOpKitContext(config)
	if err != nil {
		return err
	}
	resources := []customresource.CustomResource{
		crv1alpha1.ActionSetResource,
		crv1alpha1.BlueprintResource,
		crv1alpha1.ProfileResource,
	}
	return customresource.CreateCustomResources(*crCTX, resources)
}

func newOpKitContext(config *rest.Config) (*customresource.Context, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client.")
	}
	apiExtClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s API extension clientset")
	}
	return &customresource.Context{
		Clientset:             clientset,
		APIExtensionClientset: apiExtClientset,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}, nil
}
