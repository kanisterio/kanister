package storage

import (
	"testing"

	"gopkg.in/check.v1"

	v1 "k8s.io/api/core/v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type StorageUtilsSuite struct{}

var _ = check.Suite(&StorageUtilsSuite{})

func (s *StorageUtilsSuite) TestBucketNameUtil(c *check.C) {
	sec := &v1.Secret{
		StringData: map[string]string{
			bucketKey:        "test-key",
			endpointKey:      "test-endpoint",
			prefixKey:        "test-prefix",
			regionKey:        "test-region",
			skipSSLVerifyKey: "true",
		},
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
		sec.StringData[typeKey] = tc.locType
		sec.StringData[skipSSLVerifyKey] = tc.skipSSLVerify
		c.Assert(bucketName(sec), check.Equals, sec.StringData[bucketKey])
		c.Assert(endpoint(sec), check.Equals, sec.StringData[endpointKey])
		c.Assert(prefix(sec), check.Equals, sec.StringData[prefixKey])
		c.Assert(region(sec), check.Equals, sec.StringData[regionKey])
		c.Assert(skipSSLVerify(sec), check.Equals, tc.expectedSkipSSLVerifyValue)
		c.Assert(locationType(sec), check.Equals, tc.expectedLocType)
	}
}
