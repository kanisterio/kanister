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
	"errors"
)

type UserMessenger interface {
	error
	UserMessage() string
}

// UserError is an error type that be checked to be present in the
// error chain. And if present, can be shown to the end users
type UserError struct {
	error
	msg string
}

func (ue *UserError) UserMessage() string {
	return ue.msg
}

func (ue *UserError) Unwrap() error { return ue.error }

// UserErrorWithMessage initializes the UserError with given error and a message.
// Needs to accept error if we want to initialize the UserError in between the
// error chain
func UserErrorWithMessage(err error, msg string) UserMessenger {
	if err == nil {
		err = errors.New(msg)
	}
	return &UserError{err, msg}
}

// UserMessagesInError gets us messages of all the `UserError`s that are wrapped
// in the passed error's chain
func UserMessagesInError(err error) []string {
	userMessages := []string{}
	if err == nil {
		return nil
	}
	for err != nil {
		if e, ok := err.(*UserError); ok {
			userMessages = append(userMessages, e.UserMessage())
		}
		err = errors.Unwrap(err)
	}
	return userMessages
}
