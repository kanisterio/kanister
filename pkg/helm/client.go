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

package helm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type HelmVersion string

const (
	DefaultCommandTimeout = 15 * time.Minute

	// Add Bitnami charts url
	BitnamiRepoName = "bitnami"
	BitnamiRepoURL  = "https://charts.bitnami.com/bitnami"

	// Add elastic charts url
	ElasticRepoName = "elastic"
	ElasticRepoURL  = "https://helm.elastic.co"

	// Add couchbase chart
	CouchbaseRepoName = "couchbase"
	CouchbaseRepoURL  = "https://couchbase-partners.github.io/helm-charts"

	// HelmVersion to differentiate between helm2 and helm3 commands
	V2 HelmVersion = "helmv2"
	V3 HelmVersion = "helmv3"

	// HelmBinNameEnvVar env var to pass a helm bin name into kanister test apps
	HelmBinNameEnvVar = "KANISTER_HELM_BIN"
)

type CliClient struct {
	version HelmVersion
	helmBin string
}

// FindVersion returns HelmVersion based on helm binary present in the path
func FindVersion() (HelmVersion, error) {
	out, err := RunCmdWithTimeout(context.TODO(), GetHelmBinName(), []string{"version", "--client", "--short"})
	if err != nil {
		return "", err
	}

	// Trim prefix if output from helm v2
	out = strings.TrimPrefix(out, "Client: ")

	if strings.HasPrefix(out, "v2") {
		return V2, nil
	}
	if strings.HasPrefix(out, "v3") {
		return V3, nil
	}
	return "", fmt.Errorf("Unsupported helm version %s", out)
}

// GetHelmBinName returns a helm bin name from env var
func GetHelmBinName() string {
	if helmBin, ok := os.LookupEnv(HelmBinNameEnvVar); ok {
		return helmBin
	}
	return "helm"
}

func NewCliClient() (Client, error) {
	version, err := FindVersion()
	if err != nil {
		return nil, err
	}

	return &CliClient{
		version: version,
		helmBin: GetHelmBinName(),
	}, nil
}

// AddRepo adds new helm repo and fetches latest charts
func (h CliClient) AddRepo(ctx context.Context, name, url string) error {
	log.Info().Print("Adding helm repo", field.M{"name": name, "url": url})
	out, err := RunCmdWithTimeout(ctx, h.helmBin, []string{"repo", "add", name, url})
	if err != nil {
		return err
	}
	log.Info().Print("Result", field.M{"output": out})

	// Update all repos to fetch the latest charts
	err = h.UpdateRepo(ctx)
	return err
}

// UpdateRepo fetches latest helm charts from the repo
func (h CliClient) UpdateRepo(ctx context.Context) error {
	log.Info().Print("Fetching latest helm charts from the helm repos")
	out, err := RunCmdWithTimeout(ctx, h.helmBin, []string{"repo", "update"})
	if err != nil {
		return err
	}
	log.Info().Print("Result", field.M{"output": out})
	return nil
}

// Install installs helm chart with given release name
func (h CliClient) Install(ctx context.Context, chart, version, release, namespace string, values map[string]string) error {
	log.Info().Print("Installing helm chart", field.M{"chart": chart, "version": version, "release": release, "namespace": namespace})
	var setVals string
	for k, v := range values {
		setVals += fmt.Sprintf("%s=%s,", k, v)
	}

	var cmd []string
	if h.version == V3 {
		cmd = []string{"install", release, "--version", version, "--timeout", "15m", "--namespace", namespace, chart, "--set", setVals, "--wait"}
	} else {
		cmd = []string{"install", "--name", release, "--version", version, "--namespace", namespace, chart, "--set", setVals, "--wait"}
	}
	log.Info().Print("Command formed ", field.M{"Command": cmd})

	out, err := RunCmdWithTimeout(ctx, h.helmBin, cmd)
	log.Info().Print("Command output ", field.M{"chart": chart, "output": out})
	if err != nil {
		return err
	}
	log.Info().Print("Result", field.M{"output": out})
	return nil
}

// Uninstall deletes helm release
func (h CliClient) Uninstall(ctx context.Context, release, namespace string) error {
	log.Debug().Print("Uninstalling helm chart", field.M{"release": release, "namespace": namespace})

	var cmd []string
	if h.version == V3 {
		cmd = []string{"delete", release, "--namespace", namespace}
	} else {
		cmd = []string{"delete", "--purge", release}
	}

	out, err := RunCmdWithTimeout(ctx, h.helmBin, cmd)
	if err != nil {
		return err
	}
	log.Debug().Print("Result", field.M{"output": out})
	return nil
}

// RemoveRepo removes helm repo
func (h CliClient) RemoveRepo(ctx context.Context, name string) error {
	log.Debug().Print("Removing helm repo", field.M{"name": name})
	out, err := RunCmdWithTimeout(ctx, h.helmBin, []string{"repo", "remove", name})
	if err != nil {
		return err
	}
	log.Debug().Print("Result", field.M{"output": out})
	return nil
}

// RunCmdWithTimeout executes command on host with DefaultCommandTimeout timeout
func RunCmdWithTimeout(ctx context.Context, command string, args []string) (string, error) {
	log.Info().Print("Executing command", field.M{"command": command, "args": args})
	ctx, cancel := context.WithTimeout(ctx, DefaultCommandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
