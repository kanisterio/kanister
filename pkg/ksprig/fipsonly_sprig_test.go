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

package ksprig_test

import (
	"errors"
	"strings"
	"testing"
	"text/template"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/ksprig"
)

type FipsOnlySprigSuite struct{}

var _ = Suite(&FipsOnlySprigSuite{})

func TestFipsOnlySprigSuite(t *testing.T) { TestingT(t) }

func (f *FipsOnlySprigSuite) TestUnsupportedTxtFuncMapUsage(c *C) {
	funcMap := ksprig.TxtFuncMap()

	testCases := []struct {
		function     string
		templateText string
		usageErr     string
	}{
		{
			function:     "bcrypt",
			templateText: "{{bcrypt \"password\"}}",
			usageErr:     "bcrypt",
		},
		{
			function:     "derivePassword",
			templateText: "{{derivePassword 1 \"long\" \"password\" \"user\" \"example.com\"}}",
			usageErr:     "derivePassword",
		},
		{
			function:     "genPrivateKey",
			templateText: "{{genPrivateKey \"dsa\"}}",
			usageErr:     "genPrivateKey for dsa",
		},
		{
			function:     "htpasswd",
			templateText: "{{htpasswd \"username\" \"password\"}}",
			usageErr:     "htpasswd",
		},
	}

	for _, tc := range testCases {
		if _, ok := funcMap[tc.function]; !ok {
			c.Logf("Skipping test of %s since the tested sprig version does not support it", tc.function)
			continue
		}
		c.Logf("Testing %s", tc.function)

		temp, err := template.New("test").Funcs(funcMap).Parse(tc.templateText)
		c.Assert(err, IsNil)

		err = temp.Execute(nil, "")

		var sprigErr ksprig.UnsupportedSprigUsageErr
		c.Assert(errors.As(err, &sprigErr), Equals, true)
		c.Assert(sprigErr.Usage, Equals, tc.usageErr)
	}
}

func (f *FipsOnlySprigSuite) TestSupportedTxtFuncMapUsage(c *C) {
	funcMap := ksprig.TxtFuncMap()

	testCases := []struct {
		description  string
		function     string
		templateText string
	}{
		// The supported usages are not limited to these test cases
		{
			description:  "genPrivateKey for rsa key",
			function:     "genPrivateKey",
			templateText: "{{genPrivateKey \"rsa\"}}",
		},
		{
			description:  "genPrivateKey for ecdsa key",
			function:     "genPrivateKey",
			templateText: "{{genPrivateKey \"ecdsa\"}}",
		},
		{
			description:  "genPrivateKey for ed25519 key",
			function:     "genPrivateKey",
			templateText: "{{genPrivateKey \"ed25519\"}}",
		},
	}

	for _, tc := range testCases {
		if _, ok := funcMap[tc.function]; !ok {
			c.Logf("Skipping test of %s since the tested sprig version does not support it", tc.function)
			continue
		}
		c.Logf("Testing %s", tc.description)

		temp, err := template.New("test").Funcs(funcMap).Parse(tc.templateText)
		c.Assert(err, IsNil)

		err = temp.Execute(&strings.Builder{}, "")
		c.Assert(err, IsNil)
	}
}
