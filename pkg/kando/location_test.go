package kando

import (
	"bytes"
	"context"
	"path/filepath"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/testutil"
)

type LocationSuite struct{}

var _ = Suite(&LocationSuite{})

const testContent = "test-content"

func (s *LocationSuite) TestLocationObjectStore(c *C) {
	p := testutil.ObjectStoreProfileOrSkip(c)
	ctx := context.Background()
	dir := c.MkDir()
	path := filepath.Join(dir, "test-object1.txt")

	source := bytes.NewBufferString(testContent)
	err := locationPush(ctx, p, path, source)
	c.Assert(err, IsNil)

	target := bytes.NewBuffer(nil)
	err = locationPull(ctx, p, path, target)
	c.Assert(err, IsNil)
	c.Assert(target.String(), Equals, testContent)

	// test deleting single artifact
	err = locationDelete(ctx, p, path)
	c.Assert(err, IsNil)

	//test deleting dir with multiple artifacts
	source = bytes.NewBufferString(testContent)
	err = locationPush(ctx, p, path, source)
	c.Assert(err, IsNil)

	path = filepath.Join(dir, "test-object2.txt")

	source = bytes.NewBufferString(testContent)
	err = locationPush(ctx, p, path, source)
	c.Assert(err, IsNil)

	err = locationDelete(ctx, p, dir)
	c.Assert(err, IsNil)

}
