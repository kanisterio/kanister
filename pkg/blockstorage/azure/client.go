package azure

import (
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/pkg/errors"
)

const (
	defaultSubscriptionEnv = "AZURE_SUBSCRIPTION_ID"
	// ToDo(ilya): resourcegroup needs to be pulled from metadata
	defaultResourceGroup = "AZURE_RESOURCE_GROUP"
)

// Client is a Kasten wrapper for Client client
type Client struct {
	SubscriptionID  string
	ResourceGroup   string
	Authorizer      autorest.Authorizer
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

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.WithError(err).Print("Failed to get Azure authorizer")
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
