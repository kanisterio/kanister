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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pkg/errors"
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

	assumeRoleDuration = 25 * time.Minute
)

// GetConfig returns a configuration to establish AWS connection and connected region name.
func GetConfig(ctx context.Context, config map[string]string) (awsConfig *aws.Config, region string, err error) {
	region, ok := config[ConfigRegion]
	if !ok {
		return nil, "", errors.New("region required for storage type EBS/EFS")
	}
	accessKey, ok := config[AccessKeyID]
	if !ok {
		return nil, "", errors.New("AWS_ACCESS_KEY_ID required for storage type EBS/EFS")
	}
	secretAccessKey, ok := config[SecretAccessKey]
	if !ok {
		return nil, "", errors.New("AWS_SECRET_ACCESS_KEY required for storage type EBS/EFS")
	}
	role := config[ConfigRole]
	if role != "" {
		config, err := assumeRole(ctx, accessKey, secretAccessKey, role)
		if err != nil {
			return nil, "", errors.Wrap(err, "Failed to get temporary security credentials")
		}
		return config, region, nil
	}
	return &aws.Config{Credentials: credentials.NewStaticCredentials(accessKey, secretAccessKey, "")}, region, nil
}

func assumeRole(ctx context.Context, accessKey, secretAccessKey, role string) (*aws.Config, error) {
	creds, err := SwitchRole(ctx, accessKey, secretAccessKey, role, assumeRoleDuration)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to switch roles")
	}
	return &aws.Config{Credentials: creds}, nil
}
