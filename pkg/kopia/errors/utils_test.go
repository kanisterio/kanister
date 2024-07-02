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

package errors

import (
	"testing"

	"github.com/kanisterio/errkit"
	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func TestKopiaErrors(t *testing.T) { TestingT(t) }

type KopiaErrorsTestSuite struct{}

var _ = Suite(&KopiaErrorsTestSuite{})

// TestErrCheck verifies that error types are properly detected after wrapping them
func (s *KopiaErrorsTestSuite) TestErrCheck(c *C) {
	origErr := errors.New("Some error")

	errWithMessage := errors.WithMessage(origErr, ErrInvalidPasswordStr)
	errWrapped := errkit.Wrap(origErr, ErrInvalidPasswordStr)

	c.Assert(IsInvalidPasswordError(errWithMessage), Equals, true)
	c.Assert(IsInvalidPasswordError(errWrapped), Equals, true)
	c.Assert(IsRepoNotFoundError(errWrapped), Equals, false)

	permittedErrors := []ErrorType{ErrorInvalidPassword, ErrorRepoNotFound}
	c.Assert(CheckKopiaErrors(errWithMessage, permittedErrors), Equals, true)
	c.Assert(CheckKopiaErrors(errWrapped, permittedErrors), Equals, true)

	wrongErrors := []ErrorType{ErrorRepoNotFound}
	c.Assert(CheckKopiaErrors(errWrapped, wrongErrors), Equals, false)
}
