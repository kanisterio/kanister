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

// TODO: Switch to using the latest azure sdk and remove nolint.
// Related Ticket- https://github.com/kanisterio/kanister/issues/168
package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/log"
)

// Client is a wrapper
type Client struct {
	Cred                azcore.TokenCredential
	SubscriptionID      string
	ResourceGroup       string
	BaseURI             string
	DisksClient         *armcompute.DisksClient
	SnapshotsClient     *armcompute.SnapshotsClient
	SKUsClient          *armcompute.ResourceSKUsClient
	SubscriptionsClient *armsubscriptions.Client
}

// NewClient returns a Client struct
var (
	computeClientFactory       *armcompute.ClientFactory
	subscriptionsClientFactory *armsubscriptions.ClientFactory
)

func NewClient(ctx context.Context, config map[string]string) (*Client, error) {
	var resourceGroup string
	var subscriptionID string
	var ok bool
	var err error

	metadata := NewInstanceMetadata()

	if resourceGroup, ok = config[blockstorage.AzureResurceGroup]; !ok {
		log.Debug().Print("AZURE_RESOURCE_GROUP is not setup")
		resourceGroup, err = metadata.Text("instance/compute/resourceGroupName")
		if err != nil {
			return nil, errors.Wrap(err, "Cannot get resourceGroup from instance metadata")
		}
	}

	if subscriptionID, ok = config[blockstorage.AzureSubscriptionID]; !ok {
		log.Debug().Print("AZURE_SUBSCRIPTION_ID is not setup")
		subscriptionID, err = metadata.Text("instance/compute/subscriptionId")
		if err != nil {
			return nil, errors.Wrap(err, "Cannot get subscriptionID from instance metadata")
		}
	}

	authenticator, err := NewAzureAuthenticator(config)
	if err != nil {
		return nil, err
	}
	err = authenticator.Authenticate(config)
	if err != nil {
		return nil, err
	}
	cred := authenticator.GetAuthorizer()
	computeClientFactory, err = armcompute.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	subscriptionsClientFactory, err = armsubscriptions.NewClientFactory(cred, nil)

	if err != nil {
		return nil, err
	}

	disksClient := computeClientFactory.NewDisksClient()
	snapshotsClient := computeClientFactory.NewSnapshotsClient()
	skusClient := computeClientFactory.NewResourceSKUsClient()
	subscriptionsClient := subscriptionsClientFactory.NewClient()

	if err != nil {
		return nil, err
	}

	return &Client{
		Cred:                cred,
		BaseURI:             config[blockstorage.AzureResurceMgrEndpoint],
		SubscriptionID:      subscriptionID,
		DisksClient:         disksClient,
		SnapshotsClient:     snapshotsClient,
		SKUsClient:          skusClient,
		SubscriptionsClient: subscriptionsClient,
		ResourceGroup:       resourceGroup,
	}, nil
}

func getCredConfig(env Environment, config map[string]string) (ClientCredentialsConfig, error) {
	credConfig, err := getCredConfigForAuth(config)
	if err != nil {
		return ClientCredentialsConfig{}, err
	}

	//Todo: Find alternatives to azure.Environment
	var ok bool
	if credConfig.AADEndpoint, ok = config[blockstorage.AzureActiveDirEndpoint]; !ok || credConfig.AADEndpoint == "" {
		credConfig.AADEndpoint = env.Configuration.ActiveDirectoryAuthorityHost
		config[blockstorage.AzureActiveDirEndpoint] = credConfig.AADEndpoint
	}

	if credConfig.Resource, ok = config[blockstorage.AzureActiveDirResourceID]; !ok || credConfig.Resource == "" {
		credConfig.Resource = env.Configuration.Services[cloud.ResourceManager].Endpoint
		config[blockstorage.AzureActiveDirResourceID] = credConfig.Resource
	}

	return credConfig, nil
}
