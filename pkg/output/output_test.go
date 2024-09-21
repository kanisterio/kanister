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
	"testing"

	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type OutputSuite struct{}

var _ = check.Suite(&OutputSuite{})

func (s *OutputSuite) TestValidateKey(c *check.C) {
	for _, tc := range []struct {
		key     string
		checker check.Checker
	}{
		{"validKey", check.IsNil},
		{"validKey2", check.IsNil},
		{"valid_key", check.IsNil},
		{"invalid-key", check.NotNil},
		{"invalid.key", check.NotNil},
		{"`invalidKey", check.NotNil},
	} {
		err := ValidateKey(tc.key)
		c.Check(err, tc.checker, check.Commentf("Key (%s) failed!", tc.key))
	}
}
