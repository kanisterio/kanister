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

package restic

import (
	"crypto/sha256"
	"fmt"
)

const (
	password = "testpassword"
)

// GeneratePassword generates a password
func GeneratePassword() string {
	h := sha256.New()
	_, _ = h.Write([]byte(password))
	return fmt.Sprintf("%x", h.Sum(nil))
}
