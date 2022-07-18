// Copyright 2022 The Kanister Authors.
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

package kopia

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

const (
	LocationTypeAzure   v1alpha1.LocationType   = "AZ"
	LocationTypeGCS     v1alpha1.LocationType   = "GCS"
	LocationTypeS3      v1alpha1.LocationType   = "S3"
	SecretTypeKeyPair   v1alpha1.CredentialType = "KeyPair"
	SecretTypeK8sSecret v1alpha1.CredentialType = "Secret"
)

// Profile interface returns information and credentials for Object store or File store locations
type Profile interface {
	// Location related
	BucketName() string
	Endpoint() string
	LocationType() (v1alpha1.LocationType, error)
	Prefix() string
	SkipSSLVerification() bool
	Region() string
	// Credential related
	AccessKeyID() string
	CredType() (v1alpha1.CredentialType, error)
	Secret() *corev1.Secret
	SecretAccessKey() string
	StorageAccount() string
	StorageKey() string
	// Azure Only Method
	StorageDomain() string
}
