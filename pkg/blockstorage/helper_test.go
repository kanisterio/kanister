package blockstorage

import (
	. "gopkg.in/check.v1"
)

type HelperSuite struct{}

var _ = Suite(&HelperSuite{})

func (s *HelperSuite) SetUpSuite(c *C) {
}

func (h *HelperSuite) TestStringSlice(c *C) {
	source := []string{"test1", "test2"}
	target := StringSlice(&source)
	c.Assert(target[0], Equals, source[0])
	c.Assert(target[1], Equals, source[1])
}

func (s *HelperSuite) TestStringSlicePtr(c *C) {
	source := []string{"test1", "test2"}
	res := StringSlicePtr(source)
	target := *res
	c.Assert(target[0], Equals, source[0])
	c.Assert(target[1], Equals, source[1])
}

func (s *HelperSuite) TestSliceStringPtr(c *C) {
	source := []string{"test1", "test2"}
	res := SliceStringPtr(source)
	for i, elePtr := range res {
		var target = *elePtr
		c.Assert(target, Equals, source[i])
	}
}

func (s *HelperSuite) TestBoolFromPtr(c *C) {
	source := true
	target := Bool(&source)
	c.Assert(target, Equals, source)
}

func (s *HelperSuite) TestBoolToPtr(c *C) {
	source := true
	target := BoolPtr(source)
	c.Assert(*target, Equals, source)
}

func (s *HelperSuite) TestIntFromPtr(c *C) {
	source := 1
	target := Int(&source)
	c.Assert(target, Equals, source)
}

func (s *HelperSuite) TestIntToPtr(c *C) {
	source := 1
	target := IntPtr(source)
	c.Assert(*target, Equals, source)
}

func (s *HelperSuite) TestFloat32FromPtr(c *C) {
	source := float32(1)
	target := Float32(&source)
	c.Assert(target, Equals, source)
}

func (s *HelperSuite) TestFloat32ToPtr(c *C) {
	source := float32(1)
	target := Float32Ptr(source)
	c.Assert(*target, Equals, source)
}

func (s *HelperSuite) TestStringFromPtr(c *C) {
	source := "test"
	target := String(&source)
	c.Assert(target, Equals, source)
}

func (s *HelperSuite) TestStringToPtr(c *C) {
	source := "test"
	target := StringPtr(source)
	c.Assert(*target, Equals, source)
}
