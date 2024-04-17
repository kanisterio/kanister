// Copyright 2022 The Kanister Authors.
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

	"github.com/kanisterio/kanister/pkg/helm"
)

type HelmApp struct {
	name       string
	chart      string
	version    string
	helmValues map[string]string
	dryRun     bool
	client     helm.Client
	namespace  string
}

func NewHelmApp(values map[string]string, name, chart, ns, version string, dr bool) (*HelmApp, error) {
	cli, err := helm.NewCliClient()
	if err != nil {
		return nil, err
	}
	return &HelmApp{
		name:       name,
		chart:      chart,
		helmValues: values,
		client:     cli,
		namespace:  ns,
		version:    version,
		dryRun:     dr,
	}, nil
}

func (h *HelmApp) AddRepo(name, url string) error {
	return h.client.AddRepo(context.Background(), name, url)
}

func (h *HelmApp) Install() (string, error) {
	ctx := context.Background()
	return h.client.Install(ctx, h.chart, "", h.name, h.namespace, h.helmValues, true, h.dryRun)
}

func (h *HelmApp) Upgrade(chart string, updatedValues map[string]string) error {
	ctx := context.Background()
	return h.client.Upgrade(ctx, chart, "", h.name, h.namespace, updatedValues)
}

func (h *HelmApp) Uninstall() error {
	return h.client.Uninstall(context.Background(), h.name, h.namespace)
}
