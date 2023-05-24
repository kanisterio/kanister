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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/secrets"
)

func Test(t *testing.T) { check.TestingT(t) }

type StorageUtilsSuite struct{}

var _ = check.Suite(&StorageUtilsSuite{})

func (s *StorageUtilsSuite) TestLocationUtils(c *check.C) {
	loc := map[string][]byte{
		secrets.BucketKey:        []byte("test-key"),
		secrets.EndpointKey:      []byte("test-endpoint"),
		secrets.PrefixKey:        []byte("test-prefix"),
		secrets.RegionKey:        []byte("test-region"),
		secrets.SkipSSLVerifyKey: []byte("true"),
	}
	for _, tc := range []struct {
		LocType                    string
		expectedLocType            secrets.LocType
		skipSSLVerify              string
		expectedSkipSSLVerifyValue bool
	}{
		{
			LocType:                    "s3",
			expectedLocType:            secrets.LocTypeS3,
			skipSSLVerify:              "true",
			expectedSkipSSLVerifyValue: true,
		},
		{
			LocType:                    "gcs",
			expectedLocType:            secrets.LocTypeGCS,
			skipSSLVerify:              "false",
			expectedSkipSSLVerifyValue: false,
		},
		{
			LocType:                    "azure",
			expectedLocType:            secrets.LocTypeAzure,
			skipSSLVerify:              "true",
			expectedSkipSSLVerifyValue: true,
		},
	} {
		loc[secrets.TypeKey] = []byte(tc.LocType)
		loc[secrets.SkipSSLVerifyKey] = []byte(tc.skipSSLVerify)
		c.Assert(getBucketNameFromMap(loc), check.Equals, string(loc[secrets.BucketKey]))
		c.Assert(getEndpointFromMap(loc), check.Equals, string(loc[secrets.EndpointKey]))
		c.Assert(getPrefixFromMap(loc), check.Equals, string(loc[secrets.PrefixKey]))
		c.Assert(getRegionFromMap(loc), check.Equals, string(loc[secrets.RegionKey]))
		c.Assert(checkSkipSSLVerifyFromMap(loc), check.Equals, tc.expectedSkipSSLVerifyValue)
		c.Assert(locationType(loc), check.Equals, tc.expectedLocType)
	}
}

func (s *StorageUtilsSuite) TestGenerateEnvSpecFromCredentialSecret(c *check.C) {
	awsAccessKeyId := "access-key-id"
	awsSecretAccessKey := "secret-access-key"

	azureStorageAccountID := "azure-storage-account-id"
	azureStorageAccountKey := "azure-storage-account-key"
	azureStorageEnvironment := "AZURECLOUD"

	locSecretName := "test-secret"
	for _, tc := range []struct {
		secret          *v1.Secret
		expectedEnvVars []v1.EnvVar
		check.Checker
	}{
		{
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: locSecretName,
				},
				Type: v1.SecretType(secrets.AWSSecretType),
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte(awsAccessKeyId),
					secrets.AWSSecretAccessKey: []byte(awsSecretAccessKey),
				},
			},
			expectedEnvVars: []v1.EnvVar{
				{
					Name: aws.AccessKeyID,
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: locSecretName,
							},
							Key: secrets.AWSAccessKeyID,
						},
					},
				},
				{
					Name: aws.SecretAccessKey,
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
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
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: locSecretName,
				},
				Type: v1.SecretType(secrets.AzureSecretType),
				Data: map[string][]byte{
					secrets.AzureStorageAccountID:   []byte(azureStorageAccountID),
					secrets.AzureStorageAccountKey:  []byte(azureStorageAccountKey),
					secrets.AzureStorageEnvironment: []byte(azureStorageEnvironment),
				},
			},
			expectedEnvVars: []v1.EnvVar{
				{
					Name: azureStorageAccountEnv,
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: locSecretName,
							},
							Key: secrets.AzureStorageAccountID,
						},
					},
				},
				{
					Name: azureStorageKeyEnv,
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
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
			secret: &v1.Secret{
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
		locType        secrets.LocType
		prefix         string
		region         string
		bucket         string
		endpoint       string
		skipSSLVerify  string
		expectedOutput map[string][]byte
	}{
		{
			locType: secrets.LocTypeS3,
			expectedOutput: map[string][]byte{
				secrets.TypeKey: []byte(secrets.LocTypeS3),
			},
		},
		{
			locType: secrets.LocTypeS3,
			prefix:  prefixValue,
			expectedOutput: map[string][]byte{
				secrets.TypeKey:   []byte(secrets.LocTypeS3),
				secrets.PrefixKey: []byte(prefixValue),
			},
		},
		{
			locType: secrets.LocTypeS3,
			prefix:  prefixValue,
			region:  regionValue,
			expectedOutput: map[string][]byte{
				secrets.TypeKey:   []byte(secrets.LocTypeS3),
				secrets.PrefixKey: []byte(prefixValue),
				secrets.RegionKey: []byte(regionValue),
			},
		},
		{
			locType: secrets.LocTypeS3,
			prefix:  prefixValue,
			region:  regionValue,
			bucket:  bucketValue,
			expectedOutput: map[string][]byte{
				secrets.TypeKey:   []byte(secrets.LocTypeS3),
				secrets.PrefixKey: []byte(prefixValue),
				secrets.RegionKey: []byte(regionValue),
				secrets.BucketKey: []byte(bucketValue),
			},
		},
		{
			locType:  secrets.LocTypeS3,
			prefix:   prefixValue,
			region:   regionValue,
			bucket:   bucketValue,
			endpoint: endpointValue,
			expectedOutput: map[string][]byte{
				secrets.TypeKey:     []byte(secrets.LocTypeS3),
				secrets.PrefixKey:   []byte(prefixValue),
				secrets.RegionKey:   []byte(regionValue),
				secrets.BucketKey:   []byte(bucketValue),
				secrets.EndpointKey: []byte(endpointValue),
			},
		},
		{
			locType:       secrets.LocTypeS3,
			prefix:        prefixValue,
			region:        regionValue,
			bucket:        bucketValue,
			endpoint:      endpointValue,
			skipSSLVerify: skipSSLVerifyValue,
			expectedOutput: map[string][]byte{
				secrets.TypeKey:          []byte(secrets.LocTypeS3),
				secrets.PrefixKey:        []byte(prefixValue),
				secrets.RegionKey:        []byte(regionValue),
				secrets.BucketKey:        []byte(bucketValue),
				secrets.EndpointKey:      []byte(endpointValue),
				secrets.SkipSSLVerifyKey: []byte(skipSSLVerifyValue),
			},
		},
		{
			locType:       secrets.LocType(v1alpha1.LocationTypeS3Compliant),
			prefix:        prefixValue,
			region:        regionValue,
			bucket:        bucketValue,
			endpoint:      endpointValue,
			skipSSLVerify: skipSSLVerifyValue,
			expectedOutput: map[string][]byte{
				secrets.TypeKey:          []byte(secrets.LocTypeS3),
				secrets.PrefixKey:        []byte(prefixValue),
				secrets.RegionKey:        []byte(regionValue),
				secrets.BucketKey:        []byte(bucketValue),
				secrets.EndpointKey:      []byte(endpointValue),
				secrets.SkipSSLVerifyKey: []byte(skipSSLVerifyValue),
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
