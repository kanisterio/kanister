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
	"testing"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type UserErrorSuite struct{}

var _ = Suite(&UserErrorSuite{})

func (u *UserErrorSuite) TestUserMessagesInError(c *C) {
	expectedUserErrors := []string{"User Error Two", "User Error One"}
	err1 := errors.New("error 1")
	err2 := errors.Wrap(err1, "error2")

	ue1 := UserErrorWithMessage(err2, expectedUserErrors[1])
	err3 := errors.Wrap(ue1, "errro 3")

	ue2 := UserErrorWithMessage(err3, expectedUserErrors[0])
	err4 := errors.Wrap(ue2, "error 4")

	c.Assert(UserMessagesInError(err4), DeepEquals, expectedUserErrors)
}

// The case where there are no `UserError`s in error chain
func (u *UserErrorSuite) TestUserMessagesInErrorWithoutUserError(c *C) {
	err1 := errors.New("error 1")
	err2 := errors.Wrap(err1, "error 2")
	err3 := errors.Wrap(err2, "error 3")
	err4 := errors.Wrap(err3, "error 4")

	c.Assert(len(UserMessagesInError(err4)), Equals, 0)
}

func (u *UserErrorSuite) TestUserMessagesInErrorNilError(c *C) {
	c.Assert(UserMessagesInError(nil), Equals, nil)
}
