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
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	apiconfig "github.com/kanisterio/kanister/pkg/apis/config/v1alpha1"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	locationTypeAzure   apiconfig.LocationType = "AZ"
	locationTypeGCS     apiconfig.LocationType = "GCS"
	locationTypeS3      apiconfig.LocationType = "S3"
	secretTypeKeyPair   apiconfig.SecretType   = "KeyPair"
	secretTypeK8sSecret apiconfig.SecretType   = "Secret"
)

// profile interface returns information and credentials for Object store or File store locations
type profile interface {
	// Location related
	bucketName() string
	endpoint() string
	locationType() (apiconfig.LocationType, error)
	prefix() string
	skipSSLVerify() bool
	region() string
	// Credential related
	accessKeyID() string
	credType() (apiconfig.SecretType, error)
	secret() *corev1.Secret
	secretAccessKey() string
	storageAccount() string
	storageKey() string

	// Azure Only Method
	storageDomain() string
}

var (
	_ profile = (*KanisterProfile)(nil)
)

// KanisterProfile is a wrapper around Kanister `param.Profile` type
type KanisterProfile struct {
	*param.Profile
}

func (p *KanisterProfile) bucketName() string {
	return p.Location.Bucket
}

func (p *KanisterProfile) endpoint() string {
	return p.Location.Endpoint
}

func (p *KanisterProfile) locationType() (apiconfig.LocationType, error) {
	switch p.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		return locationTypeS3, nil
	case crv1alpha1.LocationTypeGCS:
		return locationTypeGCS, nil
	case crv1alpha1.LocationTypeAzure:
		return locationTypeAzure, nil
	default:
		return "", errors.New(fmt.Sprintf("Unsupported type for the location %s", p.Location.Type))
	}
}

func (p *KanisterProfile) prefix() string {
	return p.Location.Prefix
}

func (p *KanisterProfile) skipSSLVerify() bool {
	return p.SkipSSLVerify
}

func (p *KanisterProfile) accessKeyID() string {
	return p.Credential.KeyPair.ID
}

func (p *KanisterProfile) credType() (apiconfig.SecretType, error) {
	creds := p.Credential
	switch creds.Type {
	case param.CredentialTypeSecret:
		return secretTypeK8sSecret, nil
	case param.CredentialTypeKeyPair:
		return secretTypeKeyPair, nil
	default:
		return "", errors.New(fmt.Sprintf("Unsupported type for credentials %s", creds.Type))
	}
}

// This is only called when credential type is a secret
func (p *KanisterProfile) secret() *corev1.Secret {
	return p.Credential.Secret
}

func (p *KanisterProfile) secretAccessKey() string {
	return p.Credential.KeyPair.Secret
}

func (p *KanisterProfile) region() string {
	return p.Location.Region
}

func (p *KanisterProfile) storageAccount() string {
	return p.Credential.KeyPair.ID
}

func (p *KanisterProfile) storageKey() string {
	return p.Credential.KeyPair.Secret
}

func (p *KanisterProfile) storageDomain() string {
	// This function is only called when Key Pair credential type is used.
	// Key pair credential types will not have storage domain (required for other az environments)
	// To use other environments we need to use a secret credential type.
	return ""
}
