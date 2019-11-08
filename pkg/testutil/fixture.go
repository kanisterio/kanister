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

package testutil

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	awsconfig "github.com/kanisterio/kanister/pkg/config/aws"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	TestS3BucketName = "tests.kanister.io"
	TestS3Region     = "us-west-2"
)

func ObjectStoreProfileOrSkip(c *check.C, osType objectstore.ProviderType, location crv1alpha1.Location) *param.Profile {
	var key, val string
	switch osType {
	case objectstore.ProviderTypeS3:
		key = GetEnvOrSkip(c, awsconfig.AccessKeyID)
		val = GetEnvOrSkip(c, awsconfig.SecretAccessKey)
		if role, ok := os.LookupEnv(awsconfig.ConfigRole); ok {
			return s3ProfileWithSecretCredential(location, key, val, role)
		}
	case objectstore.ProviderTypeGCS:
		GetEnvOrSkip(c, blockstorage.GoogleCloudCreds)
		creds, err := google.FindDefaultCredentials(context.Background(), compute.ComputeScope)
		c.Check(err, check.IsNil)
		key = creds.ProjectID
		val = string(creds.JSON)
	case objectstore.ProviderTypeAzure:
		key = GetEnvOrSkip(c, blockstorage.AzureStorageAccount)
		val = GetEnvOrSkip(c, blockstorage.AzureStorageKey)
	}
	return &param.Profile{
		Location: location,
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     key,
				Secret: val,
			},
		},
	}
}

func GetEnvOrSkip(c *check.C, varName string) string {
	v := os.Getenv(varName)
	if v == "" {
		reason := fmt.Sprintf("Test %s requires the environemnt variable '%s'", c.TestName(), varName)
		c.Skip(reason)
	}
	return v
}

func s3ProfileWithSecretCredential(location crv1alpha1.Location, accessKeyID, secretAccessKey, role string) *param.Profile {
	return &param.Profile{
		Location: location,
		Credential: param.Credential{
			Type: param.CredentialTypeSecret,
			Secret: &v1.Secret{
				Type: "secrets.kanister.io/aws",
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte(accessKeyID),
					secrets.AWSSecretAccessKey: []byte(secretAccessKey),
					secrets.ConfigRole:         []byte(role),
				},
			},
		},
	}
}
