package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/pkg/errors"
)

const ActiveDirectory = "activeDirectory"

// currently avaialble types: https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authorization
// to be available with azidentity: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#readme-credential-types
// determine if the combination of creds are client secret creds
func isClientCredsAvailable(config map[string]string) bool {
	return config[blockstorage.AzureTenantID] != "" &&
		config[blockstorage.AzureClientID] != "" &&
		config[blockstorage.AzureClientSecret] != ""
}

// determine if the combination of creds are MSI creds
func isMSICredsAvailable(config map[string]string) bool {
	_, clientIDok := config[blockstorage.AzureClientID]
	return clientIDok && config[blockstorage.AzureTenantID] == "" &&
		config[blockstorage.AzureClientSecret] == ""
}

func isDefaultCredsAvailable(config map[string]string) bool {
	_, clientIDok := config[blockstorage.AzureClientID]
	_, tenantIDok := config[blockstorage.AzureTenantID]
	_, clientSecretOk := config[blockstorage.AzureClientSecret]
	return !clientIDok && !tenantIDok && !clientSecretOk
}

type ClientCredentialsConfig struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	AuxTenants   []string
	AADEndpoint  string
	Resource     string
}

// Defaults to Public Cloud and Resource Manager Endpoint.
func NewClientCredentialsConfig(clientID string, clientSecret string, tenantID string) ClientCredentialsConfig {
	return ClientCredentialsConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
		Resource:     cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint,
		//Todo: find a replacement for the AADEndpoint in the new azure sdk
		AADEndpoint: cloud.AzurePublic.Services[ActiveDirectory].Endpoint,
		//azure.PublicCloud.ActiveDirectoryEndpoint,
	}
}

// Public interface to authenticate with different Azure credentials type
type AzureAuthenticator interface {
	Authenticate(creds map[string]string) error
	GetTokenCredential() azcore.TokenCredential
}

func NewAzureAuthenticator(config map[string]string) (AzureAuthenticator, error) {
	// NewAzureAuthenticator opens up the possibility to Auth with:
	//1. Env variables
	//2. Managed Identity
	//3. Workload Identity
	//4. AzureCli
	switch {
	case isMSICredsAvailable(config):
		return &MsiAuthenticator{}, nil
	case isClientCredsAvailable(config):
		return &ClientSecretAuthenticator{}, nil
	case isDefaultCredsAvailable(config):
		return &DefaultAuthenticator{}, nil
	default:
		return nil, errors.New("Fail to get an authenticator for provided creds combination")
	}
}

// authenticate with default credential
type DefaultAuthenticator struct {
	azcore.TokenCredential
}

func (d *DefaultAuthenticator) GetTokenCredential() azcore.TokenCredential {
	return d.TokenCredential
}

func (d *DefaultAuthenticator) Authenticate(creds map[string]string) error {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create an Azure Default Identity credential")
	}
	_, err = cred.GetToken(context.Background(), policy.TokenRequestOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to create an access token")
	}
	d.TokenCredential = cred
	// creds passed authentication
	return nil
}

// authenticate with MSI creds
type MsiAuthenticator struct {
	azcore.TokenCredential
}

func (m *MsiAuthenticator) GetTokenCredential() azcore.TokenCredential {
	return m.TokenCredential
}
func (m *MsiAuthenticator) Authenticate(creds map[string]string) error {
	// check if MSI endpoint is available

	clientID, ok := creds[blockstorage.AzureClientID]
	if !ok || clientID == "" {
		return errors.New("Failed to fetch azure clientID")
	}
	azClientID := azidentity.ClientID(clientID)
	opts := azidentity.ManagedIdentityCredentialOptions{ID: azClientID}
	cred, err := azidentity.NewManagedIdentityCredential(&opts)
	if err != nil {
		return errors.Wrap(err, "Failed to create an Azure Managed Identity credential")
	}
	_, err = cred.GetToken(context.Background(), policy.TokenRequestOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to create an access token")
	}
	m.TokenCredential = cred
	// creds passed authentication
	return nil
}

// authenticate with client secret creds
type ClientSecretAuthenticator struct {
	azcore.TokenCredential
}

func (c *ClientSecretAuthenticator) GetTokenCredential() azcore.TokenCredential {
	return c.TokenCredential
}
func (c *ClientSecretAuthenticator) Authenticate(creds map[string]string) error {
	credConfig, err := getCredConfigForAuth(creds)
	if err != nil {
		return errors.Wrap(err, "Failed to get Client Secret config")
	}
	cred, err := azidentity.NewClientSecretCredential(credConfig.TenantID, credConfig.ClientID, credConfig.ClientSecret, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create an Azure Client Secret credential")
	}
	_, err = cred.GetToken(context.Background(), policy.TokenRequestOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to create an access token")
	}
	c.TokenCredential = cred
	// creds passed authentication
	return nil
}

func getCredConfigForAuth(config map[string]string) (ClientCredentialsConfig, error) {
	tenantID, ok := config[blockstorage.AzureTenantID]
	if !ok {
		return ClientCredentialsConfig{}, errors.New("Cannot get tenantID from config")
	}

	clientID, ok := config[blockstorage.AzureClientID]
	if !ok {
		return ClientCredentialsConfig{}, errors.New("Cannot get clientID from config")
	}

	clientSecret, ok := config[blockstorage.AzureClientSecret]
	if !ok {
		return ClientCredentialsConfig{}, errors.New("Cannot get clientSecret from config")
	}

	credConfig := NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	return credConfig, nil
}
