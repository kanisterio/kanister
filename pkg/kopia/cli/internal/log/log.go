// Copyright 2024 The Kanister Authors.
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

package storage

import (
	"context"
	"io"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

// NopLogger is a logger that does nothing.
// TODO: Move to log package
type NopLogger struct{}

// Print does nothing.
func (NopLogger) Print(msg string, fields ...field.M) {
}

// PrintTo does nothing.
func (NopLogger) PrintTo(w io.Writer, msg string, fields ...field.M) {
}

// WithContext does nothing.
func (NopLogger) WithContext(ctx context.Context) log.Logger {
	return &NopLogger{}
}

// WithError does nothing.
func (NopLogger) WithError(err error) log.Logger {
	return &NopLogger{}
}
