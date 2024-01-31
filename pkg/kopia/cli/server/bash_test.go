package server

import (
	"strings"
	"testing"

	"github.com/kanisterio/kanister/pkg/safecli"
	"gopkg.in/check.v1"
)

func TestBash(t *testing.T) { check.TestingT(t) }

type BashSuite struct{}

var _ = check.Suite(&BashSuite{})

func (s *BashSuite) TestBashBuilder(c *check.C) {
	expectedCLI := []string{
		"bash", "-o", "errexit", "-c", "ls", "-la",
	}
	expectedLog := strings.Join(expectedCLI, " ")

	cmd := safecli.NewBuilder("ls", "-la")
	bash := NewBashBuilder(cmd)
	c.Check(bash.Build(), check.DeepEquals, expectedCLI)
	c.Check(bash.String(), check.DeepEquals, expectedLog)
}
