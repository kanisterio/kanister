// Copyright 2026 The Kanister Authors.
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

package snapshot

import (
	"fmt"
	"testing"
)

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name      string
		errMsg    string
		transient bool
	}{
		{
			name:      "resource conflict error is transient",
			errMsg:    "the object has been modified; please apply your changes to the latest version and try again",
			transient: true,
		},
		{
			name:      "Portworx VolumeSnapshotContent is missing is transient",
			errMsg:    "VolumeSnapshotContent is missing",
			transient: true,
		},
		{
			name:      "Portworx SnapshotFinalizerError is transient",
			errMsg:    "SnapshotFinalizerError",
			transient: true,
		},
		{
			name:      "VolumeSnapshotContent is missing embedded in longer message",
			errMsg:    "failed to check and update snapshot content: VolumeSnapshotContent is missing",
			transient: true,
		},
		{
			name:      "SnapshotFinalizerError embedded in longer message",
			errMsg:    "error occurred: SnapshotFinalizerError: some detail",
			transient: true,
		},
		{
			name:      "unknown error is not transient",
			errMsg:    "some fatal error",
			transient: false,
		},
		{
			name:      "empty error is not transient",
			errMsg:    "",
			transient: false,
		},
		{
			name:      "snapshot creation failed is not transient",
			errMsg:    "snapshot creation failed permanently",
			transient: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := fmt.Errorf("%s", tc.errMsg)
			got := isTransientError(err)
			if got != tc.transient {
				t.Errorf("isTransientError(%q) = %v, want %v", tc.errMsg, got, tc.transient)
			}
		})
	}
}
