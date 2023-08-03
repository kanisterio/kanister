package azure

import (
	"context"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/pkg/errors"
)

// currently available types: https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authorization
// to be available with azidentity: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#readme-credential-types
// determine if the combination of creds are client secret creds
func isClientCredsAvailable(config map[string]string) bool {
	return (config[blockstorage.AzureTenantID] != "" &&
		config[blockstorage.AzureClientID] != "" &&
		config[blockstorage.AzureClientSecret] != "")
}

// determine if the combination of creds are MSI creds
func isMSICredsAvailable(config map[string]string) bool {
	_, clientIDok := config[blockstorage.AzureClientID]
	return (clientIDok && config[blockstorage.AzureTenantID] == "" &&
		config[blockstorage.AzureClientSecret] == "")
}

// Public interface to authenticate with different Azure credentials type
type AzureAuthenticator interface {
	Authenticate(creds map[string]string) error
}

func NewAzureAuthenticator(config map[string]string) (AzureAuthenticator, error) {
	switch {
	case isMSICredsAvailable(config):
		return &MsiAuthenticator{}, nil
	case isClientCredsAvailable(config):
		return &ClientSecretAuthenticator{}, nil
	default:
		return nil, errors.New("Fail to get an authenticator for provided creds combination")
	}
}

// authenticate with MSI creds
type MsiAuthenticator struct{}

func (m *MsiAuthenticator) Authenticate(creds map[string]string) error {
	// check if MSI endpoint is available
	if !adal.MSIAvailable(context.Background(), nil) {
		return errors.New("MSI endpoint is not supported")
	}
	// create a service principal token
	msiConfig := auth.NewMSIConfig()
	if clientID, ok := creds[blockstorage.AzureClientID]; ok && clientID != "" {
		msiConfig.ClientID = clientID
	}
	spt, err := msiConfig.ServicePrincipalToken()
	if err != nil {
		return errors.Wrap(err, "Failed to create a service principal token")
	}
	// network call to check for token
	err = spt.Refresh()
	if err != nil {
		return errors.Wrap(err, "Failed to refresh token")
	}
	// creds passed authentication
	return nil
}

// authenticate with client secret creds
type ClientSecretAuthenticator struct{}

func (c *ClientSecretAuthenticator) Authenticate(creds map[string]string) error {
	credConfig, err := getCredConfigForAuth(creds)
	if err != nil {
		return errors.Wrap(err, "Failed to get Client Secret config")
	}
	// create a service principal token
	spt, err := credConfig.ServicePrincipalToken()
	if err != nil {
		return errors.Wrap(err, "Failed to create a service principal token")
	}
	// network call to check for token
	err = spt.Refresh()
	if err != nil {
		return errors.Wrap(err, "Failed to refresh token")
	}
	// creds passed authentication
	return nil
}

func getCredConfigForAuth(config map[string]string) (auth.ClientCredentialsConfig, error) {
	tenantID, ok := config[blockstorage.AzureTenantID]
	if !ok {
		return auth.ClientCredentialsConfig{}, errors.New("Cannot get tenantID from config")
	}

	clientID, ok := config[blockstorage.AzureClientID]
	if !ok {
		return auth.ClientCredentialsConfig{}, errors.New("Cannot get clientID from config")
	}

	clientSecret, ok := config[blockstorage.AzureClientSecret]
	if !ok {
		return auth.ClientCredentialsConfig{}, errors.New("Cannot get clientSecret from config")
	}

	credConfig := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	return credConfig, nil
}
