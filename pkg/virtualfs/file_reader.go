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
	"io"

	"github.com/kopia/kopia/fs"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

// ReaderSeekerCloser implements io.Reader, io.Seeker and io.Closer
type ReaderSeekerCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// readSeekerWrapper adds a no-op Close method to a ReadSeeker
type readSeekerWrapper struct {
	io.ReadSeeker
}

func (rs readSeekerWrapper) Close() error {
	return nil
}

// readCloserWrapper adds a no-op Seek method to a ReadCloser
type readCloserWrapper struct {
	io.ReadCloser
}

func (rc readCloserWrapper) Seek(start int64, offset int) (int64, error) {
	log.Debug().Print("Seek not supported", field.M{"start": start, "offset": offset})
	return 0, nil
}

// fileReader is an in-memory implementation of kopia's fs.Reader
type fileReader struct {
	ReaderSeekerCloser
	entry fs.Entry
}

func (fr *fileReader) Entry() (fs.Entry, error) {
	return fr.entry, nil
}
