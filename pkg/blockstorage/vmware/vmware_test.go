package vmware

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/kanisterio/kanister/pkg/blockstorage"
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
			sort.Strings(snapIDs)
			sort.Strings(tc.snapIDs)
			c.Assert(snapIDs, DeepEquals, tc.snapIDs)
		}
	}
}

func (s *VMWareSuite) TestSetTagsSnapshot(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		catID         string
		snapshot      *blockstorage.Snapshot
		tags          map[string]string
		errChecker    Checker
		expNumCreates int

		errCreateTag error
	}{
		{ // success
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			tags: map[string]string{
				"t1": "v1",
				"t2": "v2",
			},
			expNumCreates: 2,
			errChecker:    IsNil,
		},
		{ // idempotent creates
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			tags: map[string]string{
				"t1": "v1",
				"t2": "v2",
			},
			expNumCreates: 2,
			errCreateTag:  fmt.Errorf("ALREADY_EXISTS"),
			errChecker:    IsNil,
		},
		{ // create failure
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			tags: map[string]string{
				"t1": "v1",
				"t2": "v2",
			},
			expNumCreates: 2,
			errCreateTag:  fmt.Errorf("bad create"),
			errChecker:    NotNil,
		},
		{ // malformed id
			catID:      "catid",
			snapshot:   &blockstorage.Snapshot{ID: "volidsnapid"},
			errChecker: NotNil,
		},
		{ // nil snapshot
			catID:      "catid",
			errChecker: NotNil,
		},
		{ // empty id, No error, not supported
			catID:      "",
			errChecker: IsNil,
		},
	} {
		ftm := &fakeTagManager{
			errCreateTag: tc.errCreateTag,
		}
		provider := &FcdProvider{
			categoryID: tc.catID,
			tagManager: ftm,
		}
		err := provider.setSnapshotTags(ctx, tc.snapshot, tc.tags)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			c.Assert(ftm.numCreates, Equals, tc.expNumCreates)
		}
	}
}

func (s *VMWareSuite) TestDeleteTagsSnapshot(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		catID         string
		snapshot      *blockstorage.Snapshot
		errChecker    Checker
		expNumDeletes int

		retGetTagsForCategory []vapitags.Tag
		errGetTagsForCategory error
		errDeleteTag          error
	}{
		{ // success deleting tags
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			retGetTagsForCategory: []vapitags.Tag{
				{Name: "volid:snapid:t1:v1"},
				{Name: "volid:snapid:t2:v2"},
				{Name: "volid:snapid2:t1:v1"},
			},
			expNumDeletes: 2,
			errChecker:    IsNil,
		},
		{ // error deleting tags
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			retGetTagsForCategory: []vapitags.Tag{
				{Name: "volid:snapid:t1:v1"},
				{Name: "volid:snapid:t2:v2"},
			},
			errDeleteTag: fmt.Errorf("Failed to delete tag"),
			errChecker:   NotNil,
		},
		{ // error parsing tags
			catID:    "catid",
			snapshot: &blockstorage.Snapshot{ID: "volid:snapid"},
			retGetTagsForCategory: []vapitags.Tag{
				{Name: "volid:snapidt1v1"},
				{Name: "volid:snapid:t2:v2"},
			},
			errChecker: NotNil,
		},
		{ // error fetching tags
			catID:                 "catid",
			snapshot:              &blockstorage.Snapshot{ID: "volid:snapid"},
			errGetTagsForCategory: fmt.Errorf("Failed to get tags"),
			errChecker:            NotNil,
		},
		{ // malformed id
			catID:      "catid",
			snapshot:   &blockstorage.Snapshot{ID: "volidsnapid"},
			errChecker: NotNil,
		},
		{ // nil snapshot
			catID:      "catid",
			errChecker: NotNil,
		},
		{ // empty id, No error, not supported
			catID:      "",
			errChecker: IsNil,
		},
	} {
		ftm := &fakeTagManager{
			retGetTagsForCategory: tc.retGetTagsForCategory,
			errGetTagsForCategory: tc.errGetTagsForCategory,
			errDeleteTag:          tc.errDeleteTag,
		}
		provider := &FcdProvider{
			categoryID: tc.catID,
			tagManager: ftm,
		}
		err := provider.deleteSnapshotTags(ctx, tc.snapshot)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			c.Assert(ftm.numDeletes, Equals, tc.expNumDeletes)
		}
	}
}

type fakeTagManager struct {
	retGetTagsForCategory []vapitags.Tag
	errGetTagsForCategory error

	numDeletes   int
	errDeleteTag error

	numCreates   int
	errCreateTag error
}

func (f *fakeTagManager) GetCategory(ctx context.Context, id string) (*vapitags.Category, error) {
	return nil, nil
}
func (f *fakeTagManager) CreateCategory(ctx context.Context, category *vapitags.Category) (string, error) {
	return "", nil
}
func (f *fakeTagManager) CreateTag(ctx context.Context, tag *vapitags.Tag) (string, error) {
	f.numCreates++
	return "", f.errCreateTag
}
func (f *fakeTagManager) GetTagsForCategory(ctx context.Context, id string) ([]vapitags.Tag, error) {
	return f.retGetTagsForCategory, f.errGetTagsForCategory
}
func (f *fakeTagManager) DeleteTag(ctx context.Context, tag *vapitags.Tag) error {
	f.numDeletes++
	return f.errDeleteTag
}
