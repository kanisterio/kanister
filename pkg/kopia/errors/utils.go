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
	"regexp"
	"strings"

	"github.com/kanisterio/errkit"
)

// FirstMatching returns the first error that matches the predicate in a
// causal dependency err->Cause()->Cause() ....
func FirstMatching(err error, predicate func(error) bool) error {
	for ; err != nil; err = errkit.Unwrap(err) {
		if predicate(err) {
			return err
		}
	}
	return nil
}

type ErrorType string

const (
	ErrorInvalidPassword ErrorType = ErrInvalidPasswordStr
	ErrorRepoNotFound    ErrorType = ErrRepoNotFoundStr
)

// CheckKopiaErrors loops through all the permitted
// error types and returns true on finding a match
func CheckKopiaErrors(err error, errorTypes []ErrorType) bool {
	for _, errorType := range errorTypes {
		if checkKopiaError(err, errorType) {
			return true
		}
	}
	return false
}

func checkKopiaError(err error, errorType ErrorType) bool {
	switch errorType {
	case ErrorInvalidPassword:
		return IsInvalidPasswordError(err)
	case ErrorRepoNotFound:
		return IsRepoNotFoundError(err)
	default:
		return false
	}
}

// IsInvalidPasswordError returns true if the error chain has `invalid repository password` error
func IsInvalidPasswordError(err error) bool {
	return FirstMatching(err, func(err error) bool {
		return strings.Contains(err.Error(), ErrInvalidPasswordStr)
	}) != nil
}

// IsRepoNotFoundError returns true if the error contains `repository not found` message
func IsRepoNotFoundError(err error) bool {
	return FirstMatching(err, func(err error) bool {
		return strings.Contains(err.Error(), ErrRepoNotFoundStr)
	}) != nil
}

// RepoNotInitialized returns true if the stderr logs contains `repository not initialized` for object stores
// or `no such file or directory` for filestore backend
func RepoNotInitialized(stderr string) bool {
	return strings.Contains(stderr, ErrRepoNotInitializedStr) || strings.Contains(stderr, ErrFilesystemRepoNotFoundStr)
}

var regexpBucketDoesNotExist = regexp.MustCompile(`bucket ".*" does not exist`)

// BucketDoesNotExist returns true if the stderr logs contain either `bucket doesn't exist`
// or `bucket "<bucket_name>" does not exist` messages.
func BucketDoesNotExist(stderr string) bool {
	return strings.Contains(stderr, ErrBucketDoesNotExistStr) ||
		strings.Contains(stderr, ErrUnableToListFromBucketStr) ||
		len(regexpBucketDoesNotExist.FindString(stderr)) != 0
}
