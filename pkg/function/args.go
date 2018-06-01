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

// OptArg returns the value of the specified argument if it exists
// It will return the default value if the argument does not exist
func OptArg(args map[string]interface{}, argName string, result interface{}, defaultValue interface{}) error {
	if _, ok := args[argName]; ok {
		return Arg(args, argName, result)
	}
	return mapstructure.Decode(defaultValue, result)
}

// ArgPresent checks if the argument exists
func ArgExists(args map[string]interface{}, argName string) bool {
	_, ok := args[argName]
	return ok
}
