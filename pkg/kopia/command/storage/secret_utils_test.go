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

package storage

import (
	"testing"

	"gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type StorageUtilsSuite struct{}

var _ = check.Suite(&StorageUtilsSuite{})

func (s *StorageUtilsSuite) TestBucketNameUtil(c *check.C) {
	loc := map[string]string{
		bucketKey:        "test-key",
		endpointKey:      "test-endpoint",
		prefixKey:        "test-prefix",
		regionKey:        "test-region",
		skipSSLVerifyKey: "true",
	}
	for _, tc := range []struct {
		LocType                    string
		expectedLocType            LocType
		skipSSLVerify              string
		expectedSkipSSLVerifyValue bool
	}{
		{
			LocType:                    "s3",
			expectedLocType:            LocTypeS3,
			skipSSLVerify:              "true",
			expectedSkipSSLVerifyValue: true,
		},
		{
			LocType:                    "gcs",
			expectedLocType:            LocTypeGCS,
			skipSSLVerify:              "false",
			expectedSkipSSLVerifyValue: false,
		},
		{
			LocType:                    "azure",
			expectedLocType:            LocTypeAzure,
			skipSSLVerify:              "true",
			expectedSkipSSLVerifyValue: true,
		},
	} {
		loc[typeKey] = tc.LocType
		loc[skipSSLVerifyKey] = tc.skipSSLVerify
		c.Assert(getBucketNameFromMap(loc), check.Equals, loc[bucketKey])
		c.Assert(getEndpointFromMap(loc), check.Equals, loc[endpointKey])
		c.Assert(getPrefixFromMap(loc), check.Equals, loc[prefixKey])
		c.Assert(getRegionFromMap(loc), check.Equals, loc[regionKey])
		c.Assert(checkSkipSSLVerifyFromMap(loc), check.Equals, tc.expectedSkipSSLVerifyValue)
		c.Assert(locationType(loc), check.Equals, tc.expectedLocType)
	}
}
