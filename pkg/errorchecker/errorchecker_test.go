package errorchecker

import (
	. "gopkg.in/check.v1"
	"testing"

	"github.com/kanisterio/errkit"
	"github.com/pkg/errors"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ErrorsTestSuite struct{}

var _ = Suite(&ErrorsTestSuite{})

func (ts *ErrorsTestSuite) Test(c *C) {
	errkitErr := errkit.New("Errkit error")
	errkitWrappedErr := errkit.Wrap(errkitErr, "errkit wrapped")
	errorsWrappedErr := errors.Wrap(errkitWrappedErr, "errors wrapped")
	errorsWrappedErr1 := errors.Wrap(errorsWrappedErr, "errors wrapped 1")

	// Ensure that errors from 'errkit' wrapped by the older 'errors' package remain matchable.
	c.Assert(errors.Is(errorsWrappedErr, errkitErr), Equals, true)
	// Ensure that transformation to string still works
	c.Assert(errorsWrappedErr1.Error(), Equals, "errors wrapped 1: errors wrapped: errkit wrapped: Errkit error")
	// Ensure that error message matching does work as expected
	c.Assert(errorsWrappedErr1, ErrorMessageMatcher, ".*errkit wrapped.*")
	c.Assert(errorsWrappedErr1, ErrorMessageMatcher, ".*Errkit error")
	c.Assert(errorsWrappedErr1, ErrorMessageMatcher, "errors wrapped 1.*")
}
