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
	"strings"
	"testing"

	"gopkg.in/check.v1"
)

func TestRedactCLI(t *testing.T) { check.TestingT(t) }

type RedactSuite struct{}

var _ = check.Suite(&RedactSuite{})

func (s *RedactSuite) TestRedactCLI(c *check.C) {
	cli := []string{
		"--password=secret",
		"--user-password=123456",
		"--server-password=pass123",
		"--server-control-password=abc123",
		"--server-cert-fingerprint=abcd1234",
		"--other-flag=value",
		"argument",
	}
	expected := []string{
		"--password=<****>",
		"--user-password=<****>",
		"--server-password=<****>",
		"--server-control-password=<****>",
		"--server-cert-fingerprint=<****>",
		"--other-flag=value",
		"argument",
	}
	result := RedactCLI(cli)
	c.Assert(result, check.Equals, strings.Join(expected, " "))
}
