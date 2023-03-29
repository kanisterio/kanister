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

package storage

import (
	"context"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/secrets"
)

type LocType string

const (
	// Location secret keys
	bucketKey        = "bucket"
	endpointKey      = "endpoint"
	prefixKey        = "prefix"
	regionKey        = "region"
	skipSSLVerifyKey = "skipSSLVerify"
	typeKey          = "type"

	// Location types
	LocTypeS3        LocType = "s3"
	LocTypeGCS       LocType = "gcs"
	LocTypeAzure     LocType = "azure"
	LocTypeFilestore LocType = "filestore"

	// Azure location related environment variables
	azureStorageAccountEnv = "AZURE_STORAGE_ACCOUNT"
	azureStorageKeyEnv     = "AZURE_STORAGE_KEY"
	azureStorageDomainEnv  = "AZURE_STORAGE_DOMAIN"
)

func getBucketNameFromMap(m map[string][]byte) string {
	return string(m[bucketKey])
}

func getEndpointFromMap(m map[string][]byte) string {
	return string(m[endpointKey])
}

func getPrefixFromMap(m map[string][]byte) string {
	return string(m[prefixKey])
}

func getRegionFromMap(m map[string][]byte) string {
	return string(m[regionKey])
}

func checkSkipSSLVerifyFromMap(m map[string][]byte) bool {
	v := string(m[skipSSLVerifyKey])
	return v == "true"
}

func locationType(m map[string][]byte) LocType {
	return LocType(m[typeKey])
}

// GenerateEnvSpecFromCredentialSecret parses the secret and returns
// list of EnvVar based on secret type
func GenerateEnvSpecFromCredentialSecret(s *v1.Secret, assumeRoleDurationS3 time.Duration) ([]v1.EnvVar, error) {
	if s == nil {
		return nil, errors.New("Secret cannot be nil")
	}
	secType := string(s.Type)
	switch secType {
	case secrets.AWSSecretType:
		return getEnvSpecForAWSCredentialSecret(s, assumeRoleDurationS3)
	case secrets.AzureSecretType:
		return getEnvSpecForAzureCredentialSecret(s)
	}
	// We only need to set the environment variables in cases where
	// secret type is AWS or Azure.
	return nil, nil
}

func getEnvSpecForAWSCredentialSecret(s *v1.Secret, assumeRoleDuration time.Duration) ([]v1.EnvVar, error) {
	var err error
	envVars := []v1.EnvVar{}
	envVars = append(
		envVars,
		getEnvVarWithSecretRef(aws.AccessKeyID, s.Name, secrets.AWSAccessKeyID),
		getEnvVarWithSecretRef(aws.SecretAccessKey, s.Name, secrets.AWSSecretAccessKey),
	)
	creds, err := secrets.ExtractAWSCredentials(context.Background(), s, assumeRoleDuration)
	if err != nil {
		return nil, err
	}
	if creds.SessionToken != "" {
		envVars = append(envVars, getEnvVar(aws.SessionToken, creds.SessionToken))
	}
	return envVars, nil
}

func getEnvSpecForAzureCredentialSecret(s *v1.Secret) ([]v1.EnvVar, error) {
	envVars := []v1.EnvVar{}
	envVars = append(
		envVars,
		getEnvVarWithSecretRef(azureStorageAccountEnv, s.Name, secrets.AzureStorageAccountID),
		getEnvVarWithSecretRef(azureStorageKeyEnv, s.Name, secrets.AzureStorageAccountKey),
	)
	azureSecret, err := secrets.ExtractAzureCredentials(s)
	if err != nil {
		return nil, err
	}
	storageEnv := azureSecret.EnvironmentName
	if storageEnv != "" {
		env, err := azure.EnvironmentFromName(storageEnv)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get azure environment from name: %s", storageEnv)
		}
		blobDomain := "blob." + env.StorageEndpointSuffix
		// TODO : Check how we can set this env to use value from secret
		envVars = append(envVars, getEnvVar(azureStorageDomainEnv, blobDomain))
	}
	return envVars, nil
}

func getEnvVarWithSecretRef(varName, secretName, secretKey string) v1.EnvVar {
	return v1.EnvVar{
		Name: varName,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				Key: secretKey,
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
			},
		},
	}
}

func getEnvVar(varName, value string) v1.EnvVar {
	return v1.EnvVar{
		Name:  varName,
		Value: value,
	}
}

// GetMapForLocationValues return a map with valid keys
// for different location values
func GetMapForLocationValues(
	locType LocType,
	prefix,
	region,
	bucket,
	endpoint,
	skipSSLVerify string,
) map[string][]byte {
	m := map[string][]byte{}
	if bucket != "" {
		m[bucketKey] = []byte(bucket)
	}
	if endpoint != "" {
		m[endpointKey] = []byte(endpoint)
	}
	if prefix != "" {
		m[prefixKey] = []byte(prefix)
	}
	if region != "" {
		m[regionKey] = []byte(region)
	}
	if skipSSLVerify != "" {
		m[skipSSLVerifyKey] = []byte(skipSSLVerify)
	}
	if locType != "" {
		m[typeKey] = []byte(locType)
		if locType == LocType(v1alpha1.LocationTypeS3Compliant) {
			m[typeKey] = []byte(LocTypeS3)
		}
	}
	return m
}
