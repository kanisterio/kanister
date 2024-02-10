package server

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestBash(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "Empty",
		CLI: func() (safecli.CommandBuilder, error) {
			cmd := safecli.NewBuilder("ls", "-la")
			bash := NewBashBuilder(cmd)
			return bash, nil
		},
		ExpectedCLI: []string{"bash", "-o", "errexit", "-c", "ls", "-la"},
	},
}))
