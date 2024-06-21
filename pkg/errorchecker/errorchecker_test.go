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

func (ts *ErrorsTestSuite) TestErrorMessageMatcher(c *C) {
	errkitError := errkit.New("Some errkit error")
	for _, tc := range []struct {
		params         []any
		names          []string
		expectedResult bool
		expectedError  string
	}{
		{
			params:         []any{nil},
			names:          []string{},
			expectedResult: false,
			expectedError:  "Error value is nil",
		},
		{
			params:         []any{10},
			names:          []string{},
			expectedResult: false,
			expectedError:  "Value is not an error",
		},
		{
			params:         []any{errkitError, ".* errkit .*"},
			names:          []string{""},
			expectedResult: true,
			expectedError:  "",
		},
		{
			params:         []any{errkitError, 1},
			names:          []string{""},
			expectedResult: false,
			expectedError:  "Regex must be a string",
		},
	} {
		r, s := ErrorMessageMatcher.Check(tc.params, tc.names)
		c.Assert(r, Equals, tc.expectedResult)
		c.Assert(s, Equals, tc.expectedError)
	}
}

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
	c.Assert(errorsWrappedErr1, ErrorMessageMatcher, ".*errkit wrapped.*")
	c.Assert(errorsWrappedErr1, ErrorMessageMatcher, ".*Errkit error")
	c.Assert(errorsWrappedErr1, ErrorMessageMatcher, "errors wrapped 1.*")
}
