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

package azure

import (
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/pkg/errors"
)

const (
	defaultSubscriptionEnv = "AZURE_SUBSCRIPTION_ID"
	defaultResourceGroup   = "AZURE_RESOURCE_GROUP"
)

// Client is a wrapper for Client client
type Client struct {
	SubscriptionID  string
	ResourceGroup   string
	Authorizer      *autorest.BearerAuthorizer
	DisksClient     *compute.DisksClient
	SnapshotsClient *compute.SnapshotsClient
}

// NewClient returns a Client struct
func NewClient(ctx context.Context) (*Client, error) {
	var resourceGroup string
	var subscriptionID string
	var ok bool
	var err error

	metadata := NewInstanceMetadata()

	if resourceGroup, ok = os.LookupEnv(defaultResourceGroup); !ok {
		log.Debug().Print("AZURE_RESOURCE_GROUP is not setup")
		resourceGroup, err = metadata.Text("instance/compute/resourceGroupName")
		if err != nil {
			return nil, errors.Wrap(err, "Cannot get resourceGroup from instance metadata")
		}
	}

	if subscriptionID, ok = os.LookupEnv(defaultSubscriptionEnv); !ok {
		log.Debug().Print("AZURE_SUBSCRIPTION_ID is not setup")
		subscriptionID, err = metadata.Text("instance/compute/subscriptionId")
		if err != nil {
			return nil, errors.Wrap(err, "Cannot get subscriptionID from instance metadata")
		}
	}

	authorizer, err := getAuthorizer(azure.PublicCloud)
	if err != nil {
		return nil, err
	}

	disksClient := compute.NewDisksClient(subscriptionID)
	disksClient.Authorizer = authorizer

	snapshotsClient := compute.NewSnapshotsClient(subscriptionID)
	snapshotsClient.Authorizer = authorizer

	return &Client{
		SubscriptionID:  subscriptionID,
		Authorizer:      authorizer,
		DisksClient:     &disksClient,
		SnapshotsClient: &snapshotsClient,
		ResourceGroup:   resourceGroup,
	}, nil
}

func getAuthorizer(env azure.Environment) (*autorest.BearerAuthorizer, error) {
	tenantID := os.Getenv("AZURE_TENANT_ID")
	if tenantID == "" {
		return nil, errors.New("Cannot get tenantID from environment")
	}

	clientID := os.Getenv("AZURE_CLIENT_ID")
	if clientID == "" {
		return nil, errors.New("Cannot get clientID from environment")
	}

	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, errors.New("Cannot get clientSecret from environment")
	}

	config := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	a, err := config.Authorizer()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get Azure authorizer")
	}
	ba, ok := a.(*autorest.BearerAuthorizer)
	if !ok {
		return nil, errors.New("Failed to get Azure authorizer")
	}
	return ba, nil
}
