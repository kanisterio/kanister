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

package ibm

const (
	// APIKeyArgName is used to pass ibmbluemix api key into client.
	APIKeyArgName = "api-key"
	// SLAPIKeyArgName is used to pass SoftLayer api key into client.
	SLAPIKeyArgName = "sl-api-key"
	// SLAPIUsernameArgName is used to pass SoftLayer api username into client.
	SLAPIUsernameArgName = "sl-api-username"
	// CfgSecretNameArgName is used to pass in cluster secret name.
	CfgSecretNameArgName = "secret-name"
	// CfgSecretNameSpaceArgName is used to pass in cluster secret name.
	CfgSecretNameSpaceArgName = "secret-namespace"
	// ProviderAttName attribute name for Volume.Provider
	ProviderAttName = "Provider"
	// ProviderTypeAttName attribute name for Volume.ProviderType
	ProviderTypeAttName = "ProviderType"
	// SnapshotSpaceAttName attribute name for Volume.SnapshotSpace
	SnapshotSpaceAttName = "SnapshotSpace"
	// TierAttName attribute name for Volume.Tier
	TierAttName = "Tier"
	// BillingTypeAttName attribute name for Volume.BillingType
	BillingTypeAttName = "BillingType"
	// RegionAttName attribute name for Volume.Region
	RegionAttName = "Region"
	// LunIDAttName attribute name for Volume.LunID
	LunIDAttName = "LunID"
	// TargetIPsAttName attribute name for Volume.TargetIPAddresses
	TargetIPsAttName = "TargetIPAddresses"
	// SoftlayerFileAttName attribute name to enable softlayer file support
	SoftlayerFileAttName = "SoftlayerFileEnabled"
)
