package model

import (
	"testing"

	"gopkg.in/check.v1"
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
		got := GenerateFullRepoPath(test.locPrefix, test.repoPathPrefix)
		c.Check(got, check.Equals, test.expected)
	}
}
