// Copyright 2022 The Kanister Authors.
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
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	envconfig "github.com/kanisterio/kanister/pkg/config"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type AWSSuite struct{}

var _ = check.Suite(&AWSSuite{})

type mockSTSClient struct {
	stsiface.STSAPI
	getCallerIdentityFunc func(*sts.GetCallerIdentityInput) (*sts.GetCallerIdentityOutput, error)
}

func (m *mockSTSClient) GetCallerIdentity(input *sts.GetCallerIdentityInput) (*sts.GetCallerIdentityOutput, error) {
	return m.getCallerIdentityFunc(input)
}

func (s AWSSuite) TestValidCreds(c *check.C) {
	ctx := context.Background()
	config := map[string]string{}
	config[AccessKeyID] = envconfig.GetEnvOrSkip(c, AccessKeyID)
	config[SecretAccessKey] = envconfig.GetEnvOrSkip(c, SecretAccessKey)
	config[ConfigRegion] = "us-west-2"

	mockSTS := &mockSTSClient{
		getCallerIdentityFunc: func(input *sts.GetCallerIdentityInput) (*sts.GetCallerIdentityOutput, error) {
			return &sts.GetCallerIdentityOutput{}, nil
		},
	}
	// Test with valid credentials
	res, err := IsAwsCredsValidWithSTS(ctx, config, mockSTS)
	c.Assert(err, check.IsNil)
	c.Assert(res, check.Equals, true)

	// Test with invalid credentials
	mockSTS = &mockSTSClient{
		getCallerIdentityFunc: func(input *sts.GetCallerIdentityInput) (*sts.GetCallerIdentityOutput, error) {
			return nil, errors.New("invalid credentials")
		},
	}
	config[AccessKeyID] = "fake-access-id"
	res, err = IsAwsCredsValidWithSTS(ctx, config, mockSTS)
	c.Assert(err, check.NotNil)
	c.Assert(res, check.Equals, false)
}
