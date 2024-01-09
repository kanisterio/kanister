// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discovery

import (
	"context"

	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kanisterio/kanister/pkg/filter"
)

// CRDMatcher returns a ResourceTypeMatcher that matches all CRs in this cluster.
func CRDMatcher(ctx context.Context, cli crdclient.Interface) (filter.ResourceTypeMatcher, error) {
	crds, err := cli.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query CRDs in cluster")
	}
	return crdsToMatcher(crds.Items), nil
}

func crdsToMatcher(crds []apiextensionsv1.CustomResourceDefinition) filter.ResourceTypeMatcher {
	gvrs := make(filter.ResourceTypeMatcher, 0, len(crds))
	for _, crd := range crds {
		gvr := filter.ResourceTypeRequirement{
			Group: crd.Spec.Group,
			// Omit Version-- match any version
			Resource: crd.Spec.Names.Plural,
		}
		gvrs = append(gvrs, gvr)
	}
	return gvrs
}
