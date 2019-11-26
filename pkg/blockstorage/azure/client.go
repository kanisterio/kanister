package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/pkg/errors"
)

const (
	defaultSubscriptionEnv = "AZURE_SUBSCRIPTION_ID"
	defaultResourceGroup   = "AZURE_RESOURCE_GROUP"
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
func NewClient(config map[string]string) (*Client, error) {
	var resourceGroup string
	var subscriptionID string
	var ok bool
	var err error

	authEnv, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		log.WithError(err).Print("Failed to get Azure environment settings")
		return nil, err
	}
	for key, value := range config {
		authEnv.Values[key] = value
	}
	authorizer, err := authEnv.GetAuthorizer()
	if err != nil {
		log.WithError(err).Print("Failed to get Azure authorizer")
		return nil, err
	}

	if resourceGroup, ok = authEnv.Values[defaultResourceGroup]; !ok {
		log.WithError(err).Print("AZURE_RESOURCE_GROUP is not setup")
		return nil, errors.Wrap(err, "AZURE_RESOURCE_GROUP is not setup")
	}

	if subscriptionID, ok = authEnv.Values[defaultSubscriptionEnv]; !ok {
		log.WithError(err).Print("AZURE_SUBSCRIPTION_ID is not setup")
		return nil, errors.Wrap(err, "AZURE_SUBSCRIPTION_ID is not setup")
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
