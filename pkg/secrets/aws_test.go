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
	"context"

	"github.com/aws/aws-sdk-go/aws/credentials"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/config"
)

type AWSSecretSuite struct{}

var _ = Suite(&AWSSecretSuite{})

func (s *AWSSecretSuite) TestExtractAWSCredentials(c *C) {
	tcs := []struct {
		secret     *corev1.Secret
		expected   *credentials.Value
		errChecker Checker
	}{
		{
			secret: &corev1.Secret{
				Type: corev1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte("key_id"),
					AWSSecretAccessKey: []byte("secret_key"),
				},
			},
			expected: &credentials.Value{
				AccessKeyID:     "key_id",
				SecretAccessKey: "secret_key",
				ProviderName:    credentials.StaticProviderName,
			},
			errChecker: IsNil,
		},
		{
			secret: &corev1.Secret{
				Type: "Opaque",
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{
			secret: &corev1.Secret{
				Type: corev1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSSecretAccessKey: []byte("secret_key"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{
			secret: &corev1.Secret{
				Type: corev1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID: []byte("key_id"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{
			secret: &corev1.Secret{
				Type: corev1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte("key_id"),
					AWSSecretAccessKey: []byte("secret_key"),
					"ExtraField":       []byte("extra_value"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
	}
	for testNum, tc := range tcs {
		creds, err := ExtractAWSCredentials(context.Background(), tc.secret, aws.AssumeRoleDurationDefault)
		c.Check(creds, DeepEquals, tc.expected, Commentf("test number: %d", testNum))
		c.Check(err, tc.errChecker)
	}
}

func (s *AWSSecretSuite) TestExtractAWSCredentialsWithSessionToken(c *C) {
	for _, tc := range []struct {
		secret *corev1.Secret
		output Checker
	}{
		{
			secret: &corev1.Secret{
				Type: corev1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte(config.GetEnvOrSkip(c, "AWS_ACCESS_KEY_ID")),
					AWSSecretAccessKey: []byte(config.GetEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")),
					ConfigRole:         []byte(config.GetEnvOrSkip(c, "role")),
				},
			},
			output: IsNil,
		},
		{
			secret: &corev1.Secret{
				Type: corev1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte(config.GetEnvOrSkip(c, "AWS_ACCESS_KEY_ID")),
					AWSSecretAccessKey: []byte(config.GetEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")),
					ConfigRole:         []byte("arn:aws:iam::000000000000:role/test-fake-role"),
				},
			},
			output: NotNil,
		},
	} {
		_, err := ExtractAWSCredentials(context.Background(), tc.secret, aws.AssumeRoleDurationDefault)
		c.Check(err, tc.output)
	}
}
