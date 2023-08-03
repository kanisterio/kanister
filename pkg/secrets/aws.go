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
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
)

const (
	// AWSSecretType represents the secret type for AWS credentials.
	AWSSecretType string = "secrets.kanister.io/aws"

	// AWSAccessKeyID is the key for AWS access key ID.
	AWSAccessKeyID string = "aws_access_key_id"
	// AWSSecretAccessKey is the key for AWS secret access key.
	AWSSecretAccessKey string = "aws_secret_access_key"
	// AWSSessionToken is the key for AWS Session token
	AWSSessionToken string = "aws_session_token"
	// ConfigRole represents the key for the ARN of the role which can be assumed.
	// It is optional.
	ConfigRole = "role"
)

// ValidateAWSCredentials validates secret has all necessary information
// for AWS credentials. It also checks the secret doesn't have unnecessary
// information.
//
// Required fields:
// - access_key_id
// - secret_access_key
//
// Optional field:
// - session_token
func ValidateAWSCredentials(secret *corev1.Secret) error {
	if string(secret.Type) != AWSSecretType {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.IncompatibleSecretTypeErrorMsg, AWSSecretType, secret.Namespace, secret.Name)
	}
	count := 0
	if _, ok := secret.Data[AWSAccessKeyID]; ok {
		count++
	}
	if _, ok := secret.Data[AWSSecretAccessKey]; ok {
		count++
	}
	if _, ok := secret.Data[ConfigRole]; ok {
		count++
	}
	if len(secret.Data) > count {
		return errors.New("Secret has an unknown field")
	}
	return nil
}

// ExtractAWSCredentials extracts AWS credential values from the given secret.
//
// Extracted values from the secrets are:
// - access_key_id (required)
// - secret_access_key (required)
// - role (optional)
//
// If the type of the secret is not "secret.kanister.io/aws", it returns an error.
// If the required types are not available in the secrets, it returns an error.
//
// ExtractAWSCredentials accepts an assumeRoleDuration which is used to set
// the duration of the AWS session token.
// When this setting is not provided, the default duration of a token is 1h.
// The minimum value allowed is 15 minutes (15m).
// The maximum value depends on the max duration setting
// of the IAM role - The setting can be viewed using instructions here
// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html#id_roles_use_view-role-max-session.
// The IAM role's max duration setting can be modified between 1h to 12h.
func ExtractAWSCredentials(ctx context.Context, secret *corev1.Secret, assumeRoleDuration time.Duration) (*credentials.Value, error) {
	if err := ValidateAWSCredentials(secret); err != nil {
		return nil, err
	}
	config := map[string]string{
		aws.AccessKeyID:        string(secret.Data[AWSAccessKeyID]),
		aws.SecretAccessKey:    string(secret.Data[AWSSecretAccessKey]),
		aws.ConfigRole:         string(secret.Data[ConfigRole]),
		aws.AssumeRoleDuration: assumeRoleDuration.String(),
	}
	creds, err := aws.GetCredentials(ctx, config)
	if err != nil {
		return nil, err
	}
	val, err := creds.Get()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AWS credentials")
	}
	exp, err := creds.ExpiresAt()
	if err == nil {
		log.Debug().Print("Credential expiration", field.M{"expirationTime": exp})
	}
	return &val, nil
}
