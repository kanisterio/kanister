package errorchecker

import (
	"testing"

	"github.com/kanisterio/errkit"
	"github.com/pkg/errors"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type ErrorsTestSuite struct{}

var _ = check.Suite(&ErrorsTestSuite{})

func (ts *ErrorsTestSuite) TestWrappingAndMatching(c *check.C) {
	errkitErr := errkit.New("Errkit error")
	errkitWrappedErr := errkit.Wrap(errkitErr, "errkit wrapped")
	errorsWrappedErr := errors.Wrap(errkitWrappedErr, "errors wrapped")
	errorsWrappedErr1 := errors.Wrap(errorsWrappedErr, "errors wrapped 1")

	// Ensure that errors from 'errkit' wrapped by the older 'errors' package remain matchable.
	c.Assert(errors.Is(errorsWrappedErr, errkitErr), check.Equals, true)
	// Ensure that transformation to string still works
	c.Assert(errorsWrappedErr1.Error(), check.Equals, "errors wrapped 1: errors wrapped: errkit wrapped: Errkit error")
	// Ensure that error message matching does work as expected
	AssertErrorMessage(c, errorsWrappedErr1, ".*errkit wrapped.*")
	AssertErrorMessage(c, errorsWrappedErr1, ".*Errkit error")
	AssertErrorMessage(c, errorsWrappedErr1, "errors wrapped 1.*")
}
