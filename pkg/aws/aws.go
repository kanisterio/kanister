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

package aws

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"

	awsrole "github.com/kanisterio/kanister/pkg/aws/role"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	// ConfigRegion represents region key required in the map "config"
	ConfigRegion = "region"
	// ConfigRole represents the key for the ARN of the role which can be assumed.
	// It is optional.
	ConfigRole = "role"

	// ConfigEFSVaultName represents the key for the EFS vault name in the map
	// "config". It is optional.
	ConfigEFSVaultName = "efsVaultName"

	// ConfigWebIdentityToken represents the key for AWS Web Identity token
	ConfigWebIdentityToken = "webIdentityToken"

	// AccessKeyID represents AWS Access key ID
	AccessKeyID = "AWS_ACCESS_KEY_ID"
	// SecretAccessKey represents AWS Secret Access Key
	SecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	// SessionToken represents AWS Session Key
	SessionToken = "AWS_SESSION_TOKEN"
	// Region represents AWS region
	Region = "AWS_REGION"

	// From AWS SDK "aws/session/env_config.go"
	WebIdentityTokenFilePathEnvKey = "AWS_WEB_IDENTITY_TOKEN_FILE"
	RoleARNEnvKey                  = "AWS_ROLE_ARN"

	// TODO: Make this configurable via `config`
	AssumeRoleDurationDefault = 60 * time.Minute
	AssumeRoleDuration        = "assumeRoleDuration"
)

var _ stscreds.TokenFetcher = (*staticToken)(nil)

// staticToken implements stscreds.TokenFetcher interface for retrieval of plaintext web
// identity token
type staticToken string

// FetchToken returns a plaintext web identity token as is.
func (f staticToken) FetchToken(ctx credentials.Context) ([]byte, error) {
	return []byte(f), nil
}

func durationFromString(config map[string]string) (time.Duration, error) {
	d, ok := config[AssumeRoleDuration]
	if !ok || d == "" {
		return AssumeRoleDurationDefault, nil
	}
	return time.ParseDuration(d)
}

func authenticateAWSCredentials(
	config map[string]string,
	assumeRoleDuration time.Duration,
) (*credentials.Credentials, string, error) {
	// If AccessKeys were provided - use those
	creds := fetchStaticAWSCredentials(config)
	if creds != nil {
		return creds, "", nil
	}

	// If Web Identity token and role were provided - use them
	var err error
	creds, err = fetchWebIdentityTokenFromConfig(config, assumeRoleDuration)
	if err != nil {
		return nil, "", err
	}
	if creds != nil {
		return creds, config[ConfigRole], nil
	}

	// Otherwise use Web Identity token file and role provided via ENV
	creds, err = fetchWebIdentityTokenFromFile(assumeRoleDuration)
	if err != nil {
		return nil, "", err
	}
	if creds != nil {
		return creds, os.Getenv(RoleARNEnvKey), nil
	}

	return nil, "", errors.New("Missing AWS credentials, please check that either AWS access keys or web identity token are provided")
}

func fetchStaticAWSCredentials(config map[string]string) *credentials.Credentials {
	if config[AccessKeyID] == "" || config[SecretAccessKey] == "" {
		return nil
	}

	return credentials.NewStaticCredentials(config[AccessKeyID], config[SecretAccessKey], "")
}

func fetchWebIdentityTokenFromConfig(config map[string]string, assumeRoleDuration time.Duration) (*credentials.Credentials, error) {
	if config[ConfigWebIdentityToken] == "" || config[ConfigRole] == "" {
		return nil, nil
	}

	creds, err := getCredentialsWithDuration(
		config[ConfigRole],
		staticToken(config[ConfigWebIdentityToken]),
		assumeRoleDuration,
	)
	if err != nil {
		return nil, err
	}

	return creds, nil
}

func fetchWebIdentityTokenFromFile(assumeRoleDuration time.Duration) (*credentials.Credentials, error) {
	if os.Getenv(WebIdentityTokenFilePathEnvKey) == "" || os.Getenv(RoleARNEnvKey) == "" {
		return nil, nil
	}

	creds, err := getCredentialsWithDuration(
		os.Getenv(RoleARNEnvKey),
		stscreds.FetchTokenPath(os.Getenv(WebIdentityTokenFilePathEnvKey)),
		assumeRoleDuration,
	)
	if err != nil {
		return nil, err
	}

	return creds, nil
}

// switchAWSRole checks if the caller wants to assume a different role
// return as is if ConfigRole is empty, or already same as assumedRole
// otherwise proceed to switch role
func switchAWSRole(ctx context.Context, creds *credentials.Credentials, targetRole string, currentRole string, assumeRoleDuration time.Duration) (*credentials.Credentials, error) {
	if targetRole == "" || targetRole == currentRole {
		return creds, nil
	}
	// When you use role chaining, your new credentials are limited to a maximum duration of one hour
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html
	if currentRole != "" {
		assumeRoleDuration = 60 * time.Minute
	}
	// If the caller wants to use a specific role, use the credentials initialized above to assume that
	// role and return those credentials instead
	creds, err := awsrole.Switch(ctx, creds, targetRole, assumeRoleDuration)
	return creds, errors.Wrap(err, "Failed to switch roles")
}

// GetCredentials returns credentials to use for AWS operations
func GetCredentials(ctx context.Context, config map[string]string) (*credentials.Credentials, error) {
	assumeRoleDuration, err := durationFromString(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get assume role duration")
	}
	log.Debug().Print("Assume Role Duration setup", field.M{"assumeRoleDuration": assumeRoleDuration})

	// authenticate AWS creds
	creds, assumedRole, err := authenticateAWSCredentials(config, assumeRoleDuration)
	if err != nil {
		return nil, err
	}
	// check if role switching is needed, then return creds
	return switchAWSRole(ctx, creds, config[ConfigRole], assumedRole, assumeRoleDuration)
}

// getCredentialsWithDuration returns credentials with the given duration.
// In order to set a custom assume role duration, we have to get the
// the provider first and then set it's Duration field before
// getting the credentials from the provider.
func getCredentialsWithDuration(
	roleARN string,
	tokenFetcher stscreds.TokenFetcher,
	duration time.Duration,
) (*credentials.Credentials, error) {
	sess, err := session.NewSessionWithOptions(session.Options{AssumeRoleDuration: duration})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session to initialize Web Identify credentials")
	}

	svc := sts.New(sess)
	p := stscreds.NewWebIdentityRoleProviderWithOptions(
		svc,
		roleARN,
		"",
		tokenFetcher,
	)
	p.Duration = duration
	return credentials.NewCredentials(p), nil
}

// GetConfig returns a configuration to establish AWS connection and connected region name.
func GetConfig(ctx context.Context, config map[string]string) (awsConfig *aws.Config, region string, err error) {
	region, ok := config[ConfigRegion]
	if !ok {
		return nil, "", errors.New("region required for storage type EBS/EFS")
	}
	creds, err := GetCredentials(ctx, config)
	if err != nil {
		return nil, "", errors.Wrap(err, "could not initialize AWS credentials for operation")
	}
	return &aws.Config{Credentials: creds}, region, nil
}

func IsAwsCredsValid(ctx context.Context, config map[string]string) (bool, error) {
	var maxRetries int = 10
	awsConfig, region, err := GetConfig(ctx, config)
	if err != nil {
		return false, errors.Wrap(err, "Failed to get config for AWS creds")
	}
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return false, errors.Wrap(err, "Failed to create session with provided creds")
	}
	stsCli := sts.New(s, aws.NewConfig().WithRegion(region).WithMaxRetries(maxRetries))
	_, err = stsCli.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return false, errors.Wrap(err, "Failed to get user with provided creds")
	}
	return true, nil
}
