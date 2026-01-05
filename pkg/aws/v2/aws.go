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

// Package awsv2 provides utilities for interacting with AWS services, including
// credential management, role switching, and configuration setup.
package awsv2

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/kanisterio/errkit"
	
	awsrole "github.com/kanisterio/kanister/pkg/aws/v2/role"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	// ConfigRegion represents region key required in the map "config".
	ConfigRegion = "region"
	// ConfigRole represents the key for the ARN of the role which can be assumed.
	// It is optional.
	ConfigRole = "role"

	// ConfigEFSVaultName represents the key for the EFS vault name in the map
	// "config". It is optional.
	ConfigEFSVaultName = "efsVaultName"

	// ConfigWebIdentityToken represents the key for AWS Web Identity token.
	ConfigWebIdentityToken = "webIdentityToken"

	// AccessKeyID represents AWS Access key ID.
	AccessKeyID = "AWS_ACCESS_KEY_ID"
	// SecretAccessKey represents AWS Secret Access Key.
	SecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	// SessionToken represents AWS Session Key.
	SessionToken = "AWS_SESSION_TOKEN"
	// Region represents AWS region.
	Region = "AWS_REGION"

	// From AWS SDK "aws/session/env_config.go".
	WebIdentityTokenFilePathEnvKey = "AWS_WEB_IDENTITY_TOKEN_FILE"
	RoleARNEnvKey                  = "AWS_ROLE_ARN"

	// TODO: Make this configurable via `config`.
	AssumeRoleDurationDefault = 60 * time.Minute
	AssumeRoleDuration        = "assumeRoleDuration"
)

var _ stscreds.IdentityTokenRetriever = (*staticTokenRetriever)(nil)

// staticTokenRetriever implements the stscreds.IdentityTokenRetriever interface for retrieval of
// plaintext web identity token.
type staticTokenRetriever string

// GetIdentityToken returns a plaintext web identity token as is.
func (f staticTokenRetriever) GetIdentityToken() ([]byte, error) {
	return []byte(f), nil
}

// fileTokenRetriever implements the stscreds.IdentityTokenRetriever interface for retrieval of
// web identity token from file.
type fileTokenRetriever struct {
	filePath string
}

// GetIdentityToken reads web identity token from file.
func (f fileTokenRetriever) GetIdentityToken() ([]byte, error) {
	tokenBytes, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to read web identity token file")
	}
	return tokenBytes, nil
}

func durationFromString(config map[string]string) (time.Duration, error) {
	d, ok := config[AssumeRoleDuration]
	if !ok || d == "" {
		return AssumeRoleDurationDefault, nil
	}
	return time.ParseDuration(d)
}

func authenticateAWSCredentials(
	ctx context.Context,
	config map[string]string,
	assumeRoleDuration time.Duration,
) (aws.CredentialsProvider, string, error) {
	// If AccessKeys were provided - use those.
	creds := fetchStaticAWSCredentials(config)
	if creds != nil {
		return creds, "", nil
	}

	// If Web Identity token and role were provided - use them.
	var err error
	creds, err = fetchWebIdentityTokenFromConfig(ctx, config, assumeRoleDuration)
	if err != nil {
		return nil, "", err
	}
	if creds != nil {
		return creds, config[ConfigRole], nil
	}

	// Otherwise use Web Identity token file and role provided via ENV.
	creds, err = fetchWebIdentityTokenFromFile(ctx, assumeRoleDuration)
	if err != nil {
		return nil, "", err
	}
	if creds != nil {
		return creds, os.Getenv(RoleARNEnvKey), nil
	}

	return nil, "", errkit.New("Missing AWS credentials, please check that either AWS access keys or web identity token are provided")
}

func fetchStaticAWSCredentials(config map[string]string) aws.CredentialsProvider {
	if config[AccessKeyID] == "" || config[SecretAccessKey] == "" {
		return nil
	}

	return credentials.NewStaticCredentialsProvider(config[AccessKeyID], config[SecretAccessKey], "")
}

func fetchWebIdentityTokenFromConfig(ctx context.Context, config map[string]string, assumeRoleDuration time.Duration) (aws.CredentialsProvider, error) {
	if config[ConfigWebIdentityToken] == "" || config[ConfigRole] == "" {
		return nil, nil
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config[ConfigRegion]))
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to load default config for web identity token from config")
	}

	creds := stscreds.NewWebIdentityRoleProvider(
		sts.NewFromConfig(cfg),
		config[ConfigRole],
		staticTokenRetriever(config[ConfigWebIdentityToken]),
		func(o *stscreds.WebIdentityRoleOptions) {
			o.Duration = assumeRoleDuration
		},
	)

	return creds, nil
}

func fetchWebIdentityTokenFromFile(ctx context.Context, assumeRoleDuration time.Duration) (aws.CredentialsProvider, error) {
	if os.Getenv(WebIdentityTokenFilePathEnvKey) == "" || os.Getenv(RoleARNEnvKey) == "" {
		return nil, nil
	}

	// LoadDefaultConfig will use AWS_REGION or AWS_DEFAULT_REGION from environment if set
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to load default config for web identity token")
	}

	provider := stscreds.NewWebIdentityRoleProvider(
		sts.NewFromConfig(cfg),
		os.Getenv(RoleARNEnvKey),
		fileTokenRetriever{filePath: os.Getenv(WebIdentityTokenFilePathEnvKey)},
		func(o *stscreds.WebIdentityRoleOptions) {
			o.Duration = assumeRoleDuration
		},
	)

	return provider, nil
}

// switchAWSRole checks if the caller wants to assume a different role
// return as is if ConfigRole is empty, or already same as assumedRole
// otherwise proceed to switch role.
func switchAWSRole(ctx context.Context, creds aws.CredentialsProvider, region string, targetRole string, currentRole string, assumeRoleDuration time.Duration) (aws.CredentialsProvider, error) {
	if targetRole == "" || targetRole == currentRole {
		return creds, nil
	}
	// When you use role chaining, your new credentials are limited to a maximum duration of one hour
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html
	if currentRole != "" {
		assumeRoleDuration = 60 * time.Minute
	}
	// If the caller wants to use a specific role, use the credentials initialized above to assume that
	// role and return those credentials instead.
	creds, err := awsrole.Switch(ctx, creds, region, targetRole, assumeRoleDuration)
	return creds, errkit.Wrap(err, "Failed to switch roles")
}

// GetCredentials returns credentials provider to use for AWS operations.
func GetCredentials(ctx context.Context, config map[string]string) (aws.CredentialsProvider, error) {
	region := config[ConfigRegion]
	assumeRoleDuration, err := durationFromString(config)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get assume role duration")
	}
	log.Debug().Print("Assume Role Duration setup", field.M{"assumeRoleDuration": assumeRoleDuration})

	creds, assumedRole, err := authenticateAWSCredentials(ctx, config, assumeRoleDuration)
	if err != nil {
		return nil, err
	}
	// Check if role switching is needed, then return creds.
	return switchAWSRole(ctx, creds, region, config[ConfigRole], assumedRole, assumeRoleDuration)
}

// GetConfig returns a configuration to establish AWS connection and connected region name.
func GetConfig(ctx context.Context, config map[string]string) (awsConfig aws.Config, region string, err error) {
	region, ok := config[ConfigRegion]
	if !ok {
		return aws.Config{}, "", errkit.New("region required for storage type EBS/EFS")
	}
	creds, err := GetCredentials(ctx, config)
	if err != nil {
		return aws.Config{}, "", errkit.Wrap(err, "could not initialize AWS credentials for operation")
	}
	return aws.Config{
		Region:      region,
		Credentials: creds,
	}, region, nil
}

func IsAwsCredsValid(ctx context.Context, config map[string]string) (bool, error) {
	awsConfig, _, err := GetConfig(ctx, config)
	if err != nil {
		return false, errkit.Wrap(err, "Failed to get config for AWS creds")
	}
	awsConfig.RetryMaxAttempts = 10

	stsCli := sts.NewFromConfig(awsConfig)
	_, err = stsCli.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return false, errkit.Wrap(err, "Failed to get user with provided creds")
	}
	return true, nil
}

// stsAPI defines the interface for STS operations, used for testing.
type stsAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// IsAwsCredsValidWithSTS uses the provided STS client to validate AWS credentials. with STS v2 sdk.
// Method solely for testing purposes, as it allows passing a mock STS client.
func IsAwsCredsValidWithSTS(ctx context.Context, config map[string]string, stsCli stsAPI) (bool, error) {
	_, err := stsCli.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return false, err
	}
	return true, nil
}
