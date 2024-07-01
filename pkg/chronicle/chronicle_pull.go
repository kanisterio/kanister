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

package chronicle

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func Pull(ctx context.Context, target io.Writer, p param.Profile, manifest string) error {
	// Read manifest
	buf := bytes.NewBuffer(nil)
	_ = location.Read(ctx, buf, p, manifest)
	// Read Data
	data, err := io.ReadAll(buf)
	if err != nil {
		return errors.Wrap(err, "Could not read chronicle manifest")
	}
	return location.Read(ctx, target, p, string(data))
}
