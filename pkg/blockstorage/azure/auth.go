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
	CredTypeManagedIdentity   string = "cred-type-msi"
	CredTypeClientSecret      string = "cred-type-client-secret"
	CredTypeClientCertificate string = "cred-type-client-certificate"
)

var availableCredsType = sets.NewString(CredTypeManagedIdentity)

type CredsValidity int

const (
	CredValidityUnknown CredsValidity = 0
	CredsTypeNotSupported
	CredsFailedAuthentication
	CredsPassedAuthentication
)

func isCredTypeSupported(credType string) bool {
	return availableCredsType.Has(credType)
}

func isClientCredsAvailable(config map[string]string) bool {
	if config[blockstorage.AzureTenantID] != "" &&
		config[blockstorage.AzureCientID] != "" &&
		config[blockstorage.AzureClentSecret] != "" {
		return true
	} else {
		return false
	}
}

func isMSICredsAvailable(config map[string]string) bool {
	if config[blockstorage.AzureTenantID] == "" &&
		config[blockstorage.AzureCientID] != "" &&
		config[blockstorage.AzureClentSecret] == "" {
		return true
	} else {
		return false
	}
}

// internal interface to authenticate with different Azure credentials type
type authenticator interface {
	authenticate(creds map[string]string) (CredsValidity, error)
}

// authenticate with MSI creds
type msiAuthenticator struct{}

func (m *msiAuthenticator) authenticate(creds map[string]string) (CredsValidity, error) {
	fmt.Println("msiAuthenticator.authenticate")
	// check if MSI endpoint is available
	if !adal.MSIAvailable(context.Background(), nil) {
		return CredsTypeNotSupported, errors.New("MSI endpoint is not supported")
	}
	// create a service principal token
	msiConfig := auth.NewMSIConfig()
	msiConfig.ClientID = creds[blockstorage.AzureTenantID]
	fmt.Println("msiAuthenticator client id: ", msiConfig.ClientID)
	spt, err := msiConfig.ServicePrincipalToken()
	if err != nil {
		return CredsFailedAuthentication, errors.Wrap(err, "Failed to create a service principal token")
	}
	// network call to check for token
	err = spt.Refresh()
	if err != nil {
		return CredsFailedAuthentication, errors.Wrap(err, "Failed to refresh token")
	}
	// creds passed authentication
	return CredsPassedAuthentication, nil
}

// return the authenticator based on credentials type
func getAuthenticator(credType string) authenticator {
	fmt.Println("credType in getAuthenticator: ", credType)
	switch credType {
	case CredTypeManagedIdentity:
		return &msiAuthenticator{}
	default:
		return nil
	}
}
