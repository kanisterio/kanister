package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	// ConfigRegion represents region key required in the map "config"
	ConfigRegion = "region"
	// ConfigRole represents the key for the ARN of the role which can be assumed.
	// It is optional.
	ConfigRole = "role"
	// AccessKeyID represents AWS Access key ID
	AccessKeyID = "AWS_ACCESS_KEY_ID"
	// SecretAccessKey represents AWS Secret Access Key
	SecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	// SessionToken represents AWS Session Key
	SessionToken = "AWS_SESSION_TOKEN"
)

// GetConfig returns a configuration to establish AWS connection, connected region name and the role to assume if it exists.
func GetConfig(config map[string]string) (awsConfig *aws.Config, region string, role string, err error) {
	region, ok := config[ConfigRegion]
	if !ok {
		return nil, "", "", errors.New("region required for storage type EBS")
	}
	accessKey, ok := config[AccessKeyID]
	if !ok {
		return nil, "", "", errors.New("AWS_ACCESS_KEY_ID required for storage type EBS")
	}
	secretAccessKey, ok := config[SecretAccessKey]
	if !ok {
		return nil, "", "", errors.New("AWS_SECRET_ACCESS_KEY required for storage type EBS")
	}
	sessionToken := config[SessionToken]
	role = config[ConfigRole]
	return &aws.Config{Credentials: credentials.NewStaticCredentials(accessKey, secretAccessKey, sessionToken)}, region, role, nil
}
