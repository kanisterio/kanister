package validate

import (
	"fmt"

	"github.com/pkg/errors"
)

var validateErr = fmt.Errorf("Validation Failed")

func errorf(format string, args ...interface{}) error {
	return errors.Wrapf(validateErr, format, args...)
}

// IsError returns true iff the underlying cause was a validation error.
func IsError(err error) bool {
	return errors.Cause(err) == validateErr
}
