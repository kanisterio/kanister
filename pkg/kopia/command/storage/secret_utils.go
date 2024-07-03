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

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage/azure"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

const (
	// Azure location related environment variables
	azureStorageAccountEnv = "AZURE_STORAGE_ACCOUNT"
	azureStorageKeyEnv     = "AZURE_STORAGE_KEY"
	azureStorageDomainEnv  = "AZURE_STORAGE_DOMAIN"
)

func getBucketNameFromMap(m map[string][]byte) string {
	return string(m[repositoryserver.BucketKey])
}

func getEndpointFromMap(m map[string][]byte) string {
	return string(m[repositoryserver.EndpointKey])
}

func getPrefixFromMap(m map[string][]byte) string {
	return string(m[repositoryserver.PrefixKey])
}

func getRegionFromMap(m map[string][]byte) string {
	return string(m[repositoryserver.RegionKey])
}

func checkSkipSSLVerifyFromMap(m map[string][]byte) bool {
	v := string(m[repositoryserver.SkipSSLVerifyKey])
	return v == "true"
}

func locationType(m map[string][]byte) repositoryserver.LocType {
	return repositoryserver.LocType(m[repositoryserver.TypeKey])
}

// GenerateEnvSpecFromCredentialSecret parses the secret and returns
// list of EnvVar based on secret type
func GenerateEnvSpecFromCredentialSecret(s *corev1.Secret, assumeRoleDurationS3 time.Duration) ([]corev1.EnvVar, error) {
	if s == nil {
		return nil, errkit.New("Secret cannot be nil")
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

func getEnvSpecForAWSCredentialSecret(s *corev1.Secret, assumeRoleDuration time.Duration) ([]corev1.EnvVar, error) {
	var err error
	envVars := []corev1.EnvVar{}
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

func getEnvSpecForAzureCredentialSecret(s *corev1.Secret) ([]corev1.EnvVar, error) {
	envVars := []corev1.EnvVar{}
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
			return nil, errkit.Wrap(err, "Failed to get azure environment from name", "storageEnv", storageEnv)
		}
		blobDomain := "blob." + env.StorageEndpointSuffix
		// TODO : Check how we can set this env to use value from secret
		envVars = append(envVars, getEnvVar(azureStorageDomainEnv, blobDomain))
	}
	return envVars, nil
}

func getEnvVarWithSecretRef(varName, secretName, secretKey string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: varName,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				Key: secretKey,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
			},
		},
	}
}

func getEnvVar(varName, value string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  varName,
		Value: value,
	}
}

// GetMapForLocationValues return a map with valid keys
// for different location values
func GetMapForLocationValues(
	locType repositoryserver.LocType,
	prefix,
	region,
	bucket,
	endpoint,
	skipSSLVerify string,
) map[string][]byte {
	m := map[string][]byte{}
	if bucket != "" {
		m[repositoryserver.BucketKey] = []byte(bucket)
	}
	if endpoint != "" {
		m[repositoryserver.EndpointKey] = []byte(endpoint)
	}
	if prefix != "" {
		m[repositoryserver.PrefixKey] = []byte(prefix)
	}
	if region != "" {
		m[repositoryserver.RegionKey] = []byte(region)
	}
	if skipSSLVerify != "" {
		m[repositoryserver.SkipSSLVerifyKey] = []byte(skipSSLVerify)
	}
	if locType != "" {
		m[repositoryserver.TypeKey] = []byte(locType)
		if locType == repositoryserver.LocType(crv1alpha1.LocationTypeS3Compliant) {
			m[repositoryserver.TypeKey] = []byte(repositoryserver.LocTypeS3)
		}
	}
	return m
}
