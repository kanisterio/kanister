// +build blueprints_test

package blueprints

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type BlueprintsSuite struct{}

var _ = Suite(&BlueprintsSuite{})

func (s *BlueprintsSuite) TestEnsureNoMissingBlueprints(c *C) {
	// Ensure there are no extraneous .yaml symlinks present (not in blueprintPaths)
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	c.Assert(err, IsNil)

	missingBlueprintPaths := make(map[string]string)
	for _, file := range files {
		lstat, err := os.Lstat(file)
		c.Assert(err, IsNil)
		// ignore non-symlink files
		if lstat.Mode()&os.ModeSymlink == os.ModeSymlink {
			blueprint := filepath.Base(file)
			link, err := os.Readlink(file)
			c.Assert(err, IsNil)

			if _, present := blueprintPaths[blueprint]; !present {
				missingBlueprintPaths[blueprint] = link
			}
		}
	}

	if len(missingBlueprintPaths) > 0 {
		c.Log("The following symlinks are missing from blueprintPaths:")
		for blueprint, path := range missingBlueprintPaths {
			c.Logf("\t%q: %q,", blueprint, path)
		}
		c.Fail()
	}
}

func (s *BlueprintsSuite) TestBlueprintPathsMatchSymlinks(c *C) {
	// Ensure all of the paths in blueprintPaths match the symlinks
	for blueprint, path := range blueprintPaths {
		link, err := os.Readlink(blueprint)
		c.Assert(err, IsNil)
		c.Assert(path, Equals, link)
	}
}

func (s *BlueprintsSuite) TestPathFor(c *C) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	for blueprint, _ := range blueprintPaths {
		directStat, err := os.Stat(filepath.Join(dir, blueprint))
		c.Assert(err, IsNil)

		path, err := PathFor(blueprint)
		c.Assert(err, IsNil)
		pathForStat, err := os.Stat(path)

		if !os.SameFile(directStat, pathForStat) {
			c.Errorf("PathFor returned %q, which doesn't match the symlink for %q", path, blueprint)
		}
	}
}
