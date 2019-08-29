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

package output

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func splitLines(ctx context.Context, r io.ReadCloser, f func(context.Context, string) error) error {
	// Call r.Close() if the context is canceled or if s.Scan() == false.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		_ = r.Close()
	}()

	// Scan log lines when ready.
	s := bufio.NewScanner(r)
	for s.Scan() {
		l := s.Text()
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if err := f(ctx, l); err != nil {
			return err
		}
	}
	return errors.Wrap(s.Err(), "Split lines failed")
}

func LogAndParse(ctx context.Context, r io.ReadCloser) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	err := splitLines(ctx, r, func(ctx context.Context, l string) error {
		log.Info("Pod Out:", l)
		o, err := Parse(l)
		if err != nil {
			return err
		}
		if o != nil {
			out[o.Key] = o.Value
		}
		return nil
	})
	return out, err
}

func Log(ctx context.Context, r io.ReadCloser) error {
	err := splitLines(ctx, r, func(ctx context.Context, l string) error {
		log.Info("Pod Out:", l)
		return nil
	})
	return err
}
