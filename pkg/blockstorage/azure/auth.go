package azure

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

// currently avaialble types: https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authorization
// to be available with azidentity: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#readme-credential-types
// to faciliate future migration, only the common ones between two SDKs are listed
const (
	CredTypeManagedIdentity string = "cred-type-msi"
	CredTypeClientSecret    string = "cred-type-client-secret"
)

var availableCredsType = sets.NewString(CredTypeManagedIdentity, CredTypeClientSecret)

func isCredTypeSupported(credType string) bool {
	return availableCredsType.Has(credType)
}

// determine if the combination of creds are client secret creds
func IsClientCredsAvailable(config map[string]string) bool {
	return (config[blockstorage.AzureTenantID] != "" &&
		config[blockstorage.AzureCientID] != "" &&
		config[blockstorage.AzureClentSecret] != "")
}

// determine if the combination of creds are MSI creds
func IsMSICredsAvailable(config map[string]string) bool {
	return (config[blockstorage.AzureTenantID] == "" &&
		config[blockstorage.AzureCientID] != "" &&
		config[blockstorage.AzureClentSecret] == "")
}

// internal interface to authenticate with different Azure credentials type
type authenticator interface {
	authenticate(creds map[string]string) error
}

// authenticate with MSI creds
type msiAuthenticator struct{}

func (m *msiAuthenticator) authenticate(creds map[string]string) error {
	fmt.Println("msiAuthenticator.authenticate")
	// check if MSI endpoint is available
	if !adal.MSIAvailable(context.Background(), nil) {
		return errors.New("MSI endpoint is not supported")
	}
	// create a service principal token
	msiConfig := auth.NewMSIConfig()
	msiConfig.ClientID = creds[blockstorage.AzureCientID]
	fmt.Println("msiAuthenticator client id: ", msiConfig.ClientID)
	spt, err := msiConfig.ServicePrincipalToken()
	if err != nil {
		return errors.Wrap(err, "Failed to create a service principal token")
	}
	// network call to check for token
	err = spt.Refresh()
	if err != nil {
		fmt.Println("Failed to refresh token for msiAuthenticator", err)
		return errors.Wrap(err, "Failed to refresh token")
	}
	// creds passed authentication
	return nil
}

type clientSecretAuthenticator struct{}

func (c *clientSecretAuthenticator) authenticate(creds map[string]string) error {
	fmt.Println("clientSecretAuthenticator.authenticate")
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
		fmt.Println("Failed to refresh token for clientSecretAuthenticator", err)
		return errors.Wrap(err, "Failed to refresh token")
	}
	// creds passed authentication
	return nil
}

// return the authenticator based on credentials type
func getAuthenticator(credType string) authenticator {
	fmt.Println("credType in getAuthenticator: ", credType)
	switch credType {
	case CredTypeManagedIdentity:
		return &msiAuthenticator{}
	case CredTypeClientSecret:
		return &clientSecretAuthenticator{}
	default:
		return nil
	}
}

func getCredConfigForAuth(config map[string]string) (auth.ClientCredentialsConfig, error) {
	tenantID, ok := config[blockstorage.AzureTenantID]
	if !ok {
		return auth.ClientCredentialsConfig{}, errors.New("Cannot get tenantID from config")
	}

	clientID, ok := config[blockstorage.AzureCientID]
	if !ok {
		return auth.ClientCredentialsConfig{}, errors.New("Cannot get clientID from config")
	}

	clientSecret, ok := config[blockstorage.AzureClentSecret]
	if !ok {
		return auth.ClientCredentialsConfig{}, errors.New("Cannot get clientSecret from config")
	}

	credConfig := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)

	fmt.Println("Finished getCredConfigForAuth", credConfig)
	return credConfig, nil
}
