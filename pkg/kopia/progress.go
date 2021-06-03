// Copyright 2021 The Kanister Authors.
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

package kopia

import (
	"sync"
	"sync/atomic"

	"github.com/kopia/kopia/snapshot/snapshotfs"
)

// kandoUploadProgress implements the snapshotfs.UploadProgress
// The progress counters are updated by kopia upload process
type kandoUploadProgress struct {
	snapshotfs.NullUploadProgress

	// all int64 must precede all int32 due to alignment requirements on ARM
	uploadedBytes int64
	cachedBytes   int64
	hashedBytes   int64

	getMutex sync.Mutex
}

// UploadStarted implements snapshotfs.UploadProgress
func (p *kandoUploadProgress) UploadStarted() {}

// EstimatedDataSize implements snapshotfs.UploadProgress
func (p *kandoUploadProgress) EstimatedDataSize(fileCount int, totalBytes int64) {}

// UploadFinished implements snapshotfs.UploadProgress
func (p *kandoUploadProgress) UploadFinished() {}

// HashedBytes implements snapshotfs.UploadProgress
func (p *kandoUploadProgress) HashedBytes(numBytes int64) {
	atomic.AddInt64(&p.hashedBytes, numBytes)
}

// ExcludedFile implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) ExcludedFile(fname string, numBytes int64) {}

// ExcludedDir implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) ExcludedDir(dirname string) {}

// CachedFile implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) CachedFile(fname string, numBytes int64) {
	atomic.AddInt64(&p.cachedBytes, numBytes)
}

// UploadedBytes implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) UploadedBytes(numBytes int64) {
	atomic.AddInt64(&p.uploadedBytes, numBytes)
}

// HashingFile implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) HashingFile(fname string) {}

// FinishedHashingFile implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) FinishedHashingFile(fname string, numBytes int64) {}

// StartedDirectory implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) StartedDirectory(dirname string) {}

// FinishedDirectory implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) FinishedDirectory(dirname string) {}

// Error implements Kopia snapshotfs.UploadProgress
func (p *kandoUploadProgress) Error(path string, err error, isIgnored bool) {}

var _ snapshotfs.UploadProgress = (*kandoUploadProgress)(nil)

// GetStats returns the stats collected by the kandoUploadProgress
func (p *kandoUploadProgress) GetStats() (hashed, cached, uploaded int64) {
	p.getMutex.Lock()
	defer p.getMutex.Unlock()

	hashed = p.hashedBytes
	cached = p.cachedBytes
	uploaded = p.uploadedBytes
	return
}
