package kanctl

import (
	"fmt"

	"github.com/pkg/errors"
)

var argsLengthErr = fmt.Errorf("Incorrect number of arguments")

func newArgsLengthError(format string, args ...interface{}) error {
	return errors.Wrapf(argsLengthErr, format, args...)
}

// IsArgsLengthError returns true iff the underlying cause was an argsLengthErr.
func IsArgsLengthError(err error) bool {
	return errors.Cause(err) == argsLengthErr
}
