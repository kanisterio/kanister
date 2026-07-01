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

package function

import "testing"

// The constructors exist so versioned overrides (e.g. a downstream
// v1.0.0-alpha registration) can embed the generic implementation and reuse its
// orchestration while overriding only Exec. Verify they return a usable
// function whose Name matches the generic registration.

func TestNewKubeTaskWithBackupPVCFunc(t *testing.T) {
	f := NewKubeTaskWithBackupPVCFunc()
	if f == nil {
		t.Fatal("NewKubeTaskWithBackupPVCFunc returned nil")
	}
	if got := f.Name(); got != KubeTaskWithBackupPVCFuncName {
		t.Fatalf("Name() = %q, want %q", got, KubeTaskWithBackupPVCFuncName)
	}
}

func TestNewKubeTaskWithRestorePVCFunc(t *testing.T) {
	f := NewKubeTaskWithRestorePVCFunc()
	if f == nil {
		t.Fatal("NewKubeTaskWithRestorePVCFunc returned nil")
	}
	if got := f.Name(); got != KubeTaskWithRestorePVCFuncName {
		t.Fatalf("Name() = %q, want %q", got, KubeTaskWithRestorePVCFuncName)
	}
}
