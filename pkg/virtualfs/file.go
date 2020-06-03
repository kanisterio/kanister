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

	"github.com/kopia/kopia/fs"
	"github.com/pkg/errors"
)

// File is an in-memory implementation of kopia's fs.File
type File struct {
	dirEntry

	source func() (ReaderSeekerCloser, error)
}

// SetContents changes the contents of a given file.
func (f *File) SetContents(b []byte) {
	f.source = func() (ReaderSeekerCloser, error) {
		return readerSeekerCloser{bytes.NewReader(b)}, nil
	}
}

// Open opens the file for reading.
func (f *File) Open(ctx context.Context) (fs.Reader, error) {
	r, err := f.source()
	if err != nil {
		return nil, err
	}

	return &fileReader{
		ReaderSeekerCloser: r,
		entry:              f,
	}, nil
}

func streamReader(sourceEndpoint string) (ReaderSeekerCloser, error) {
	req, err := http.NewRequest("GET", sourceEndpoint, nil)
	if err != nil {
		return readerCloser{nil}, errors.Wrap(err, "Failed to generate HTTP request")
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return readerCloser{nil}, errors.Wrap(err, "Failed to make HTTP request")
	}

	return readerCloser{resp.Body}, nil
}
