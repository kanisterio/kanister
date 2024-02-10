package maintenance

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestMaintenanceFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty CustomerOwner should not generate a flag",
		Flag: CustomerOwner(""),
	},
	{
		Name:        "CustomerOwner with value should generate a flag with given value",
		Flag:        CustomerOwner("owner"),
		ExpectedCLI: []string{"--owner=owner"},
	},
}))
