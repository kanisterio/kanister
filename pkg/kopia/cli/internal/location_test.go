// Copyright 2024 The Kanister Authors.
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

package internal_test

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

func TestLocation(t *testing.T) { check.TestingT(t) }

type LocationSuite struct{}

var _ = check.Suite(&LocationSuite{})

func (s *LocationSuite) TestLocation(c *check.C) {
	type expected struct {
		Type             rs.LocType
		Region           string
		BucketName       string
		Endpoint         string
		Prefix           string
		IsInsecure       bool
		HasSkipSSLVerify bool
		IsPITSupported   bool
	}

	tests := []struct {
		name     string
		location internal.Location
		expected expected
	}{
		{
			name:     "Test with no fields",
			location: internal.Location{},
			expected: expected{},
		},
		{
			name: "Test with all fields",
			location: internal.Location{
				rs.TypeKey:          []byte("Type1"),
				rs.RegionKey:        []byte("Region1"),
				rs.BucketKey:        []byte("Bucket1"),
				rs.EndpointKey:      []byte("http://Endpoint1"),
				rs.PrefixKey:        []byte("Prefix1"),
				rs.SkipSSLVerifyKey: []byte("true"),
			},
			expected: expected{
				Type:             "Type1",
				Region:           "Region1",
				BucketName:       "Bucket1",
				Endpoint:         "http://Endpoint1",
				Prefix:           "Prefix1",
				IsInsecure:       true,
				HasSkipSSLVerify: true,
			},
		},
		{
			name: "Test PIT Support for S3 Compliant",
			location: internal.Location{
				rs.TypeKey: []byte(rs.LocTypes3Compliant),
			},
			expected: expected{
				Type:           "s3Compliant",
				IsPITSupported: true,
			},
		},
		{
			name: "Test PIT Support for S3",
			location: internal.Location{
				rs.TypeKey: []byte(rs.LocTypeS3),
			},
			expected: expected{
				Type:           "s3",
				IsPITSupported: true,
			},
		},
		{
			name: "Test PIT Support for Azure",
			location: internal.Location{
				rs.TypeKey: []byte(rs.LocTypeAzure),
			},
			expected: expected{
				Type:           "azure",
				IsPITSupported: true,
			},
		},
		{
			name: "Test No PIT Support for GCS",
			location: internal.Location{
				rs.TypeKey: []byte(rs.LocTypeGCS),
			},
			expected: expected{
				Type:           "gcs",
				IsPITSupported: false,
			},
		},
		{
			name: "Test No PIT Support for FS",
			location: internal.Location{
				rs.TypeKey: []byte(rs.LocTypeFilestore),
			},
			expected: expected{
				Type:           "filestore",
				IsPITSupported: false,
			},
		},
	}
	for _, test := range tests {
		c.Check(test.location.Type(), check.Equals, test.expected.Type)
		c.Check(test.location.Region(), check.Equals, test.expected.Region)
		c.Check(test.location.BucketName(), check.Equals, test.expected.BucketName)
		c.Check(test.location.Endpoint(), check.Equals, test.expected.Endpoint)
		c.Check(test.location.Prefix(), check.Equals, test.expected.Prefix)
		c.Check(test.location.IsInsecureEndpoint(), check.Equals, test.expected.IsInsecure)
		c.Check(test.location.HasSkipSSLVerify(), check.Equals, test.expected.HasSkipSSLVerify)
		c.Check(test.location.IsPointInTypeSupported(), check.Equals, test.expected.IsPITSupported)
	}
}
