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
	"testing"

	"gopkg.in/check.v1"

	envconfig "github.com/kanisterio/kanister/pkg/config"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type AWSSuite struct{}

var _ = check.Suite(&AWSSuite{})

func (s AWSSuite) TestValidCreds(c *check.C) {
	ctx := context.Background()
	config := map[string]string{}

	config[AccessKeyID] = envconfig.GetEnvOrSkip(c, AccessKeyID)
	config[SecretAccessKey] = envconfig.GetEnvOrSkip(c, SecretAccessKey)
	config[ConfigRegion] = "us-west-2"

	res, err := IsAwsCredsValid(ctx, config)
	c.Assert(err, check.IsNil)
	c.Assert(res, check.Equals, true)

	config[AccessKeyID] = "fake-access-id"
	res, err = IsAwsCredsValid(ctx, config)
	c.Assert(err, check.NotNil)
	c.Assert(res, check.Equals, false)
}
