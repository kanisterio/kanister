package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/consts"
)

const ActiveDirectory = "activeDirectory"

// currently available types: https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authorization
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
		// azure.PublicCloud.ActiveDirectoryEndpoint,
	}
}

// Public interface to authenticate with different Azure credentials type
type AzureAuthenticator interface {
	Authenticate(creds map[string]string) error
	GetAuthorizer() azcore.TokenCredential
}

type NewManagedIdentityCredentialFunc func(opts *azidentity.ManagedIdentityCredentialOptions) (*azidentity.ManagedIdentityCredential, error)
type NewClientSecretCredentialFunc func(tenantID, clientID, clientSecret string, opts *azidentity.ClientSecretCredentialOptions) (*azidentity.ClientSecretCredential, error)
type AzIdentityType struct {
	NewManagedIdentityCredential NewManagedIdentityCredentialFunc
	NewClientSecretCredential    NewClientSecretCredentialFunc
}

func NewAzureAuthenticator(config map[string]string, identity *AzIdentityType) (AzureAuthenticator, error) {
	if identity == nil {
		identity = &AzIdentityType{
			NewManagedIdentityCredential: azidentity.NewManagedIdentityCredential,
			NewClientSecretCredential:    azidentity.NewClientSecretCredential,
		}
	}

	switch {
	case isMSICredsAvailable(config):
		return &MsiAuthenticator{azIdentity: identity}, nil
	case isClientCredsAvailable(config):
		return &ClientSecretAuthenticator{azIdentity: identity}, nil
	default:
		return nil, errors.New("Fail to get an authenticator for provided creds combination")
	}
}

// authenticate with MSI creds
type MsiAuthenticator struct {
	azcore.TokenCredential
	azIdentity *AzIdentityType
}

func (m *MsiAuthenticator) GetAuthorizer() azcore.TokenCredential {
	return m.TokenCredential
}
func (m *MsiAuthenticator) Authenticate(config map[string]string) error {
	// check if MSI endpoint is available
	clientID, ok := config[blockstorage.AzureClientID]
	if !ok || clientID == "" {
		return errors.New("Failed to fetch azure clientID")
	}
	azClientID := azidentity.ClientID(clientID)
	opts := azidentity.ManagedIdentityCredentialOptions{ID: azClientID, ClientOptions: azcore.ClientOptions{}}
	opts.ClientOptions.Cloud = GetCloudConfig(config[blockstorage.AzureCloudEnvironmentID])
	cred, err := azidentity.NewManagedIdentityCredential(&opts)
	if err != nil {
		return errors.Wrap(err, "Failed to create an Azure Managed Identity credential")
	}
	m.TokenCredential = cred
	// config passed authentication
	return nil
}

// authenticate with client secret creds
type ClientSecretAuthenticator struct {
	azcore.TokenCredential
	azIdentity *AzIdentityType
}

func (c *ClientSecretAuthenticator) GetAuthorizer() azcore.TokenCredential {
	return c.TokenCredential
}
func (c *ClientSecretAuthenticator) Authenticate(creds map[string]string) error {
	credConfig, err := getCredConfigForAuth(creds)
	if err != nil {
		return errors.Wrap(err, "Failed to get Client Secret config")
	}
	opts := &azidentity.ClientSecretCredentialOptions{ClientOptions: azcore.ClientOptions{}}
	opts.ClientOptions.Cloud = GetCloudConfig(creds[blockstorage.AzureCloudEnvironmentID])
	cred, err := azidentity.NewClientSecretCredential(credConfig.TenantID, credConfig.ClientID, credConfig.ClientSecret, opts)
	if err != nil {
		return errors.Wrap(err, "Failed to create an Azure Client Secret credential")
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

func GetCloudConfig(cloudEnv string) cloud.Configuration {
	switch cloudEnv {
	case consts.AzureChinaCloud:
		return cloud.AzureChina
	case consts.AzureUSGovernmentCloud:
		return cloud.AzureGovernment
	default:
		return cloud.AzurePublic
	}
}
