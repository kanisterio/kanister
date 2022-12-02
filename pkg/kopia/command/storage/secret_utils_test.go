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

func (s *StorageUtilsSuite) TestLocationUtils(c *check.C) {
	loc := map[string][]byte{
		bucketKey:        []byte("test-key"),
		endpointKey:      []byte("test-endpoint"),
		prefixKey:        []byte("test-prefix"),
		regionKey:        []byte("test-region"),
		skipSSLVerifyKey: []byte("true"),
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
		loc[typeKey] = []byte(tc.LocType)
		loc[skipSSLVerifyKey] = []byte(tc.skipSSLVerify)
		c.Assert(getBucketNameFromMap(loc), check.Equals, string(loc[bucketKey]))
		c.Assert(getEndpointFromMap(loc), check.Equals, string(loc[endpointKey]))
		c.Assert(getPrefixFromMap(loc), check.Equals, string(loc[prefixKey]))
		c.Assert(getRegionFromMap(loc), check.Equals, string(loc[regionKey]))
		c.Assert(checkSkipSSLVerifyFromMap(loc), check.Equals, tc.expectedSkipSSLVerifyValue)
		c.Assert(locationType(loc), check.Equals, tc.expectedLocType)
	}
}

func (s *StorageUtilsSuite) TestGetMapForLocationValues(c *check.C) {
	prefixValue := "test-prefix"
	regionValue := "test-region"
	bucketValue := "test-bucket"
	endpointValue := "test-endpoint"
	skipSSLVerifyValue := "true"
	for _, tc := range []struct {
		locType        LocType
		prefix         string
		region         string
		bucket         string
		endpoint       string
		skipSSLVerify  string
		expectedOutput map[string][]byte
	}{
		{
			locType: LocTypeS3,
			expectedOutput: map[string][]byte{
				typeKey: []byte(LocTypeS3),
			},
		},
		{
			locType: LocTypeS3,
			prefix:  prefixValue,
			expectedOutput: map[string][]byte{
				typeKey:   []byte(LocTypeS3),
				prefixKey: []byte(prefixValue),
			},
		},
		{
			locType: LocTypeS3,
			prefix:  prefixValue,
			region:  regionValue,
			expectedOutput: map[string][]byte{
				typeKey:   []byte(LocTypeS3),
				prefixKey: []byte(prefixValue),
				regionKey: []byte(regionValue),
			},
		},
		{
			locType: LocTypeS3,
			prefix:  prefixValue,
			region:  regionValue,
			bucket:  bucketValue,
			expectedOutput: map[string][]byte{
				typeKey:   []byte(LocTypeS3),
				prefixKey: []byte(prefixValue),
				regionKey: []byte(regionValue),
				bucketKey: []byte(bucketValue),
			},
		},
		{
			locType:  LocTypeS3,
			prefix:   prefixValue,
			region:   regionValue,
			bucket:   bucketValue,
			endpoint: endpointValue,
			expectedOutput: map[string][]byte{
				typeKey:     []byte(LocTypeS3),
				prefixKey:   []byte(prefixValue),
				regionKey:   []byte(regionValue),
				bucketKey:   []byte(bucketValue),
				endpointKey: []byte(endpointValue),
			},
		},
		{
			locType:       LocTypeS3,
			prefix:        prefixValue,
			region:        regionValue,
			bucket:        bucketValue,
			endpoint:      endpointValue,
			skipSSLVerify: skipSSLVerifyValue,
			expectedOutput: map[string][]byte{
				typeKey:          []byte(LocTypeS3),
				prefixKey:        []byte(prefixValue),
				regionKey:        []byte(regionValue),
				bucketKey:        []byte(bucketValue),
				endpointKey:      []byte(endpointValue),
				skipSSLVerifyKey: []byte(skipSSLVerifyValue),
			},
		},
	} {
		op := GetMapForLocationValues(
			tc.locType,
			tc.prefix,
			tc.region,
			tc.bucket,
			tc.endpoint,
			tc.skipSSLVerify,
		)
		c.Assert(op, check.DeepEquals, tc.expectedOutput)
	}
}
