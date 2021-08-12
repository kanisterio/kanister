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

	// AccessKeyID represents AWS Access key ID
	AccessKeyID = "AWS_ACCESS_KEY_ID"
	// SecretAccessKey represents AWS Secret Access Key
	SecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	// SessionToken represents AWS Session Key
	SessionToken = "AWS_SESSION_TOKEN"
	// Region represents AWS region
	Region = "AWS_REGION"

	// From AWS SDK "aws/session/env_config.go"
	webIdentityTokenFilePathEnvKey = "AWS_WEB_IDENTITY_TOKEN_FILE"
	roleARNEnvKey                  = "AWS_ROLE_ARN"

	// TODO: Make this configurable via `config`
	AssumeRoleDurationDefault = 90 * time.Minute
	AssumeRoleDuration        = "assumeRoleDuration"
)

func durationFromString(config map[string]string) (time.Duration, error) {
	d, ok := config[AssumeRoleDuration]
	if !ok || d == "" {
		return AssumeRoleDurationDefault, nil
	}
	return time.ParseDuration(d)
}

// GetCredentials returns credentials to use for AWS operations
func GetCredentials(ctx context.Context, config map[string]string) (*credentials.Credentials, error) {
	var creds *credentials.Credentials
	var assumedRole string
	assumeRoleDuration, err := durationFromString(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get assume role duration")
	}
	log.Debug().Print("Assume Role Duration setup", field.M{"assumeRoleDuration": assumeRoleDuration})
	switch {
	case config[AccessKeyID] != "" && config[SecretAccessKey] != "":
		// If AccessKeys were provided - use those
		creds = credentials.NewStaticCredentials(config[AccessKeyID], config[SecretAccessKey], "")
	case os.Getenv(webIdentityTokenFilePathEnvKey) != "" && os.Getenv(roleARNEnvKey) != "":
		sess, err := session.NewSessionWithOptions(session.Options{AssumeRoleDuration: assumeRoleDuration})
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create session to initialize Web Identify credentials")
		}
		creds = getCredentialsWithDuration(sess, assumeRoleDuration)
		assumedRole = os.Getenv(roleARNEnvKey)
	default:
		return nil, errors.New("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY required to initialize AWS credentials")
	}
	// If the caller didn't want to assume a different role, we're done
	if config[ConfigRole] == "" || config[ConfigRole] == assumedRole {
		return creds, nil
	}
	// When you use role chaining, your new credentials are limited to a maximum duration of one hour
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html
	if assumedRole != "" {
		assumeRoleDuration = 60 * time.Minute
	}

	// If the caller wants to use a specific role, use the credentials initialized above to assume that
	// role and return those credentials instead
	creds, err = awsrole.Switch(ctx, creds, config[ConfigRole], assumeRoleDuration)
	return creds, errors.Wrap(err, "Failed to switch roles")
}

// getCredentialsWithDuration returns credentials with the given duration.
// In order to set a custom assume role duration, we have to get the
// the provider first and then set it's Duration field before
// getting the credentials from the provider.
func getCredentialsWithDuration(sess *session.Session, duration time.Duration) *credentials.Credentials {
	svc := sts.New(sess)
	p := stscreds.NewWebIdentityRoleProvider(svc, os.Getenv(roleARNEnvKey), "", os.Getenv(webIdentityTokenFilePathEnvKey))
	p.Duration = duration
	return credentials.NewCredentials(p)
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
