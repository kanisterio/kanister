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
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/kopia/kopia/fs"
	"github.com/pkg/errors"
)

// Directory is a mock in-memory implementation of kopia's fs.Directory
type Directory struct {
	dirEntry

	children []fs.Entry
}

var _ (fs.Directory) = (*Directory)(nil)

// AddDir adds a directory with a given name and permissions
func (d *Directory) AddDir(name string, permissions os.FileMode) (*Directory, error) {
	subdir := &Directory{
		dirEntry: dirEntry{
			name: name,
			mode: permissions | os.ModeDir,
		},
	}

	if err := d.addChild(subdir); err != nil {
		return nil, err
	}
	return subdir, nil
}

// AddAllDirs creates under d, all the necessary directories in pathname, similar to os.MkdirAll
func (d *Directory) AddAllDirs(pathname string, permissions os.FileMode) (subdir *Directory, err error) {
	p, missing, err := d.resolveDirs(pathname)
	if err != nil {
		return nil, err
	}

	for _, n := range missing {
		if p, err = p.AddDir(n, permissions); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to add sub directory '%s'", n))
		}
	}

	return p, nil
}

// Child gets the named child of a directory
func (d *Directory) Child(ctx context.Context, name string) (fs.Entry, error) {
	return fs.IterateEntriesAndFindChild(ctx, d, name)
}

// Iterate returns directory iterator.
func (d *Directory) Iterate(ctx context.Context) (fs.DirectoryIterator, error) {
	return fs.StaticIterator(append([]fs.Entry{}, d.children...), nil), nil
}

// Remove removes directory dirEntry with the given name
func (d *Directory) Remove(name string) {
	newChildren := d.children[:0]

	for _, e := range d.children {
		if e.Name() != name {
			newChildren = append(newChildren, e)
		}
	}

	d.children = newChildren
}

// Subdir finds a subdirectory with the given name
func (d *Directory) Subdir(name string) (*Directory, error) {
	curr := d

	subdir := fs.FindByName(curr.children, name)
	if subdir == nil {
		return nil, errors.New(fmt.Sprintf("'%s' not found in '%s'", name, curr.Name()))
	}
	if !subdir.IsDir() {
		return nil, errors.New(fmt.Sprintf("'%s' is not a directory in '%s'", name, curr.Name()))
	}

	return subdir.(*Directory), nil
}

// Summary returns summary of a directory
func (d *Directory) Summary() *fs.DirectorySummary {
	return nil
}

// addChild adds the given entry under d, errors out if the entry is already present
func (d *Directory) addChild(e fs.Entry) error {
	if strings.Contains(e.Name(), "/") {
		return errors.New("Failed to add child entry: name cannot contain '/'")
	}

	child := fs.FindByName(d.children, e.Name())
	if child != nil {
		return errors.New("Failed to add child entry: already exists")
	}

	d.children = append(d.children, e)
	fs.Sort(d.children)
	return nil
}

// resolveDirs finds the directories in the pathname under d and returns a list of missing sub directories
func (d *Directory) resolveDirs(pathname string) (parent *Directory, missing []string, err error) {
	if pathname == "" {
		return d, nil, nil
	}

	p := d
	parts := strings.Split(path.Clean(pathname), "/")
	for i, n := range parts {
		i2 := fs.FindByName(p.children, n)
		if i2 == nil {
			return p, parts[i:], nil
		}
		if !i2.IsDir() {
			return nil, nil, errors.New(fmt.Sprintf("'%s' is not a directory in '%s'", n, p.Name()))
		}
		p = i2.(*Directory)
	}

	return p, nil, nil
}

func (d *Directory) SupportsMultipleIterations() bool {
	return true
}

func (d *Directory) Close() {}

// AddFileWithStreamSource adds a virtual file with the specified name, permissions and source
func AddFileWithStreamSource(d *Directory, filePath, sourceEndpoint string, dirPermissions, filePermissions os.FileMode) (*file, error) {
	dir, name := path.Split(filePath)
	p, err := d.AddAllDirs(dir, dirPermissions)
	if err != nil {
		return nil, err
	}

	f := FileFromEndpoint(name, sourceEndpoint, filePermissions)
	if err := p.addChild(f); err != nil {
		return nil, errors.Wrap(err, "Failed to add file")
	}
	return f, nil
}

// AddFileWithContent adds a virtual file with specified name, permissions and content
func AddFileWithContent(d *Directory, filePath string, content []byte, dirPermissions, filePermissions os.FileMode) (*file, error) {
	dir, name := path.Split(filePath)
	p, err := d.AddAllDirs(dir, dirPermissions)
	if err != nil {
		return nil, err
	}

	f := FileWithContent(name, filePermissions, content)
	if err := p.addChild(f); err != nil {
		return nil, errors.Wrap(err, "Failed to add file")
	}
	return f, nil
}
