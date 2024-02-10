package policy

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestCompressionAlgorithm(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name:        "Empty CompressionAlgorithm should generate a flag with default value",
		Flag:        CompressionAlgorithm{},
		ExpectedCLI: []string{"--compression=s2-default"},
	},
	{
		Name: "CompressionAlgorithm with value should generate a flag with the given value",
		Flag: CompressionAlgorithm{
			CompressionAlgorithm: "gzip",
		},
		ExpectedCLI: []string{"--compression=gzip"},
	},
}))
