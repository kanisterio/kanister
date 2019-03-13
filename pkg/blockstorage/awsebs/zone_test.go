package awsebs

import (
	"context"
	"os"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage/zone"
)

type ZoneSuite struct{}

var _ = Suite(&ZoneSuite{})

func (s ZoneSuite) TestZoneWithUnknownNodeZones(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		region string
		in     string
		out    string
	}{
		{
			region: "us-west-2",
			in:     "us-west-2a",
			out:    "us-west-2a",
		},
		{
			region: "us-west-2",
			in:     "us-east-1f",
			out:    "us-west-2a",
		},
		{
			region: "us-west-2",
			in:     "us-east-2b",
			out:    "us-west-2b",
		},
		{
			region: "us-west-2",
			in:     "us-east-1f",
			out:    "us-west-2a",
		},
	} {
		config := getConfigForTest(c, tc.region)
		provider, err := NewProvider(config)
		z, err := zone.WithUnknownNodeZones(ctx, provider.(zone.Mapper), tc.region, tc.in)
		c.Assert(err, IsNil)
		c.Assert(z, Not(Equals), "")
		if tc.out != "" {
			c.Assert(z, Equals, tc.out)
		}
	}
}

func getConfigForTest(c *C, region string) map[string]string {
	config := make(map[string]string)
	config[ConfigRegion] = region
	accessKey, ok := os.LookupEnv(AccessKeyID)
	if !ok {
		c.Skip("The necessary env variable AWS_ACCESS_KEY_ID is not set.")
	}
	secretAccessKey, ok := os.LookupEnv(SecretAccessKey)
	if !ok {
		c.Skip("The necessary env variable AWS_SECRET_ACCESS_KEY is not set.")
	}
	config[AccessKeyID] = accessKey
	config[SecretAccessKey] = secretAccessKey

	return config
}
