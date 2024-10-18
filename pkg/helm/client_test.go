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

package helm

import (
	"context"
	"testing"

	"gopkg.in/check.v1"
)

type ExecSuite struct {
	command string
	args    []string
	output  string
	err     bool
}

// Valid command
var _ = check.Suite(&ExecSuite{
	command: "echo",
	args:    []string{"success"},
	output:  "success",
})

// Invalid command
var _ = check.Suite(&ExecSuite{
	command: "invalid",
	err:     true,
})

// Check timeout
var _ = check.Suite(&ExecSuite{
	command: "sleep",
	args:    []string{"11m"},
	err:     true,
})

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

func (s *ExecSuite) TestRunCmdWithTimeout(c *check.C) {
	ctx := context.Background()
	out, err := RunCmdWithTimeout(ctx, s.command, s.args)
	if s.err {
		c.Assert(err, check.NotNil)
		return
	}
	c.Assert(err, check.IsNil)
	c.Assert(out, check.Equals, s.output)
}
