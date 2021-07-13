// +build filesystem_tests

// Copyright 2020 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type BlueprintsSuite struct{}

var _ = Suite(&BlueprintsSuite{})

// Ensure that each ../blueprint/blueprints/*-blueprint.yaml exists in the
// wellknownBlueprintPaths map.
func (s *BlueprintsSuite) TestMissingWellknownBlueprints(c *C) {
	// List all ../blueprint/blueprints/*-blueprint.yaml files
	files, err := filepath.Glob(filepath.Join(blueprintsPath(), "*-blueprint.yaml"))
	c.Assert(err, IsNil)

	missingBlueprintPaths := make(map[string]string)
	for _, file := range files {
		blueprint := strings.TrimSuffix(filepath.Base(file), "-blueprint.yaml")

		// Ensure the well-known blueprint name is present
		if _, present := wellknownBlueprintPaths[blueprint]; !present {
			// Collect information on the missing blueprint
			lstat, err := os.Lstat(file)
			c.Assert(err, IsNil)
			if lstat.Mode()&os.ModeSymlink == os.ModeSymlink {
				link, err := os.Readlink(file)
				c.Assert(err, IsNil)
				missingBlueprintPaths[blueprint] = link
			} else {
				missingBlueprintPaths[blueprint] = filepath.Base(file)
			}
		}
	}

	// List missing blueprint paths
	if len(missingBlueprintPaths) > 0 {
		c.Log("The following symlinks are missing from wellknownBlueprintPaths:")
		for blueprint, path := range missingBlueprintPaths {
			c.Logf("\t%q: %q,", blueprint, path)
		}
		c.Fail()
	}
}

// Ensure that each blueprint path in the wellknownBlueprintPaths matches the
// file/symlink on the filesystem.
func (s *BlueprintsSuite) TestBlueprintPathsMatchSymlinks(c *C) {
	for blueprint, path := range wellknownBlueprintPaths {
		// Stat the corresponding file/symlink
		file := filepath.Join(blueprintsPath(), blueprint+"-blueprint.yaml")
		lstat, err := os.Lstat(file)
		c.Assert(err, IsNil)

		if lstat.Mode()&os.ModeSymlink == os.ModeSymlink {
			// Ensure the symlink matches the well-known path
			link, err := os.Readlink(file)
			c.Assert(err, IsNil)
			c.Assert(path, Equals, link)
		} else {
			// Ensure the filename matches the well-known path
			c.Assert(path, Equals, filepath.Base(file))
		}
	}
}

// Ensure WellknownApp returns the correct, resolved paths for each well-known
// blueprint path.
func (s *BlueprintsSuite) TestWellknownApp(c *C) {
	for blueprint, _ := range wellknownBlueprintPaths {
		appBlueprint, err := WellknownApp(blueprint)
		c.Assert(err, IsNil)

		// .App should match the blueprint
		c.Assert(appBlueprint.App, Equals, blueprint)

		// .Path should match the resolved path (resolved by the Stat call)
		directStat, err := os.Stat(filepath.Join(blueprintsPath(), blueprint+"-blueprint.yaml"))
		c.Assert(err, IsNil)
		wellknownStat, err := os.Stat(appBlueprint.Path)
		c.Assert(err, IsNil)
		if !os.SameFile(directStat, wellknownStat) {
			c.Errorf("WellknownApp returned %q, which doesn't match the symlink for %q", appBlueprint.Path, blueprint)
		}
	}
}
