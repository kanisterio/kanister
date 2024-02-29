package internal_test

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/opts"
)

func TestNewKopiaCommand(t *testing.T) { check.TestingT(t) }

type NewKopiaCommandSuite struct{}

var _ = check.Suite(&NewKopiaCommandSuite{})

func (s *PathSuite) TestNewKopiaCommandSuite(c *check.C) {
	cmd, err := internal.NewKopiaCommand(opts.JSON(true))
	c.Check(err, check.IsNil)
	c.Check(cmd.Build(), check.DeepEquals, []string{"kopia", "--json"})
}
