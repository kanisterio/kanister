package repository

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kopia/kopia/repo"
)

func Test(t *testing.T) { check.TestingT(t) }

type RepositoryClientSuite struct{}

var _ = check.Suite(&RepositoryClientSuite{})

func (s *RepositoryClientSuite) TestKopiaAllowedAlgorithms(c *check.C) {
	pbkdf2Supported := false
	for _, algorithm := range repo.SupportedLocalCacheKeyDerivationAlgorithms() {
		if algorithm == pbkdf2Algorithm {
			pbkdf2Supported = true
		}
	}
	c.Assert(pbkdf2Supported, check.Equals, true)
}
