package gcepd

import (
	"context"
	"os"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
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
			region: "us-west2",
			in:     "us-west2a",
			out:    "us-west2a",
		},
		{
			region: "us-west2",
			in:     "us-east1f",
			out:    "us-west2a",
		},
		{
			region: "us-west2",
			in:     "us-east2b",
			out:    "us-west2b",
		},
		{
			region: "us-west2",
			in:     "us-east1f",
			out:    "us-west2a",
		},
	} {
		config := getConfigForTest(c)
		provider, err := NewProvider(config)
		z, err := zone.WithUnknownNodeZones(ctx, provider.(zone.Mapper), tc.region, tc.in)
		c.Assert(err, IsNil)
		c.Assert(z, Not(Equals), "")
		if tc.out != "" {
			c.Assert(z, Equals, tc.out)
		}
	}
}

func getConfigForTest(c *C) map[string]string {
	config := make(map[string]string)
	creds, ok := os.LookupEnv(blockstorage.GoogleCloudCreds)
	if !ok {
		c.Skip("The necessary env variable GOOGLE_APPLICATION_CREDENTIALS is not set.")
	}
	config[blockstorage.GoogleCloudCreds] = creds
	return config
}
