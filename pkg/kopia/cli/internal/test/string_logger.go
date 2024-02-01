// Copyright 2024 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"context"
	"io"
	"regexp"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

// StringLogger implements log.Logger and stores log messages in a slice of strings.
// It is useful for testing.
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
