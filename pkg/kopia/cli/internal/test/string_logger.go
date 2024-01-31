package test

import (
	"context"
	"io"
	"regexp"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type StringLogger []string

func (l *StringLogger) Print(msg string, fields ...field.M) {
	*l = append(*l, msg)
}

func (l *StringLogger) PrintTo(w io.Writer, msg string, fields ...field.M) {
	*l = append(*l, msg)
}

func (l *StringLogger) WithContext(ctx context.Context) log.Logger {
	return l
}

func (l *StringLogger) WithError(err error) log.Logger {
	return l
}

func (l *StringLogger) MatchString(pattern string) bool {
	for _, line := range *l {
		if found, _ := regexp.MatchString(pattern, line); found {
			return true
		}
	}
	return false
}
