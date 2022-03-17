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

package objectstore

// ProviderConfig describes the config for the object store (which provider to use)
type ProviderConfig struct {
	// object store type
	Type ProviderType
	// Endpoint used to access the object store. It can be implicit for
	// stores from certain cloud providers such as AWS. In that case it can
	// be empty
	Endpoint string
	// Region specifies the region of the object store.
	Region string
	// If true, disable SSL verification. If false (the default), SSL
	// verification is enabled.
	SkipSSLVerify bool
}

// SecretAws AWS keys
type SecretAws struct {
	// access key Id
	AccessKeyID string
	// secret access key
	SecretAccessKey string
	// session token
	SessionToken string
}

// SecretAzure Azure credentials
type SecretAzure struct {
	// storage account
	StorageAccount string
	// storage key
	StorageKey string
	// environment name
	EnvironmentName string
}

// SecretGcp GCP credentials
type SecretGcp struct {
	// project Id
	ProjectID string
	// base64 encoded service account key
	ServiceKey string
}

// Secret contains the credentials for different providers
type Secret struct {
	// aws
	Aws *SecretAws
	// azure
	Azure *SecretAzure
	// gcp
	Gcp *SecretGcp
	// type
	Type SecretType
}
