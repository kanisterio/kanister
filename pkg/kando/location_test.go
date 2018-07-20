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
	path := filepath.Join(c.MkDir(), "test-object.txt")

	source := bytes.NewBufferString(testContent)
	err := locationPush(ctx, p, path, source)
	c.Assert(err, IsNil)

	target := bytes.NewBuffer(nil)
	err = locationPull(ctx, p, path, target)
	c.Assert(err, IsNil)
	c.Assert(target.String(), Equals, testContent)
}
