// Copyright 2022 The Kanister Authors.
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

package errors

import (
	"bytes"

	"github.com/pkg/errors"
)

type errorList []error

var _ error = errorList{}

func (e errorList) String() string {
	sep := ""
	var buf bytes.Buffer
	buf.WriteRune('[')
	for _, err := range e {
		buf.WriteString(sep)
		sep = ","
		buf.WriteRune('"')
		buf.WriteString(err.Error())
		buf.WriteRune('"')
	}
	buf.WriteRune(']')
	return buf.String()
}

func (e errorList) Error() string {
	return e.String()
}

// Append creates a new combined error from err1, err2. If either error is nil,
// then the other error is returned.
func Append(err1, err2 error) error {
	if err1 == nil {
		return err2
	}
	if err2 == nil {
		return err1
	}
	el1, ok1 := err1.(errorList)
	el2, ok2 := err2.(errorList)
	switch {
	case ok1 && ok2:
		return append(el1, el2...)
	case ok1:
		return append(el1, err2)
	case ok2:
		return append(el2, err1)
	}
	return errorList{err1, err2}
}

// FirstMatching returns the first error that matches the predicate in a
// causal dependency err->Cause()->Cause() ....
func FirstMatching(err error, predicate func(error) bool) error {
	for ; err != nil; err = errors.Unwrap(err) {
		if predicate(err) {
			return err
		}
	}
	return nil
}
