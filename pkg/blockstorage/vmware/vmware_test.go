package vmware

import (
	"os"
	"testing"
	"time"

	vapitags "github.com/vmware/govmomi/vapi/tags"
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

func (s *VMWareSuite) TestTimeoutEnvSetting(c *C) {
	tempEnv := os.Getenv(vmWareTimeoutMinEnv)
	os.Unsetenv(vmWareTimeoutMinEnv)
	timeout := time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, defaultWaitTime)

	os.Setenv(vmWareTimeoutMinEnv, "7")
	timeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, 7*time.Minute)

	os.Setenv(vmWareTimeoutMinEnv, "badValue")
	timeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, defaultWaitTime)

	os.Setenv(vmWareTimeoutMinEnv, "-1")
	timeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, defaultWaitTime)

	os.Setenv(vmWareTimeoutMinEnv, "0")
	timeout = time.Duration(getEnvAsIntOrDefault(vmWareTimeoutMinEnv, int(defaultWaitTime/time.Minute))) * time.Minute
	c.Assert(timeout, Equals, defaultWaitTime)

	timeout = time.Duration(getEnvAsIntOrDefault("someotherenv", 5)) * time.Minute
	c.Assert(timeout, Equals, 5*time.Minute)

	os.Setenv(vmWareTimeoutMinEnv, tempEnv)
}

func (s *VMWareSuite) TestGetSnapshotIDsFromTags(c *C) {
	for _, tc := range []struct {
		catTags    []vapitags.Tag
		tags       map[string]string
		errChecker Checker
		snapIDs    []string
	}{
		{
			catTags: []vapitags.Tag{
				{Name: "v1:s1:k1:v1"},
				{Name: "v1:s1:k2:v2"},
				{Name: "v1:s2:k1:v1"},
			},
			tags: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			snapIDs:    []string{"s1"},
			errChecker: IsNil,
		},
		{
			catTags: []vapitags.Tag{
				{Name: "v1:s1:k1:v1"},
				{Name: "v1:s1:k2:v2"},
				{Name: "v1:s2:k1:v1"},
			},
			tags: map[string]string{
				"k1": "v1",
			},
			snapIDs:    []string{"s1", "s2"},
			errChecker: IsNil,
		},
		{
			catTags: []vapitags.Tag{
				{Name: "v1:s1:k1:v1"},
				{Name: "v1:s1:k2:v2"},
				{Name: "v1:s2:k1:v1"},
			},
			snapIDs:    []string{"s1", "s2"},
			errChecker: IsNil,
		},
		{
			catTags: []vapitags.Tag{
				{Name: "v1:s1k1:v1"},
			},
			tags: map[string]string{
				"k1": "v1",
			},
			errChecker: NotNil,
		},
	} {
		fp := &FcdProvider{}
		snapIDs, err := fp.getSnapshotIDsFromTags(tc.catTags, tc.tags)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			c.Assert(snapIDs, DeepEquals, tc.snapIDs)
		}
	}
}
