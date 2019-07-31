package envdir

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type EnvDirSuite struct{}

var _ = Suite(&EnvDirSuite{})

func (s *EnvDirSuite) TestEnvDir(c *C) {
	d := c.MkDir()
	p := filepath.Join(d, "FOO")
	err := ioutil.WriteFile(p, []byte("BAR"), os.ModePerm)
	c.Assert(err, IsNil)
	e, err := EnvDir(d)
	c.Assert(err, IsNil)
	c.Assert(e, DeepEquals, []string{"FOO=BAR"})
}
