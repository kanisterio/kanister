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
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"gopkg.in/check.v1"

	envconfig "github.com/kanisterio/kanister/pkg/config"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type AWSSuite struct{}

var _ = check.Suite(&AWSSuite{})

type mockSTSClient struct {
	getCallerIdentityFunc func(context.Context, *sts.GetCallerIdentityInput, ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func (m *mockSTSClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return m.getCallerIdentityFunc(ctx, params, optFns...)
}

func (s AWSSuite) TestResolveRegion(c *check.C) {
	origRegion, regionSet := os.LookupEnv("AWS_REGION")
	origDefault, defaultSet := os.LookupEnv("AWS_DEFAULT_REGION")
	defer func() {
		restoreEnv(c, "AWS_REGION", origRegion, regionSet)
		restoreEnv(c, "AWS_DEFAULT_REGION", origDefault, defaultSet)
	}()

	cases := []struct {
		name       string
		config     map[string]string
		awsRegion  string
		awsDefault string
		expected   string
	}{
		{
			name:       "config region wins over env",
			config:     map[string]string{ConfigRegion: "us-east-1"},
			awsRegion:  "us-west-2",
			awsDefault: "eu-west-1",
			expected:   "us-east-1",
		},
		{
			name:       "falls back to AWS_REGION when config empty",
			config:     map[string]string{},
			awsRegion:  "us-west-2",
			awsDefault: "eu-west-1",
			expected:   "us-west-2",
		},
		{
			name:       "falls back to AWS_DEFAULT_REGION when AWS_REGION unset",
			config:     map[string]string{},
			awsRegion:  "",
			awsDefault: "eu-west-1",
			expected:   "eu-west-1",
		},
		{
			name:       "returns empty when nothing is set",
			config:     map[string]string{},
			awsRegion:  "",
			awsDefault: "",
			expected:   "",
		},
		{
			name:       "nil config falls back to env",
			config:     nil,
			awsRegion:  "ap-south-1",
			awsDefault: "",
			expected:   "ap-south-1",
		},
		{
			name:       "empty config entry treated as unset",
			config:     map[string]string{ConfigRegion: ""},
			awsRegion:  "us-west-2",
			awsDefault: "",
			expected:   "us-west-2",
		},
	}

	for _, tc := range cases {
		c.Assert(os.Setenv("AWS_REGION", tc.awsRegion), check.IsNil)
		c.Assert(os.Setenv("AWS_DEFAULT_REGION", tc.awsDefault), check.IsNil)
		got := resolveRegion(tc.config)
		c.Assert(got, check.Equals, tc.expected, check.Commentf("case: %s", tc.name))
	}
}

func restoreEnv(c *check.C, key, value string, wasSet bool) {
	if wasSet {
		c.Assert(os.Setenv(key, value), check.IsNil)
		return
	}
	c.Assert(os.Unsetenv(key), check.IsNil)
}

// stsMinDurationSeconds is the smallest DurationSeconds AWS STS accepts for
// AssumeRole / AssumeRoleWithWebIdentity (the documented minimum is 900s / 15m).
const stsMinDurationSeconds = 900

// stsDurationSeconds mirrors how the AWS SDK v2 stscreds providers derive the
// STS DurationSeconds request parameter from the configured session duration:
// DurationSeconds = int32(duration / time.Second).
func stsDurationSeconds(d time.Duration) int32 {
	return int32(d / time.Second)
}

// TestAssumeRoleDurationRoundTrip guards against passing a sub-second session
// duration (for example time.Duration(time.Now().Second()), which yields 0-59
// nanoseconds) into the assume-role credential path. Such a value is stored as
// its String() form, parsed back by durationFromString, and finally divided by
// time.Second by the SDK, collapsing to DurationSeconds=0 while remaining
// non-zero. STS rejects that for role-based secrets, so the resolved duration
// must convert to a DurationSeconds of at least the STS minimum.
func (s AWSSuite) TestAssumeRoleDurationRoundTrip(c *check.C) {
	// The default used across the credential call sites must resolve to a
	// valid, non-truncating STS session duration.
	d, err := durationFromString(map[string]string{
		AssumeRoleDuration: AssumeRoleDurationDefault.String(),
	})
	c.Assert(err, check.IsNil)
	c.Assert(stsDurationSeconds(d) >= stsMinDurationSeconds, check.Equals, true,
		check.Commentf("resolved DurationSeconds=%d must be >= %d", stsDurationSeconds(d), stsMinDurationSeconds))

	// An unset/empty duration falls back to the default and is likewise valid.
	d, err = durationFromString(map[string]string{})
	c.Assert(err, check.IsNil)
	c.Assert(stsDurationSeconds(d) >= stsMinDurationSeconds, check.Equals, true,
		check.Commentf("default DurationSeconds=%d must be >= %d", stsDurationSeconds(d), stsMinDurationSeconds))

	// Document the regressed behaviour: a sub-second duration round-trips to a
	// DurationSeconds of 0 while staying non-zero, which STS rejects. The value
	// that reaches this path must never be sub-second.
	for _, ns := range []time.Duration{1, 37, 59} {
		bad, perr := durationFromString(map[string]string{
			AssumeRoleDuration: ns.String(),
		})
		c.Assert(perr, check.IsNil)
		c.Assert(stsDurationSeconds(bad), check.Equals, int32(0),
			check.Commentf("a %s duration truncates to DurationSeconds=0", ns))
	}
}

func (s AWSSuite) TestValidCreds(c *check.C) {
	ctx := context.Background()
	config := map[string]string{}
	config[AccessKeyID] = envconfig.GetEnvOrSkip(c, AccessKeyID)
	config[SecretAccessKey] = envconfig.GetEnvOrSkip(c, SecretAccessKey)
	config[ConfigRegion] = "us-west-2"

	mockSTS := &mockSTSClient{
		getCallerIdentityFunc: func(ctx context.Context, input *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return &sts.GetCallerIdentityOutput{}, nil
		},
	}
	// Test with valid credentials
	res, err := IsAwsCredsValidWithSTS(ctx, config, mockSTS)
	c.Assert(err, check.IsNil)
	c.Assert(res, check.Equals, true)

	// Test with invalid credentials
	mockSTS = &mockSTSClient{
		getCallerIdentityFunc: func(ctx context.Context, input *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return nil, errors.New("invalid credentials")
		},
	}
	config[AccessKeyID] = "fake-access-id"
	res, err = IsAwsCredsValidWithSTS(ctx, config, mockSTS)
	c.Assert(err, check.NotNil)
	c.Assert(res, check.Equals, false)
}
