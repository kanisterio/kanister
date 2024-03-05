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
)

func TestPath(t *testing.T) { check.TestingT(t) }

type PathSuite struct{}

var _ = check.Suite(&PathSuite{})

func (s *PathSuite) TestGenerateFullRepoPath(c *check.C) {
	tests := []struct {
		locPrefix      string
		repoPathPrefix string
		expected       string
	}{
		{
			locPrefix:      "",
			repoPathPrefix: "repo",
			expected:       "repo",
		},
		{
			locPrefix:      "loc",
			repoPathPrefix: "repo",
			expected:       "loc/repo/",
		},
	}
	for _, test := range tests {
		got := internal.GenerateFullRepoPath(test.locPrefix, test.repoPathPrefix)
		c.Check(got, check.Equals, test.expected)
	}
}
