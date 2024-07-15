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
	"log"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	customresource "github.com/kanisterio/kanister/pkg/customresource"
	"github.com/kanisterio/kanister/pkg/field"
)

const (
	createOrUpdateCRDEnvVar = "CREATEORUPDATE_CRDS"
)

// CreateCustomResources creates the given custom resources and waits for them to initialize
func CreateCustomResources(ctx context.Context, config *rest.Config) error {
	crCTX, err := newOpKitContext(ctx, config)
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

func newOpKitContext(ctx context.Context, config *rest.Config) (*customresource.Context, error) {
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
		Context:               ctx,
	}, nil
}

// CreateRepoServerCustomResource creates the kopia repository server custom resource
func CreateRepoServerCustomResource(ctx context.Context, config *rest.Config) error {
	crCTX, err := newOpKitContext(ctx, config)
	if err != nil {
		return err
	}

	resources := []customresource.CustomResource{
		crv1alpha1.RepositoryServerResource,
	}

	return customresource.CreateCustomResources(*crCTX, resources)
}

func CreateOrUpdateCRDs() bool {
	createOrUpdateCRD := os.Getenv(createOrUpdateCRDEnvVar)
	if createOrUpdateCRD == "" {
		return true
	}

	c, err := strconv.ParseBool(createOrUpdateCRD)
	if err != nil {
		log.Print("environment variable", field.M{"CREATEORUPDATE_CRDS": createOrUpdateCRD})
		return true
	}

	return c
}
