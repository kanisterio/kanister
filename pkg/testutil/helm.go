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
	"context"
	"fmt"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type HelmClient struct{}

func NewHelmClient() HelmClient {
	return HelmClient{}
}

// AddRepo adds new helm repo and fetches latest charts
func (h *HelmClient) AddRepo(ctx context.Context, name, url string) error {
	log.Debug().Print("Adding helm repo", field.M{"name": name, "url": url})
	out, err := RunCmdWithTimeout(ctx, "helm", []string{"repo", "add", name, url})
	if err != nil {
		return err
	}
	log.Debug().Print("Result", field.M{"output": out})

	// Update all repos to fetch the latest charts
	h.UpdateRepo(ctx)
	return nil
}

// UpdateRepo fetches latest helm charts from the repo
func (h HelmClient) UpdateRepo(ctx context.Context) error {
	log.Debug().Print("Fetching latest helm charts from the helm repos")
	out, err := RunCmdWithTimeout(ctx, "helm", []string{"repo", "update"})
	if err != nil {
		return err
	}
	log.Debug().Print("Result", field.M{"output": out})
	return nil
}

// Install installs helm chart with given release name
func (h HelmClient) Install(ctx context.Context, chart, release, namespace string, values map[string]string) error {
	log.Debug().Print("Installing helm chart", field.M{"chart": chart, "release": release, "namespace": namespace})
	var setVals string
	for k, v := range values {
		setVals += fmt.Sprintf("%s=%s,", k, v)
	}
	out, err := RunCmdWithTimeout(ctx, "helm", []string{"install", release, "--namespace", namespace, chart, "--set", setVals})
	if err != nil {
		return err
	}
	log.Debug().Print("Result", field.M{"output": out})
	return nil
}

// Uninstall deletes helm release
func (h HelmClient) Uninstall(ctx context.Context, release, namespace string) error {
	log.Debug().Print("Uninstalling helm chart", field.M{"release": release, "namespace": namespace})
	out, err := RunCmdWithTimeout(ctx, "helm", []string{"delete", release, "--namespace", namespace})
	if err != nil {
		return err
	}
	log.Debug().Print("Result", field.M{"output": out})
	return nil
}

// RemoveRepo removes helm repo
func (h *HelmClient) RemoveRepo(ctx context.Context, name string) error {
	log.Debug().Print("Removing helm repo", field.M{"name": name})
	out, err := RunCmdWithTimeout(ctx, "helm", []string{"repo", "remove", name})
	if err != nil {
		return err
	}
	log.Debug().Print("Result", field.M{"output": out})
	return nil
}
