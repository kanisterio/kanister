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

package envdir

import (
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type EnvDirSuite struct{}

var _ = Suite(&EnvDirSuite{})

func (s *EnvDirSuite) TestEnvDir(c *C) {
	d := c.MkDir()
	p := filepath.Join(d, "FOO")
	err := os.WriteFile(p, []byte("BAR"), os.ModePerm)
	c.Assert(err, IsNil)
	e, err := EnvDir(d)
	c.Assert(err, IsNil)
	c.Assert(e, DeepEquals, []string{"FOO=BAR"})
}
