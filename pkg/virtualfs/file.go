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
	"bytes"
	"context"
	"net/http"
	"os"

	"github.com/kopia/kopia/fs"
	"github.com/pkg/errors"
)

// file is an in-memory implementation of kopia's fs.File
type file struct {
	dirEntry

	source func() (ReaderSeekerCloser, error)
}

var _ fs.File = (*file)(nil)

// Open opens the file for reading
func (f *file) Open(ctx context.Context) (fs.Reader, error) {
	r, err := f.source()
	if err != nil {
		return nil, err
	}

	return &fileReader{
		ReaderSeekerCloser: r,
		entry:              f,
	}, nil
}

// FileWithSource returns a file with given name, permissions and source
func FileWithSource(name string, permissions os.FileMode, source func() (ReaderSeekerCloser, error)) *file {
	return &file{
		dirEntry: dirEntry{
			name: name,
			mode: permissions,
			// TODO: add owner and other information
		},
		source: source,
	}
}

// FileWithContent returns a file with given content
func FileWithContent(name string, permissions os.FileMode, content []byte) *file {
	s := func() (ReaderSeekerCloser, error) {
		return readSeekerWrapper{bytes.NewReader(content)}, nil
	}

	return FileWithSource(name, permissions, s)
}

// FileFromEndpoint returns a file with contents from given source endpoint
func FileFromEndpoint(name, sourceEndpoint string, permissions os.FileMode) *file {
	s := func() (ReaderSeekerCloser, error) {
		return httpStreamReader(sourceEndpoint)
	}

	return FileWithSource(name, permissions, s)
}

// httpStreamReader reads the data stream from the given source endpoint
func httpStreamReader(sourceEndpoint string) (ReaderSeekerCloser, error) {
	req, err := http.NewRequest("GET", sourceEndpoint, nil)
	if err != nil {
		return readCloserWrapper{nil}, errors.Wrap(err, "Failed to generate HTTP request")
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return readCloserWrapper{nil}, errors.Wrap(err, "Failed to make HTTP request")
	}

	return readCloserWrapper{resp.Body}, nil
}
