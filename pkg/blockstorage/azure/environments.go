package azure

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"strings"
)

const (
	// NotAvailable is used for endpoints and resource IDs that are not available for a given cloud.
	NotAvailable = "N/A"
)

var environments = map[string]Environment{
	"AZURECHINACLOUD":        ChinaCloud,
	"AZUREGERMANCLOUD":       GermanCloud,
	"AZURECLOUD":             PublicCloud,
	"AZUREPUBLICCLOUD":       PublicCloud,
	"AZUREUSGOVERNMENT":      USGovernmentCloud,
	"AZUREUSGOVERNMENTCLOUD": USGovernmentCloud,
}

// Environment represents a set of endpoints for each of Azure's Clouds.
type Environment struct {
	Name                    string `json:"name"`
	ResourceManagerEndpoint string `json:"resourceManagerEndpoint"`
	ActiveDirectoryEndpoint string `json:"activeDirectoryEndpoint"`
	StorageEndpointSuffix   string `json:"storageEndpointSuffix"`
	Configuration           cloud.Configuration
}

var (
	// PublicCloud is the default public Azure cloud environment
	//Ref: https://github.com/Azure/azure-sdk-for-go/blob/sdk/storage/azblob/v0.4.0/eng/common/TestResources/clouds/AzureCloud.json
	PublicCloud = Environment{
		Name:                    "AzurePublicCloud",
		ResourceManagerEndpoint: "https://management.azure.com/",
		ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
		StorageEndpointSuffix:   "core.windows.net",
		Configuration:           cloud.AzurePublic,
	}

	// USGovernmentCloud is the cloud environment for the US Government
	//Ref: https://github.com/Azure/azure-sdk-for-go/blob/sdk/storage/azblob/v0.4.0/eng/common/TestResources/clouds/AzureUSGovernment.json
	USGovernmentCloud = Environment{
		Name:                    "AzureUSGovernmentCloud",
		ResourceManagerEndpoint: "https://management.usgovcloudapi.net/",
		ActiveDirectoryEndpoint: "https://login.microsoftonline.us/",
		StorageEndpointSuffix:   "core.usgovcloudapi.net",
		Configuration:           cloud.AzureGovernment,
	}

	// ChinaCloud is the cloud environment operated in China
	//Ref: https://github.com/Azure/azure-sdk-for-go/blob/sdk/storage/azblob/v0.4.0/eng/common/TestResources/clouds/AzureChinaCloud.json
	ChinaCloud = Environment{
		Name:                    "AzureChinaCloud",
		ResourceManagerEndpoint: "https://management.chinacloudapi.cn/",
		ActiveDirectoryEndpoint: "https://login.chinacloudapi.cn/",
		StorageEndpointSuffix:   "core.chinacloudapi.cn",
		Configuration:           cloud.AzureChina,
	}

	// GermanCloud is the cloud environment operated in Germany has been deprecated
	// Ref: https://learn.microsoft.com/en-us/previous-versions/azure/germany/germany-welcome
	GermanCloud = Environment{
		Name:                    "AzureGermanCloud",
		ResourceManagerEndpoint: NotAvailable,
		ActiveDirectoryEndpoint: NotAvailable,
		StorageEndpointSuffix:   NotAvailable,
		Configuration:           cloud.Configuration{},
	}
)

// EnvironmentFromName returns an Environment based on the common name specified.
func EnvironmentFromName(name string) (Environment, error) {
	name = strings.ToUpper(name)
	env, ok := environments[name]
	if !ok {
		return env, fmt.Errorf("environment/azure: There is no cloud environment matching the name %q", name)
	}

	return env, nil
}
