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
//
//nolint:staticcheck
package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/pkg/errors"
)

// Client is a wrapper for Client client
type Client struct {
	SubscriptionID  string
	ResourceGroup   string
	BaseURI         string
	Authorizer      *autorest.BearerAuthorizer
	DisksClient     *compute.DisksClient
	SnapshotsClient *compute.SnapshotsClient
}

// NewClient returns a Client struct
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

	if id, ok := config[blockstorage.AzureCloudEnvironmentID]; !ok || id == "" {
		config[blockstorage.AzureCloudEnvironmentID] = azure.PublicCloud.Name
	}

	env, err := azure.EnvironmentFromName(config[blockstorage.AzureCloudEnvironmentID])
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch the cloud environment.")
	}

	authorizer, err := getAuthorizer(env, config)
	if err != nil {
		return nil, err
	}

	_, ok = config[blockstorage.AzureResurceMgrEndpoint]
	if !ok {
		config[blockstorage.AzureResurceMgrEndpoint] = env.ResourceManagerEndpoint
	}

	disksClient := compute.NewDisksClientWithBaseURI(config[blockstorage.AzureResurceMgrEndpoint], subscriptionID)
	disksClient.Authorizer = authorizer

	snapshotsClient := compute.NewSnapshotsClientWithBaseURI(config[blockstorage.AzureResurceMgrEndpoint], subscriptionID)
	snapshotsClient.Authorizer = authorizer

	return &Client{
		BaseURI:         config[blockstorage.AzureResurceMgrEndpoint],
		SubscriptionID:  subscriptionID,
		Authorizer:      authorizer,
		DisksClient:     &disksClient,
		SnapshotsClient: &snapshotsClient,
		ResourceGroup:   resourceGroup,
	}, nil
}

//nolint:unparam
func getAuthorizer(env azure.Environment, config map[string]string) (*autorest.BearerAuthorizer, error) {
	if isClientCredsAvailable(config) {
		return getClientCredsAuthorizer(env, config)
	} else if isMSICredsAvailable(config) {
		return getMSIsAuthorizer(config)
	}
	return nil, errors.New("Missing credentials, or credential type not supported")
}

func getClientCredsAuthorizer(env azure.Environment, config map[string]string) (*autorest.BearerAuthorizer, error) {
	credConfig, err := getCredConfig(env, config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get Azure Client Credentials Config")
	}
	a, err := credConfig.Authorizer()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get Azure Client Credentials authorizer")
	}
	ba, ok := a.(*autorest.BearerAuthorizer)
	if !ok {
		return nil, errors.New("Failed to get Azure authorizer")
	}
	return ba, nil
}

func getMSIsAuthorizer(config map[string]string) (*autorest.BearerAuthorizer, error) {
	msiConfig := auth.NewMSIConfig()
	msiConfig.ClientID = config[blockstorage.AzureClientID]
	a, err := msiConfig.Authorizer()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get Azure MSI authorizer")
	}
	ba, ok := a.(*autorest.BearerAuthorizer)
	if !ok {
		return nil, errors.New("Failed to get Azure authorizer")
	}
	return ba, nil
}

func getCredConfig(env azure.Environment, config map[string]string) (auth.ClientCredentialsConfig, error) {
	credConfig, err := getCredConfigForAuth(config)
	if err != nil {
		return auth.ClientCredentialsConfig{}, err
	}
	var ok bool
	if credConfig.AADEndpoint, ok = config[blockstorage.AzureActiveDirEndpoint]; !ok || credConfig.AADEndpoint == "" {
		credConfig.AADEndpoint = env.ActiveDirectoryEndpoint
		config[blockstorage.AzureActiveDirEndpoint] = credConfig.AADEndpoint
	}

	if credConfig.Resource, ok = config[blockstorage.AzureActiveDirResourceID]; !ok || credConfig.Resource == "" {
		credConfig.Resource = env.ResourceManagerEndpoint
		config[blockstorage.AzureActiveDirResourceID] = credConfig.Resource
	}

	return credConfig, nil
}
