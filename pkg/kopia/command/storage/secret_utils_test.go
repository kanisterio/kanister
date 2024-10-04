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
	"testing"
	"time"

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

func Test(t *testing.T) { check.TestingT(t) }

type StorageUtilsSuite struct{}

var _ = check.Suite(&StorageUtilsSuite{})

func (s *StorageUtilsSuite) TestLocationUtils(c *check.C) {
	loc := map[string][]byte{
		repositoryserver.BucketKey:        []byte("test-key"),
		repositoryserver.EndpointKey:      []byte("test-endpoint"),
		repositoryserver.PrefixKey:        []byte("test-prefix"),
		repositoryserver.RegionKey:        []byte("test-region"),
		repositoryserver.SkipSSLVerifyKey: []byte("true"),
	}
	for _, tc := range []struct {
		LocType                    string
		expectedLocType            repositoryserver.LocType
		skipSSLVerify              string
		expectedSkipSSLVerifyValue bool
	}{
		{
			LocType:                    "s3",
			expectedLocType:            repositoryserver.LocTypeS3,
			skipSSLVerify:              "true",
			expectedSkipSSLVerifyValue: true,
		},
		{
			LocType:                    "gcs",
			expectedLocType:            repositoryserver.LocTypeGCS,
			skipSSLVerify:              "false",
			expectedSkipSSLVerifyValue: false,
		},
		{
			LocType:                    "azure",
			expectedLocType:            repositoryserver.LocTypeAzure,
			skipSSLVerify:              "true",
			expectedSkipSSLVerifyValue: true,
		},
	} {
		loc[repositoryserver.TypeKey] = []byte(tc.LocType)
		loc[repositoryserver.SkipSSLVerifyKey] = []byte(tc.skipSSLVerify)
		c.Assert(getBucketNameFromMap(loc), check.Equals, string(loc[repositoryserver.BucketKey]))
		c.Assert(getEndpointFromMap(loc), check.Equals, string(loc[repositoryserver.EndpointKey]))
		c.Assert(getPrefixFromMap(loc), check.Equals, string(loc[repositoryserver.PrefixKey]))
		c.Assert(getRegionFromMap(loc), check.Equals, string(loc[repositoryserver.RegionKey]))
		c.Assert(checkSkipSSLVerifyFromMap(loc), check.Equals, tc.expectedSkipSSLVerifyValue)
		c.Assert(locationType(loc), check.Equals, tc.expectedLocType)
	}
}

func (s *StorageUtilsSuite) TestGenerateEnvSpecFromCredentialSecret(c *check.C) {
	awsAccessKeyID := "access-key-id"
	awsSecretAccessKey := "secret-access-key"

	azureStorageAccountID := "azure-storage-account-id"
	azureStorageAccountKey := "azure-storage-account-key"
	azureStorageEnvironment := "AZURECLOUD"

	locSecretName := "test-secret"
	for _, tc := range []struct {
		secret          *corev1.Secret
		expectedEnvVars []corev1.EnvVar
		check.Checker
	}{
		{
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: locSecretName,
				},
				Type: corev1.SecretType(secrets.AWSSecretType),
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte(awsAccessKeyID),
					secrets.AWSSecretAccessKey: []byte(awsSecretAccessKey),
				},
			},
			expectedEnvVars: []corev1.EnvVar{
				{
					Name: aws.AccessKeyID,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: locSecretName,
							},
							Key: secrets.AWSAccessKeyID,
						},
					},
				},
				{
					Name: aws.SecretAccessKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: locSecretName,
							},
							Key: secrets.AWSSecretAccessKey,
						},
					},
				},
			},
			Checker: check.IsNil,
		},
		{
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: locSecretName,
				},
				Type: corev1.SecretType(secrets.AzureSecretType),
				Data: map[string][]byte{
					secrets.AzureStorageAccountID:   []byte(azureStorageAccountID),
					secrets.AzureStorageAccountKey:  []byte(azureStorageAccountKey),
					secrets.AzureStorageEnvironment: []byte(azureStorageEnvironment),
				},
			},
			expectedEnvVars: []corev1.EnvVar{
				{
					Name: azureStorageAccountEnv,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: locSecretName,
							},
							Key: secrets.AzureStorageAccountID,
						},
					},
				},
				{
					Name: azureStorageKeyEnv,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: locSecretName,
							},
							Key: secrets.AzureStorageAccountKey,
						},
					},
				},
				{
					Name:  azureStorageDomainEnv,
					Value: "blob.core.windows.net",
				},
			},
			Checker: check.IsNil,
		},
		{
			secret:  nil,
			Checker: check.NotNil,
		},
		{
			secret: &corev1.Secret{
				Type: "Opaque",
			},
			Checker:         check.IsNil,
			expectedEnvVars: nil,
		},
	} {
		envVars, err := GenerateEnvSpecFromCredentialSecret(tc.secret, time.Duration(0))
		c.Assert(err, tc.Checker)
		c.Assert(envVars, check.DeepEquals, tc.expectedEnvVars)
	}
}

func (s *StorageUtilsSuite) TestGetMapForLocationValues(c *check.C) {
	prefixValue := "test-prefix"
	regionValue := "test-region"
	bucketValue := "test-bucket"
	endpointValue := "test-endpoint"
	skipSSLVerifyValue := "true"
	for _, tc := range []struct {
		locType        repositoryserver.LocType
		prefix         string
		region         string
		bucket         string
		endpoint       string
		skipSSLVerify  string
		expectedOutput map[string][]byte
	}{
		{
			locType: repositoryserver.LocTypeS3,
			expectedOutput: map[string][]byte{
				repositoryserver.TypeKey: []byte(repositoryserver.LocTypeS3),
			},
		},
		{
			locType: repositoryserver.LocTypeS3,
			prefix:  prefixValue,
			expectedOutput: map[string][]byte{
				repositoryserver.TypeKey:   []byte(repositoryserver.LocTypeS3),
				repositoryserver.PrefixKey: []byte(prefixValue),
			},
		},
		{
			locType: repositoryserver.LocTypeS3,
			prefix:  prefixValue,
			region:  regionValue,
			expectedOutput: map[string][]byte{
				repositoryserver.TypeKey:   []byte(repositoryserver.LocTypeS3),
				repositoryserver.PrefixKey: []byte(prefixValue),
				repositoryserver.RegionKey: []byte(regionValue),
			},
		},
		{
			locType: repositoryserver.LocTypeS3,
			prefix:  prefixValue,
			region:  regionValue,
			bucket:  bucketValue,
			expectedOutput: map[string][]byte{
				repositoryserver.TypeKey:   []byte(repositoryserver.LocTypeS3),
				repositoryserver.PrefixKey: []byte(prefixValue),
				repositoryserver.RegionKey: []byte(regionValue),
				repositoryserver.BucketKey: []byte(bucketValue),
			},
		},
		{
			locType:  repositoryserver.LocTypeS3,
			prefix:   prefixValue,
			region:   regionValue,
			bucket:   bucketValue,
			endpoint: endpointValue,
			expectedOutput: map[string][]byte{
				repositoryserver.TypeKey:     []byte(repositoryserver.LocTypeS3),
				repositoryserver.PrefixKey:   []byte(prefixValue),
				repositoryserver.RegionKey:   []byte(regionValue),
				repositoryserver.BucketKey:   []byte(bucketValue),
				repositoryserver.EndpointKey: []byte(endpointValue),
			},
		},
		{
			locType:       repositoryserver.LocTypeS3,
			prefix:        prefixValue,
			region:        regionValue,
			bucket:        bucketValue,
			endpoint:      endpointValue,
			skipSSLVerify: skipSSLVerifyValue,
			expectedOutput: map[string][]byte{
				repositoryserver.TypeKey:          []byte(repositoryserver.LocTypeS3),
				repositoryserver.PrefixKey:        []byte(prefixValue),
				repositoryserver.RegionKey:        []byte(regionValue),
				repositoryserver.BucketKey:        []byte(bucketValue),
				repositoryserver.EndpointKey:      []byte(endpointValue),
				repositoryserver.SkipSSLVerifyKey: []byte(skipSSLVerifyValue),
			},
		},
		{
			locType:       repositoryserver.LocType(crv1alpha1.LocationTypeS3Compliant),
			prefix:        prefixValue,
			region:        regionValue,
			bucket:        bucketValue,
			endpoint:      endpointValue,
			skipSSLVerify: skipSSLVerifyValue,
			expectedOutput: map[string][]byte{
				repositoryserver.TypeKey:          []byte(repositoryserver.LocTypeS3),
				repositoryserver.PrefixKey:        []byte(prefixValue),
				repositoryserver.RegionKey:        []byte(regionValue),
				repositoryserver.BucketKey:        []byte(bucketValue),
				repositoryserver.EndpointKey:      []byte(endpointValue),
				repositoryserver.SkipSSLVerifyKey: []byte(skipSSLVerifyValue),
			},
		},
	} {
		op := GetMapForLocationValues(
			tc.locType,
			tc.prefix,
			tc.region,
			tc.bucket,
			tc.endpoint,
			tc.skipSSLVerify,
		)
		c.Assert(op, check.DeepEquals, tc.expectedOutput)
	}
}
