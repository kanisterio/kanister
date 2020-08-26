package vmware

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type VMWareSuite struct{}

var _ = Suite(&VMWareSuite{})

func (s *VMWareSuite) TestURLParse(c *C) {
	for _, tc := range []struct {
		config       map[string]string
		errCheck     Checker
		expErrString string
	}{
		{
			config:       map[string]string{},
			errCheck:     NotNil,
			expErrString: "Failed to find VSphere endpoint value",
		},
		{
			config: map[string]string{
				VSphereEndpointKey: "ep",
			},
			errCheck:     NotNil,
			expErrString: "Failed to find VSphere username value",
		},
		{
			config: map[string]string{
				VSphereEndpointKey: "ep",
				VSphereUsernameKey: "user",
			},
			errCheck:     NotNil,
			expErrString: "Failed to find VSphere password value",
		},
		{ // until we can run against a VIM setup this will always fail.
			config: map[string]string{
				VSphereEndpointKey: "ep",
				VSphereUsernameKey: "user",
				VSpherePasswordKey: "pass",
			},
			errCheck:     NotNil,
			expErrString: "Failed to create VIM client",
		},
	} {
		_, err := NewProvider(tc.config)
		c.Check(err, tc.errCheck)
		if err != nil {
			c.Assert(err, ErrorMatches, ".*"+tc.expErrString+".*")
		}
	}
}
