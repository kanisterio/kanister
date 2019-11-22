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

import "context"

// Client implements methods to manage helm repos and helm charts
type Client interface {
	// AddRepo adds new helm repo and fetches latest charts
	AddRepo(ctx context.Context, name, url string) error
	// UpdateRepo fetches latest helm charts from the repo
	// Should be called whenever new repo is added
	UpdateRepo(context.Context) error
	// RemoveRepo removes helm repo
	RemoveRepo(ctx context.Context, name string) error
	// Install installs helm chart with given release name in the namespace
	Install(ctx context.Context, chart, release, namespace string, values map[string]string) error
	// Uninstall deletes helm release from the given namespace
	Uninstall(ctx context.Context, release, namespace string) error
}

// ChartInfo holds information to fetch and install helm chart
type ChartInfo struct {
	Release  string
	Chart    string
	RepoUrl  string
	RepoName string
	Values   map[string]string
}
