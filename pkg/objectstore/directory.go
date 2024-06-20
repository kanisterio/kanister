// Copyright 2019 The Kanister Authors.
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

package objectstore

// Directories using Stow

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/graymeta/stow"
	"github.com/pkg/errors"
)

var _ Directory = (*directory)(nil)

type directory struct {
	// Every directory is part of a bucket.
	bucket *bucket
	path   string // Starts (and if needed, ends) with a '/'
}

// String creates a string representation that can used by OpenDirectory()
func (d *directory) String() string {
	return fmt.Sprintf("%s%s", d.bucket.hostEndPoint, d.path)
}

// CreateDirectory creates the d.path/dir/ object.
func (d *directory) CreateDirectory(ctx context.Context, dir string) (Directory, error) {
	dir = d.absDirName(dir)
	// Create directory marker
	if err := d.PutBytes(ctx, dir, nil, nil); err != nil {
		return nil, err
	}
	return &directory{
		bucket: d.bucket,
		path:   dir,
	}, nil
}

// GetDirectory gets the directory object
func (d *directory) GetDirectory(ctx context.Context, dir string) (Directory, error) {
	if dir == "" {
		return d, nil
	}
	dir = d.absDirName(dir)
	if _, err := d.bucket.container.Item(cloudName(dir)); err == nil {
		return &directory{
			bucket: d.bucket,
			path:   dir,
		}, nil
	}

	// Minio does not support `GET` on "directory" objects. To workaround,
	// we do a prefix search i.e. if we're trying to open directory `dir1/`,
	// we check if there is at least 1 object with the prefix `dir1/`.
	items, _, err := d.bucket.container.Items(cloudName(dir), stow.CursorStart, 1)
	switch {
	case err != nil:
		return nil, errors.Wrapf(err, "could not get directory marker %s", dir)
	case len(items) == 0:
		return nil, errors.Errorf("no items found. could not get directory marker %s", dir)
	}
	return &directory{
		bucket: d.bucket,
		path:   dir,
	}, nil
}

// ListDirectories lists all the directories that have d.path as the prefix.
// the returned map is indexed by the relative directory name (without trailing '/')
func (d *directory) ListDirectories(ctx context.Context) (map[string]Directory, error) {
	if d.path == "" {
		return nil, errors.New("invalid entry")
	}

	directories := make(map[string]Directory)

	err := stow.Walk(d.bucket.container, cloudName(d.path), 10000,
		func(item stow.Item, err error) error {
			if err != nil {
				return err
			}

			// Check if the object is nested in a directory hierarchy
			// and if so - return the first directory entry.
			// e.g.
			// If we're doing a prefix search for all objects under
			// `parent/` and we find `parent/dir1/` - then we want
			// to track `dir1/`
			dir := strings.TrimPrefix(item.Name(), cloudName(d.path))
			if dir == "" {
				// e.g., /<d.path>/
				return nil
			}

			if dirEnt, ok := getFirstDirectoryMarker(dir); ok {
				// Use maps to uniqify
				// e.g., /dir1/, /dir1/file1, /dir1/dir2/, /dir1/dir2/file2 will leave /dir
				directories[dirEnt] = &directory{
					bucket: d.bucket,
					path:   d.absDirName(dirEnt),
				}
			}

			return nil
		})
	if err != nil {
		return nil, err
	}
	return directories, nil
}

// ListObjects lists all the files that have d.dirname as the prefix.
func (d *directory) ListObjects(ctx context.Context) ([]string, error) {
	if d.path == "" {
		return nil, errors.New("invalid entry")
	}

	objects := make([]string, 0, 1)
	err := stow.Walk(d.bucket.container, cloudName(d.path), 10000,
		func(item stow.Item, err error) error {
			if err != nil {
				return err
			}
			objName := strings.TrimPrefix(item.Name(), cloudName(d.path))
			if objName != "" && !strings.Contains(objName, "/") {
				objects = append(objects, objName)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return objects, nil
}

// DeleteDirectory deletes all objects that have d.path as the prefix
// <bucket>/<d.path>/<everything> including <bucket>/<d.path>/<some dir>/<objects>
func (d *directory) DeleteDirectory(ctx context.Context) error {
	if d.path == "" {
		return errors.New("invalid entry")
	}
	return deleteWithPrefix(ctx, d.bucket.container, cloudName(d.path))
}

// DeleteDirectory deletes all objects that have d.path/dir as the prefix
// <bucket>/<d.path>/dir/<everything> including <bucket>/<d.path>/dir/<some dir>/<objects>
func (d *directory) DeleteAllWithPrefix(ctx context.Context, prefix string) error {
	p := cloudName(filepath.Join(d.path, prefix))
	return deleteWithPrefix(ctx, d.bucket.container, p)
}

func deleteWithPrefix(ctx context.Context, c stow.Container, prefix string) error {
	err := stow.Walk(c, prefix, 10000,
		func(item stow.Item, err error) error {
			if err != nil {
				return err
			}
			return c.RemoveItem(item.Name())
		})
	if err != nil {
		return errors.Wrapf(err, "Failed to delete item %s", prefix)
	}
	return nil
}

func (d *directory) Get(ctx context.Context, name string) (io.ReadCloser, map[string]string, error) {
	if d.path == "" {
		return nil, nil, errors.New("invalid entry")
	}

	objName := d.absPathName(name)

	item, err := d.bucket.container.Item(cloudName(objName))
	if err != nil {
		return nil, nil, err
	}

	// Open the object and read all data
	r, err := item.Open()
	if err != nil {
		return nil, nil, err
	}
	rTags, err := item.Metadata()
	if err != nil {
		return nil, nil, err
	}

	// Convert tags:map[string]interface{} into map[string]string
	tags := make(map[string]string)
	for key, val := range rTags {
		if sVal, ok := val.(string); ok {
			tags[key] = sVal
		}
	}

	return r, tags, nil
}

// Get data and tags associated with an object <bucket>/<d.path>/name.
func (d *directory) GetBytes(ctx context.Context, name string) ([]byte, map[string]string, error) {
	r, tags, err := d.Get(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close() //nolint:errcheck

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	return data, tags, nil
}

func (d *directory) Put(ctx context.Context, name string, r io.Reader, size int64, tags map[string]string) error {
	if d.path == "" {
		return errors.New("invalid entry")
	}
	// K10 tags include '/'. Remove them, at least for S3
	sTags := sanitizeTags(tags)

	objName := d.absPathName(name)

	// For versioned buckets, Put can return the new version name
	// TODO: Support versioned buckets
	_, err := d.bucket.container.Put(cloudName(objName), r, size, sTags)
	return err
}

// Put stores a blob in d.path/<name>
func (d *directory) PutBytes(ctx context.Context, name string, data []byte, tags map[string]string) error {
	return d.Put(ctx, name, bytes.NewReader(data), int64(len(data)), tags)
}

// Delete removes an object
func (d *directory) Delete(ctx context.Context, name string) error {
	if d.path == "" {
		return errors.New("invalid entry")
	}

	objName := d.absPathName(name)

	return d.bucket.container.RemoveItem(cloudName(objName))
}

// If name does not start with '/', prefix with d.path. Add '/' as suffix
func (d *directory) absDirName(dir string) string {
	dir = d.absPathName(dir)

	// End with a '/'
	if !strings.HasSuffix(dir, "/") {
		dir = filepath.Clean(dir) + "/"
	}

	return dir
}

// S3 ignores the root '/' while creating objects. During
// filtering operations however, the '/' is not ignored.
// GCS creates an explicit '/' in the bucket. cloudName
// strips the initial '/' for stow operations. '/' still
// implies root for objectstore.
func cloudName(dir string) string {
	return strings.TrimPrefix(dir, "/")
}

// If name does not start with '/', prefix with d.path.
func (d *directory) absPathName(name string) string {
	if name == "" {
		return ""
	}
	if !filepath.IsAbs(name) {
		name = d.path + name
	}

	return name
}

// sanitizeTags replaces '/' with "-" in tag keys
func sanitizeTags(tags map[string]string) map[string]interface{} {
	cTags := make(map[string]interface{})
	for key, val := range tags {
		cKey := strings.ReplaceAll(key, "/", "-")
		cTags[cKey] = val
	}
	return cTags
}

// getFirstDirectoryMarker checks if path includes one '/'.
// If so, returns value until first '/' which is the first
// directory marker
// path is of the form elem1/elem2/, returns elem1
func getFirstDirectoryMarker(path string) (string, bool) {
	// TODO: Change this to strings.SplitN(path, "/", 2)
	s := strings.SplitN(path, "/", 3)
	switch len(s) {
	case 1:
		// No '/' e.g. "elem"
		return "", false
	case 2:
		// e.g. dir1/dir2 -> return dir1
		return s[0], true
	case 3:
		// e.g. dir1/dir2/elem -> return dir1
		return s[0], true
	}

	// Not reached
	return "", false
}
