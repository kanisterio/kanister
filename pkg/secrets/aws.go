package secrets

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/config/aws"
	"github.com/pkg/errors"
)

const (
	// AWSSecretType represents the secret type for AWS credentials.
	AWSSecretType string = "secrets.kanister.io/aws"

	// AWSAccessKeyID is the key for AWS access key ID.
	AWSAccessKeyID string = "access_key_id"
	// AWSSecretAccessKey is the key for AWS secret access key.
	AWSSecretAccessKey string = "secret_access_key"
	// AWSSessionToken is the key for optional AWS session token.
	AWSSessionToken string = "session_token"
	// ConfigRole represents the key for the ARN of the role which can be assumed.
	// It is optional.
	ConfigRole         = "role"
	assumeRoleDuration = 25 * time.Minute
)

// ValidateAWSCredentials validates secret has all necessary information
// for AWS credentials. It also checks the secret doesn't have unnnecessary
// information.
//
// Required fields:
// - access_key_id
// - secret_access_key
//
// Optional field:
// - session_token
func ValidateAWSCredentials(secret *v1.Secret) error {
	if string(secret.Type) != AWSSecretType {
		return errors.New("Secret is not AWS secret")
	}
	if _, ok := secret.Data[AWSAccessKeyID]; !ok {
		return errors.New("awsAccessKeyID is a required field")
	}
	if _, ok := secret.Data[AWSSecretAccessKey]; !ok {
		return errors.New("awsSecretAccessKey is a required field")
	}
	count := 2
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
// If the required types are not avaialable in the secrets, it returns an errror.
func ExtractAWSCredentials(ctx context.Context, secret *v1.Secret) (*credentials.Value, error) {
	if err := ValidateAWSCredentials(secret); err != nil {
		return nil, err
	}
	accessKeyID := string(secret.Data[AWSAccessKeyID])
	secretAccessKey := string(secret.Data[AWSSecretAccessKey])
	role := string(secret.Data[ConfigRole])
	if role != "" {
		creds, err := aws.SwitchRole(ctx, accessKeyID, secretAccessKey, role, assumeRoleDuration)
		if err != nil {
			return nil, err
		}
		val, err := creds.Get()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get AWS credentials")
		}
		return &val, nil
	}
	return &credentials.Value{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    "",
	}, nil
}
