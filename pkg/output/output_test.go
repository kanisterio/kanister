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

package output

import (
	"bytes"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type OutputSuite struct{}

var _ = Suite(&OutputSuite{})

func (s *OutputSuite) TestValidateKey(c *C) {
	for _, tc := range []struct {
		key     string
		checker Checker
	}{
		{"validKey", IsNil},
		{"validKey2", IsNil},
		{"valid_key", IsNil},
		{"invalid-key", NotNil},
		{"invalid.key", NotNil},
		{"`invalidKey", NotNil},
	} {
		err := ValidateKey(tc.key)
		c.Check(err, tc.checker, Commentf("Key (%s) failed!", tc.key))
	}
}

func (s *OutputSuite) TestParseValid(c *C) {
	key, val := "foo", "bar"
	b := bytes.NewBuffer(nil)
	err := fPrintOutput(b, key, val)
	c.Check(err, IsNil)

	o, err := Parse(b.String())
	c.Assert(err, IsNil)
	c.Assert(o, NotNil)
	c.Assert(o.Key, Equals, key)
	c.Assert(o.Value, Equals, val)
}

func (s *OutputSuite) TestParseNoOutput(c *C) {
	key, val := "foo", "bar"
	b := bytes.NewBuffer(nil)
	err := fPrintOutput(b, key, val)
	c.Check(err, IsNil)
	valid := b.String()
	for _, tc := range []struct {
		s       string
		checker Checker
	}{
		{
			s:       valid[0 : len(valid)-2],
			checker: NotNil,
		},
		{
			s:       valid[1 : len(valid)-1],
			checker: IsNil,
		},
	} {
		o, err := Parse(tc.s)
		c.Assert(err, tc.checker)
		c.Assert(o, IsNil)
	}
}
