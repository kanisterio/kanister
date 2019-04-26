package filter

import (
	"context"

	"github.com/pkg/errors"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CRDMatcher returns a ResourceMatcher that matches all CRs in this cluster.
func CRDMatcher(ctx context.Context, cli crdclient.Interface) (ResourceMatcher, error) {
	crds, err := cli.ApiextensionsV1beta1().CustomResourceDefinitions().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query CRDs in cluster")
	}
	return crdsToMatcher(crds.Items), nil
}

func crdsToMatcher(crds []apiextensions.CustomResourceDefinition) ResourceMatcher {
	gvrs := make(ResourceMatcher, 0, len(crds))
	for _, crd := range crds {
		gvr := ResourceRequirement{
			Group:    crd.Spec.Group,
			Version:  crd.Spec.Version,
			Resource: crd.Spec.Names.Plural,
		}
		gvrs = append(gvrs, gvr)
	}
	return gvrs
}
