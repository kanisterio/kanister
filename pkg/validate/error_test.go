package validate

import (
	"fmt"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
)

type ErrorSuite struct{}

var _ = Suite(&ErrorSuite{})

func (s *ErrorSuite) TestIsError(c *C) {
	for _, tc := range []struct {
		err error
		is  bool
	}{
		{
			err: nil,
			is:  false,
		},
		{
			err: fmt.Errorf("test error"),
			is:  false,
		},
		{
			err: validateErr,
			is:  true,
		},
		{
			err: errors.Wrap(nil, "test"),
			is:  false,
		},
		{
			err: errors.WithStack(nil),
			is:  false,
		},
		{
			err: errors.Wrap(validateErr, "test"),
			is:  true,
		},
		{
			err: errors.WithStack(validateErr),
			is:  true,
		},
		{
			err: errors.New("test"),
			is:  false,
		},
	} {
		c.Check(IsError(tc.err), Equals, tc.is)
	}

}
