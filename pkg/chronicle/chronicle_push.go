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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/envdir"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
)

type PushParams struct {
	ProfilePath  string
	ArtifactFile string
	Frequency    time.Duration
	EnvDir       string
	Command      []string
}

func (p PushParams) Validate() error {
	return nil
}

func Push(p PushParams) error {
	log.Debug().Print("", field.M{"PushParams": p})
	ctx := setupSignalHandler(context.Background())
	var i int
	for {
		start := time.Now().UTC()

		if err := push(ctx, p, i); err != nil {
			return err
		}

		end := time.Now().UTC()
		sleep := p.Frequency - end.Sub(start)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(sleep):
		}
		i++
	}
}

func setupSignalHandler(ctx context.Context) context.Context {
	var can context.CancelFunc
	ctx, can = context.WithCancel(ctx)
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Print("Shutting down process")
		can()
		<-c
		log.Print("Killing process")
		os.Exit(1)
	}()
	return ctx
}

func push(ctx context.Context, p PushParams, ord int) error {
	// Read profile.
	prof, ok, err := readProfile(p.ProfilePath)
	if !ok || err != nil {
		return err
	}
	// Get envdir values if set.
	var env []string
	if p.EnvDir != "" {
		var err error
		env, err = envdir.EnvDir(p.EnvDir)
		if err != nil {
			return err
		}
	}
	ap, _ := readArtifactPathFile(p.ArtifactFile)
	log.Debug().Print("Pushing output from Command ", field.M{"order": ord, "command": p.Command, "Environment": env})
	return pushWithEnv(ctx, p.Command, ap, ord, prof, env)
}

func pushWithEnv(ctx context.Context, c []string, suffix string, ord int, prof param.Profile, env []string) error {
	// Chronicle command w/ piped output.
	cmd := exec.CommandContext(ctx, "sh", "-c", strings.Join(c, " "))
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "Failed to open command pipe")
	}
	cmd.Stderr = os.Stderr
	cur := fmt.Sprintf("%s-%d", suffix, ord)
	// Write data to object store
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "Failed to start chronicle pipe command")
	}
	if err := location.Write(ctx, out, prof, cur); err != nil {
		return errors.Wrap(err, "Failed to write command output to object storage")
	}
	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "Chronicle pipe command failed")
	}

	// Write manifest pointing to new data
	man := strings.NewReader(cur)
	if err := location.Write(ctx, man, prof, suffix); err != nil {
		return errors.Wrap(err, "Failed to write command output to object storage")
	}
	// Delete old data
	prev := fmt.Sprintf("%s-%d", suffix, ord-1)
	_ = location.Delete(ctx, prof, prev)
	return nil
}

func readArtifactPathFile(path string) (string, error) {
	buf, err := os.ReadFile(path)
	t := strings.TrimSuffix(string(buf), "\n")
	return t, errors.Wrap(err, "Could not read artifact path file")
}

func readProfile(path string) (param.Profile, bool, error) {
	var buf []byte
	buf, err := os.ReadFile(path)
	var p param.Profile
	switch {
	case os.IsNotExist(err):
		err = nil
		return p, false, err
	case err != nil:
		return p, false, errors.Wrap(err, "Failed to read profile")
	}
	if err = json.Unmarshal(buf, &p); err != nil {
		return p, false, errors.Wrap(err, "Failed to unmarshal profile")
	}
	return p, true, nil
}

func writeProfile(path string, p param.Profile) error {
	buf, err := json.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "Failed to write profile")
	}
	return os.WriteFile(path, buf, os.ModePerm)
}
