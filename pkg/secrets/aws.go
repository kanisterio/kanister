package secrets

import (
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

const (
	// AWSSecretType represent the secret type for AWS credentials.
	AWSSecretType string = "secrets.kanister.io/aws"

	// AWSAccessKeyID is the key for AWS access key ID.
	AWSAccessKeyID string = "awsAccessKeyID"
	// AWSSecretAccessKey is the key for AWS secret access key.
	AWSSecretAccessKey string = "awsSecretAccessKey"
	// AWSSessionToken is the key for optional AWS session token.
	AWSSessionToken string = "awsSessionToken"
)

// ValidateAWSCredentials validates secret has all necessary information
// for AWS credentials. It also checks the secret doesn't have unnnecessary
// information.
//
// Required fields:
// - AWS_ACCESS_KEY_ID
// - AWS_SECRET_ACCESS_KEY
//
// Optional field:
// - AWS_SESSION_TOKEN
func ValidateAWSCredentials(secret *v1.Secret) error {
	if string(secret.Type) != AWSSecretType {
		return errors.New("Secret is not AWS secret")
	}
	if _, ok := secret.Data[AWSAccessKeyID]; !ok {
		return errors.New("AWS_ACCESS_KEY_ID is a required field")
	}
	if _, ok := secret.Data[AWSSecretAccessKey]; !ok {
		return errors.New("AWS_SECRET_ACCESS_KEY is a required field")
	}
	count := 2
	if _, ok := secret.Data[AWSSessionToken]; ok {
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
// - AWS_ACCESS_KEY_ID (required)
// - AWS_SECRET_ACCESS_KEY (required)
// - AWS_SESSION_TOKEN (optional)
//
// If the type of the secret is not "secret.kanister.io/aws", it returns an error.
// If the required types are not avaialable in the secrets, it returns an errror.
func ExtractAWSCredentials(secret *v1.Secret) (*credentials.Value, error) {
	if err := ValidateAWSCredentials(secret); err != nil {
		return nil, err
	}
	accessKeyID := secret.Data[AWSAccessKeyID]
	secretAccessKey := secret.Data[AWSSecretAccessKey]
	sessionToken := secret.Data[AWSSessionToken]
	return &credentials.Value{
		AccessKeyID:     string(accessKeyID),
		SecretAccessKey: string(secretAccessKey),
		SessionToken:    string(sessionToken),
	}, nil
}
