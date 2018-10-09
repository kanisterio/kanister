package output

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type OutputSuite struct{}

var _ = Suite(&OutputSuite{})

func (s *OutputSuite) TestValidateKey(c *C) {
	for _, tc := range []struct {
		key     string
		checker Checker
	}{
		{"validKey", IsNil},
		{"validKey2", IsNil},
		{"valid_key", IsNil},
		{"invalid-key", NotNil},
		{"invalid.key", NotNil},
		{"`invalidKey", NotNil},
	} {
		err := ValidateKey(tc.key)
		c.Check(err, tc.checker, Commentf("Key (%s) failed!", tc.key))
	}
}
