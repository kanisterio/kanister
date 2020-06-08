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
	"strings"

	"github.com/kopia/kopia/fs"
	"github.com/pkg/errors"
)

// Directory is a mock in-memory implementation of kopia's fs.Directory
type Directory struct {
	dirEntry

	children     fs.Entries
	readdirError error
	onReaddir    func()
}

// Summary returns summary of a directory.
func (d *Directory) Summary() *fs.DirectorySummary {
	return nil
}

// AddFileWithStreamSource adds a mock file with the specified name, permissions and source.
func (d *Directory) AddFileWithStreamSource(name, sourceEndpoint string, permissions os.FileMode) (file *File, err error) {
	d, name, err = d.resolveAndAddSubdir(name)
	if err != nil {
		return nil, err
	}

	file = &File{
		dirEntry: dirEntry{
			name: name,
			mode: permissions,
			// TODO: Add owner information
		},
		source: func() (ReaderSeekerCloser, error) {
			return streamReader(sourceEndpoint)
		},
	}

	err = d.addChild(file)

	return file, errors.Wrap(err, "Failed to add file")
}

// AddDir adds a fake directory with a given name and permissions.
func (d *Directory) AddDir(name string, permissions os.FileMode) (subdir *Directory, err error) {
	d, name, err = d.resolveSubdir(name)
	if err != nil {
		return nil, err
	}

	subdir = &Directory{
		dirEntry: dirEntry{
			name: name,
			mode: permissions | os.ModeDir,
		},
	}

	err = d.addChild(subdir)

	return subdir, err
}

// Subdir finds a subdirectory with the given name.
func (d *Directory) Subdir(name ...string) (*Directory, error) {
	i := d

	for _, n := range name {
		i2 := i.children.FindByName(n)
		if i2 == nil {
			return nil, errors.New(fmt.Sprintf("'%s' not found in '%s'", n, i.Name()))
		}

		if !i2.IsDir() {
			return nil, errors.New(fmt.Sprintf("'%s' is not a directory in '%s'", n, i.Name()))
		}

		i = i2.(*Directory)
	}

	return i, nil
}

// Remove removes directory dirEntry with the given name.
func (d *Directory) Remove(name string) {
	newChildren := d.children[:0]

	for _, e := range d.children {
		if e.Name() != name {
			newChildren = append(newChildren, e)
		}
	}

	d.children = newChildren
}

// OnReaddir invokes the provided function on read.
func (d *Directory) OnReaddir(cb func()) {
	d.onReaddir = cb
}

// Child gets the named child of a directory.
func (d *Directory) Child(ctx context.Context, name string) (fs.Entry, error) {
	return fs.ReadDirAndFindChild(ctx, d, name)
}

// Readdir gets the contents of a directory.
func (d *Directory) Readdir(ctx context.Context) (fs.Entries, error) {
	if d.readdirError != nil {
		return nil, d.readdirError
	}

	if d.onReaddir != nil {
		d.onReaddir()
	}

	return append(fs.Entries(nil), d.children...), nil
}

func (d *Directory) addChild(e fs.Entry) error {
	if strings.Contains(e.Name(), "/") {
		return errors.New("Failed to add child entry: name cannot contain '/'")
	}

	d.children = append(d.children, e)
	d.children.Sort()
	return nil
}

func (d *Directory) resolveSubdir(name string) (parent *Directory, leaf string, err error) {
	parts := strings.Split(name, "/")
	for _, n := range parts[0 : len(parts)-1] {
		if d, err = d.Subdir(n); err != nil {
			return nil, "", errors.Wrap(err, fmt.Sprintf("Failed to resolve sub directory '%s'", n))
		}
	}

	return d, parts[len(parts)-1], nil
}

func (d *Directory) resolveAndAddSubdir(name string) (parent *Directory, leaf string, err error) {
	parts := strings.Split(name, "/")
	for _, n := range parts[0 : len(parts)-1] {
		if d, err = d.AddDir(n, 0777); err != nil {
			return nil, "", errors.Wrap(err, fmt.Sprintf("Failed to add sub directory '%s'", n))
		}
	}

	return d, parts[len(parts)-1], nil
}
