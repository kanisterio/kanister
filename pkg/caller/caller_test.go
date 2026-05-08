package caller

import (
	"testing"

	"gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type CallerSuite struct{}

var _ = check.Suite(&CallerSuite{})

func (s *CallerSuite) TestCallerFields(c *check.C) {
	frame := GetFrame(0)
	// Check against the GetFrame() function
	c.Assert(frame.Function, check.Equals, "github.com/kanisterio/kanister/pkg/caller.GetFrame")
	// This breaks if the function is relocated or the file is renamed or the
	// this test is run independently
	c.Assert(frame.File, check.Equals, "pkg/caller/frame.go")

	frame = indirectCall(1)
	// Check against the indirectCall() function
	c.Assert(frame.Function, check.Equals, "github.com/kanisterio/kanister/pkg/caller.indirectCall")
	// This breaks if the function is relocated or the file is renamed
	c.Assert(frame.File, check.Equals, "pkg/caller/caller_test.go")

	frame = indirectCall(2)
	// Check against this function
	c.Assert(frame.Function, check.Equals, "github.com/kanisterio/kanister/pkg/caller.(*CallerSuite).TestCallerFields")
	// This breaks if the function is relocated or the file is renamed
	c.Assert(frame.File, check.Equals, "pkg/caller/caller_test.go")
}

func indirectCall(depth int) Frame {
	return GetFrame(depth)
}
