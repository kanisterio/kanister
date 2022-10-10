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
		locType                    string
		expectedLocType            locType
		skipSSLVerify              string
		expectedSkipSSLVerifyValue bool
	}{
		{
			locType:                    "s3",
			expectedLocType:            locTypeS3,
			skipSSLVerify:              "true",
			expectedSkipSSLVerifyValue: true,
		},
		{
			locType:                    "gcs",
			expectedLocType:            locTypeGCS,
			skipSSLVerify:              "false",
			expectedSkipSSLVerifyValue: false,
		},
		{
			locType:                    "azure",
			expectedLocType:            locTypeAzure,
			skipSSLVerify:              "true",
			expectedSkipSSLVerifyValue: true,
		},
	} {
		loc[typeKey] = tc.locType
		loc[skipSSLVerifyKey] = tc.skipSSLVerify
		c.Assert(bucketName(loc), check.Equals, loc[bucketKey])
		c.Assert(endpoint(loc), check.Equals, loc[endpointKey])
		c.Assert(prefix(loc), check.Equals, loc[prefixKey])
		c.Assert(region(loc), check.Equals, loc[regionKey])
		c.Assert(skipSSLVerify(loc), check.Equals, tc.expectedSkipSSLVerifyValue)
		c.Assert(locationType(loc), check.Equals, tc.expectedLocType)
	}
}
