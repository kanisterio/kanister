package function

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// Arg returns the value of the specified argument
// It will return an error if the argument type does not match the result type
func Arg(args map[string]interface{}, argName string, result interface{}) error {
	if val, ok := args[argName]; ok {
		if err := mapstructure.Decode(val, result); err != nil {
			return errors.Wrapf(err, "Failed to decode arg `%s`", argName)
		}
		return nil
	}
	return errors.New("Argument missing " + argName)
}
