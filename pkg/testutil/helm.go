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

package testutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const DefaultCommandTimeout = 10 * time.Minute

type HelmClient struct{}

func NewHelmClient() HelmClient {
	return HelmClient{}
}

// Add adds new helm repo and fetches latest charts
func (h *HelmClient) AddRepo(ctx context.Context, name, url string) error {
	log.Debugf("Adding helm repo %s\n", name)
	out, err := RunCmdWithTimeout(ctx, "helm", []string{"repo", "add", name, url})
	if err != nil {
		return err
	}
	log.Debug("Result: ", out)

	// Update all repos to fetch the latest charts
	h.RepoUpdate(ctx)
	return nil
}

// Update fetches latest helm charts from the repo
func (h HelmClient) RepoUpdate(ctx context.Context) error {
	log.Debugf("Fetching latest helm charts from the helm repos\n")
	out, err := RunCmdWithTimeout(ctx, "helm", []string{"repo", "update"})
	if err != nil {
		return err
	}
	log.Debug("Result: ", out)
	return nil
}

// Install installs helm chart with given release name
func (h HelmClient) Install(ctx context.Context, chart, release, namespace string, values map[string]string) error {
	log.Debugf("Installing helm chart %s\n", chart)
	var setVals string
	for k, v := range values {
		setVals += fmt.Sprintf("%s=%s,", k, v)
	}
	out, err := RunCmdWithTimeout(ctx, "helm", []string{"install", release, "--namespace", namespace, chart, "--set", setVals})
	if err != nil {
		return err
	}
	log.Debug("Result: ", out)
	return nil
}

// RunCmdWithTimeout executes command on host with DefaultCommandTimeout timeout
func RunCmdWithTimeout(ctx context.Context, command string, args []string) (string, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	log.Debug("Executing command ", command, args)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		log.Error("Command Start error: %s", err.Error())
		return out.String(), err
	}

	// Stop command exec if exceeds timeout
	timer := time.AfterFunc(DefaultCommandTimeout, func() {
		cmd.Process.Kill()
	})
	err := cmd.Wait()
	timer.Stop()

	return strings.TrimSpace(out.String()), err
}
