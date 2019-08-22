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

package resource

import (
	"context"
	"time"

	"github.com/pkg/errors"
	opkit "github.com/rook/operator-kit"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// CreateCustomResources creates the given custom resources and waits for them to initialize
func CreateCustomResources(ctx context.Context, config *rest.Config) error {
	opKitCTX, err := newOpKitContext(config)
	if err != nil {
		return err
	}
	resources := []opkit.CustomResource{
		crv1alpha1.ActionSetResource,
		crv1alpha1.BlueprintResource,
		crv1alpha1.ProfileResource,
	}
	return opkit.CreateCustomResources(*opKitCTX, resources)
}

func newOpKitContext(config *rest.Config) (*opkit.Context, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client.")
	}
	apiExtClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s API extension clientset")
	}
	return &opkit.Context{
		Clientset:             clientset,
		APIExtensionClientset: apiExtClientset,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}, nil
}
