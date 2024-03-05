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

package internal_test

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/opts"
)

func TestNewKopiaCommand(t *testing.T) { check.TestingT(t) }

type KopiaCommandSuite struct{}

var _ = check.Suite(&KopiaCommandSuite{})

func (s *KopiaCommandSuite) TestNewKopiaCommandSuite(c *check.C) {
	cmd, err := internal.NewKopiaCommand(opts.JSON(true))
	c.Check(err, check.IsNil)
	c.Check(cmd.Build(), check.DeepEquals, []string{"kopia", "--json"})
}
