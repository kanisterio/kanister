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
