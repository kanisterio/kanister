package errkitchecker

import (
	"fmt"
	"regexp"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
)

type errkitMatchesChecker struct {
	*check.CheckerInfo
}

var ErrkitErrorMatches check.Checker = errkitMatchesChecker{
	&check.CheckerInfo{Name: "ErrorMatches", Params: []string{"value", "regex"}},
}

func (checker errkitMatchesChecker) Check(
	params []interface{},
	names []string,
) (result bool, errStr string) {
	if params[0] == nil {
		return false, "Error value is nil"
	}
	err, ok := params[0].(*errkit.Error)
	if !ok {
		return false, "Value is not an error"
	}
	params[0] = err.Message()
	names[0] = "error"
	return matches(params[0], params[1])
}

func matches(value, regex interface{}) (result bool, error string) {
	reStr, ok := regex.(string)
	if !ok {
		return false, "Regex must be a string"
	}
	valueStr, valueIsStr := value.(string)
	if !valueIsStr {
		if valueWithStr, valueHasStr := value.(fmt.Stringer); valueHasStr {
			valueStr, valueIsStr = valueWithStr.String(), true
		}
	}
	if valueIsStr {
		matches, err := regexp.MatchString("^"+reStr+"$", valueStr)
		if err != nil {
			return false, "Can't compile regex: " + err.Error()
		}
		return matches, ""
	}
	return false, "Obtained value is not a string and has no .String()"
}
