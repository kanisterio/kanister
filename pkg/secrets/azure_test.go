// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secrets

import (
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/objectstore"
)

type AzureSecretSuite struct{}

var _ = Suite(&AzureSecretSuite{})

func (s *AzureSecretSuite) TestExtractAzureCredentials(c *C) {
	for i, tc := range []struct {
		secret     *corev1.Secret
		expected   *objectstore.SecretAzure
		errChecker Checker
	}{
		{
			secret: &corev1.Secret{
				Type: corev1.SecretType(AzureSecretType),
				Data: map[string][]byte{
					AzureStorageAccountID:   []byte("key_id"),
					AzureStorageAccountKey:  []byte("secret_key"),
					AzureStorageEnvironment: []byte("env"),
				},
			},
			expected: &objectstore.SecretAzure{
				StorageAccount:  "key_id",
				StorageKey:      "secret_key",
				EnvironmentName: "env",
			},
			errChecker: IsNil,
		},
		{ // bad type
			secret: &corev1.Secret{
				Type: corev1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AzureStorageAccountID:   []byte("key_id"),
					AzureStorageAccountKey:  []byte("secret_key"),
					AzureStorageEnvironment: []byte("env"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{ // missing field
			secret: &corev1.Secret{
				Type: corev1.SecretType(AzureSecretType),
				Data: map[string][]byte{
					AzureStorageAccountID:   []byte("key_id"),
					AzureStorageEnvironment: []byte("env"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{ // additional field
			secret: &corev1.Secret{
				Type: corev1.SecretType(AzureSecretType),
				Data: map[string][]byte{
					AzureStorageAccountID:   []byte("key_id"),
					AzureStorageAccountKey:  []byte("secret_key"),
					AzureStorageEnvironment: []byte("env"),
					"bad field":             []byte("bad"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
	} {
		azsecret, err := ExtractAzureCredentials(tc.secret)
		c.Check(azsecret, DeepEquals, tc.expected, Commentf("test number: %d", i))
		c.Check(err, tc.errChecker)
	}
}
