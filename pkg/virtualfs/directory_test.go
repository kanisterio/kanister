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

package virtualfs

import (
	"context"
	"os"

	"gopkg.in/check.v1"
)

const (
	defaultPermissions os.FileMode = 0777
	dirPermissions     os.FileMode = defaultPermissions | os.ModeDir
)

type DirectorySuite struct{}

var _ = check.Suite(&DirectorySuite{})

func (s *DirectorySuite) TestAddDir(c *check.C) {
	sourceDir, err := NewDirectory("root")
	c.Assert(err, check.IsNil)

	// New directory
	dir, err := sourceDir.AddDir("d1", defaultPermissions)
	c.Assert(err, check.IsNil)
	c.Assert(dir.Name(), check.Equals, "d1")

	// Duplicate directory
	_, err = sourceDir.AddDir("d1", defaultPermissions)
	c.Assert(err, check.NotNil)

	// Invalid name
	_, err = sourceDir.AddDir("/d2", defaultPermissions)
	c.Assert(err, check.NotNil)
}

func (s *DirectorySuite) TestAddAllDirs(c *check.C) {
	sourceDir, err := NewDirectory("root")
	c.Assert(err, check.IsNil)

	c.Log("Add a directory - root/d1")
	subdir, err := sourceDir.AddAllDirs("d1", defaultPermissions)
	c.Assert(err, check.IsNil)
	c.Assert(subdir.Name(), check.Equals, "d1")
	d1, err := sourceDir.Subdir("d1")
	c.Assert(err, check.IsNil)
	c.Assert(d1, check.NotNil)
	c.Assert(d1.Name(), check.Equals, "d1")
	c.Assert(d1.Mode(), check.Equals, dirPermissions)

	c.Log("Add a sub-dir under an existing directory - root/d1/d2")
	subdir, err = sourceDir.AddAllDirs("d1/d2", defaultPermissions)
	c.Assert(err, check.IsNil)
	c.Assert(subdir.Name(), check.Equals, "d2")
	d2, err := d1.Subdir("d2")
	c.Assert(err, check.IsNil)
	c.Assert(d2, check.NotNil)
	c.Assert(d2.Name(), check.Equals, "d2")
	c.Assert(d2.Mode(), check.Equals, dirPermissions)

	c.Log("Add third/fourth level dirs - root/d1/d3/d4")
	subdir, err = sourceDir.AddAllDirs("d1/d3/d4", defaultPermissions)
	c.Assert(err, check.IsNil)
	c.Assert(subdir.Name(), check.Equals, "d4")
	d3, err := d1.Subdir("d3")
	c.Assert(err, check.IsNil)
	c.Assert(d3, check.NotNil)
	c.Assert(d3.Name(), check.Equals, "d3")
	c.Assert(d3.Mode(), check.Equals, dirPermissions)
	d4, err := d3.Subdir("d4")
	c.Assert(err, check.IsNil)
	c.Assert(d4, check.NotNil)
	c.Assert(d4.Name(), check.Equals, "d4")
	c.Assert(d4.Mode(), check.Equals, dirPermissions)

	c.Log("Fail adding directory under a file - root/f1/d6")
	f, err := AddFileWithContent(sourceDir, "f1", []byte("test"), defaultPermissions, defaultPermissions)
	c.Assert(err, check.IsNil)
	c.Assert(f.Name(), check.Equals, "f1")
	_, err = sourceDir.AddAllDirs("f1/d6", defaultPermissions)
	c.Assert(err, check.NotNil)
}

func (s *DirectorySuite) TestAddFile(c *check.C) {
	sourceDir, err := NewDirectory("root")
	c.Assert(err, check.IsNil)

	c.Log("Add file with stream source - root/f1")
	f, err := AddFileWithStreamSource(sourceDir, "f1", "http://test-endpoint", defaultPermissions, defaultPermissions)
	c.Assert(err, check.IsNil)
	c.Assert(f, check.NotNil)
	c.Assert(f.Name(), check.Equals, "f1")

	c.Log("Add file with stream source at third level - root/d1/f2")
	f, err = AddFileWithStreamSource(sourceDir, "d1/f2", "http://test-endpoint", defaultPermissions, defaultPermissions)
	c.Assert(err, check.IsNil)
	c.Assert(f, check.NotNil)
	c.Assert(f.Name(), check.Equals, "f2")
	d1, err := sourceDir.Subdir("d1")
	c.Assert(err, check.IsNil)
	c.Assert(d1, check.NotNil)
	c.Assert(d1.Name(), check.Equals, "d1")
	c.Assert(d1.Mode(), check.Equals, dirPermissions)
	e, err := d1.Child(context.Background(), "f2")
	c.Assert(err, check.IsNil)
	c.Assert(e, check.NotNil)

	c.Log("Add file with content at third level - root/d2/f3")
	f, err = AddFileWithContent(sourceDir, "d2/f3", []byte("test"), defaultPermissions, defaultPermissions)
	c.Assert(err, check.IsNil)
	c.Assert(f, check.NotNil)
	c.Assert(f.Name(), check.Equals, "f3")
	d2, err := sourceDir.Subdir("d2")
	c.Assert(err, check.IsNil)
	c.Assert(d2, check.NotNil)
	c.Assert(d2.Name(), check.Equals, "d2")
	c.Assert(d2.Mode(), check.Equals, dirPermissions)
	e, err = d2.Child(context.Background(), "f3")
	c.Assert(err, check.IsNil)
	c.Assert(e, check.NotNil)
}
