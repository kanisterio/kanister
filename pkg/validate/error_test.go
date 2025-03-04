// Copyright 2019 The Kanister Authors.
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

package validate

import (
	"fmt"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
)

type ErrorSuite struct{}

var _ = check.Suite(&ErrorSuite{})

func (s *ErrorSuite) TestIsError(c *check.C) {
	for _, tc := range []struct {
		err error
		is  bool
	}{
		{
			err: nil,
			is:  false,
		},
		{
			err: fmt.Errorf("test error"),
			is:  false,
		},
		{
			err: errValidate,
			is:  true,
		},
		{
			err: errkit.Wrap(nil, "test"),
			is:  false,
		},
		{
			err: errkit.WithStack(nil),
			is:  false,
		},
		{
			err: errkit.Wrap(errValidate, "test"),
			is:  true,
		},
		{
			err: errkit.WithStack(errValidate),
			is:  true,
		},
		{
			err: errkit.New("test"),
			is:  false,
		},
	} {
		c.Check(IsError(tc.err), check.Equals, tc.is)
	}
}
