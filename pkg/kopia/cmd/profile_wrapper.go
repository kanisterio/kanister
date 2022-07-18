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

package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/param"
)

var (
	_ kopia.Profile = (*KanisterProfile)(nil)
)

// KanisterProfile is a wrapper around Kanister `param.Profile` type
type KanisterProfile struct {
	*param.Profile
}

func (p *KanisterProfile) BucketName() string {
	return p.Location.Bucket
}

func (p *KanisterProfile) Endpoint() string {
	return p.Location.Endpoint
}

func (p *KanisterProfile) LocationType() (crv1alpha1.LocationType, error) {
	switch p.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		return kopia.LocationTypeS3, nil
	case crv1alpha1.LocationTypeGCS:
		return kopia.LocationTypeGCS, nil
	case crv1alpha1.LocationTypeAzure:
		return kopia.LocationTypeAzure, nil
	default:
		return "", errors.New(fmt.Sprintf("Unsupported type for the location %s", p.Location.Type))
	}
}

func (p *KanisterProfile) Prefix() string {
	return p.Location.Prefix
}

func (p *KanisterProfile) SkipSSLVerification() bool {
	return p.SkipSSLVerify
}

func (p *KanisterProfile) AccessKeyID() string {
	return p.Credential.KeyPair.ID
}

func (p *KanisterProfile) CredType() (crv1alpha1.CredentialType, error) {
	creds := p.Credential
	switch creds.Type {
	case param.CredentialTypeSecret:
		return kopia.SecretTypeK8sSecret, nil
	case param.CredentialTypeKeyPair:
		return kopia.SecretTypeKeyPair, nil
	default:
		return "", errors.New(fmt.Sprintf("Unsupported type for credentials %s", creds.Type))
	}
}

// This is only called when credential type is a secret
func (p *KanisterProfile) Secret() *corev1.Secret {
	return p.Credential.Secret
}

func (p *KanisterProfile) SecretAccessKey() string {
	return p.Credential.KeyPair.Secret
}

func (p *KanisterProfile) Region() string {
	return p.Location.Region
}

func (p *KanisterProfile) StorageAccount() string {
	return p.Credential.KeyPair.ID
}

func (p *KanisterProfile) StorageKey() string {
	return p.Credential.KeyPair.Secret
}

func (p *KanisterProfile) StorageDomain() string {
	// This function is only called when Key Pair credential type is used.
	// Key pair credential types will not have storage domain (required for other az environments)
	// To use other enviornments we need to use a secret credential type.
	return ""
}
