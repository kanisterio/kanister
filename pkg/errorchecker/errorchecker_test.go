package errorchecker

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/errkit"
	"github.com/pkg/errors"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ErrorsTestSuite struct{}

var _ = Suite(&ErrorsTestSuite{})

func (ts *ErrorsTestSuite) TestWrappingAndMatching(c *C) {
	errkitErr := errkit.New("Errkit error")
	errkitWrappedErr := errkit.Wrap(errkitErr, "errkit wrapped")
	errorsWrappedErr := errors.Wrap(errkitWrappedErr, "errors wrapped")
	errorsWrappedErr1 := errors.Wrap(errorsWrappedErr, "errors wrapped 1")

	// Ensure that errors from 'errkit' wrapped by the older 'errors' package remain matchable.
	c.Assert(errors.Is(errorsWrappedErr, errkitErr), Equals, true)
	// Ensure that transformation to string still works
	c.Assert(errorsWrappedErr1.Error(), Equals, "errors wrapped 1: errors wrapped: errkit wrapped: Errkit error")
	// Ensure that error message matching does work as expected
	AssertErrorMessage(c, errorsWrappedErr1, ".*errkit wrapped.*")
	AssertErrorMessage(c, errorsWrappedErr1, ".*Errkit error")
	AssertErrorMessage(c, errorsWrappedErr1, "errors wrapped 1.*")
}
