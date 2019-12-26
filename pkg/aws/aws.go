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
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/config/aws"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	maxRetries = 10
)

// GetConfigFromProfile extracts AWS creds from profile
func GetConfigFromProfile(profile *param.Profile) (map[string]string, error) {
	config := make(map[string]string)
	if profile.Location.Type != crv1alpha1.LocationTypeS3Compliant {
		return nil, errors.New("Not a S3Compliant location")
	}

	if profile.Credential.Type == param.CredentialTypeKeyPair {
		config[awsconfig.AccessKeyID] = profile.Credential.KeyPair.ID
		config[awsconfig.SecretAccessKey] = profile.Credential.KeyPair.Secret
	} else if profile.Credential.Type == param.CredentialTypeSecret {
		config[awsconfig.AccessKeyID] = string(profile.Credential.Secret.Data[secrets.AWSAccessKeyID])
		config[awsconfig.SecretAccessKey] = string(profile.Credential.Secret.Data[secrets.AWSSecretAccessKey])
		config[awsconfig.ConfigRole] = string(profile.Credential.Secret.Data[secrets.ConfigRole])
		config[awsconfig.SessionToken] = string(profile.Credential.Secret.Data[secrets.AWSSessionToken])
	}
	config[awsconfig.ConfigRegion] = profile.Location.Region
	return config, nil
}

// NewConfigWithSession creates aws.Config with session
func NewConfigWithSession(ctx context.Context, config map[string]string) (*aws.Config, *session.Session, error) {
	awsConfig, region, err := awsconfig.GetConfig(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create session")
	}
	creds := awsConfig.Credentials
	return awsConfig.WithMaxRetries(maxRetries).WithRegion(region).WithCredentials(creds), s, nil
}
