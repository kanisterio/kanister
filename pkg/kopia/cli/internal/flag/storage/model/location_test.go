package model

import (
	"testing"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"gopkg.in/check.v1"
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
	}

	tests := []struct {
		name     string
		location Location
		expected expected
	}{
		{
			name:     "Test with no fields",
			location: Location{},
			expected: expected{},
		},
		{
			name: "Test with all fields",
			location: Location{
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
	}
	for _, test := range tests {
		c.Check(test.location.Type(), check.Equals, test.expected.Type)
		c.Check(test.location.Region(), check.Equals, test.expected.Region)
		c.Check(test.location.BucketName(), check.Equals, test.expected.BucketName)
		c.Check(test.location.Endpoint(), check.Equals, test.expected.Endpoint)
		c.Check(test.location.Prefix(), check.Equals, test.expected.Prefix)
		c.Check(test.location.IsInsecureEndpoint(), check.Equals, test.expected.IsInsecure)
		c.Check(test.location.HasSkipSSLVerify(), check.Equals, test.expected.HasSkipSSLVerify)
	}
}
