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

package openshift

import (
	"context"
)

type OSClient interface {
	// CreateNamespace creates a namespace in the openshift cluster
	CreateNamespace(ctx context.Context, namespace string) (string, error)

	// NewApp installs new app in the openshift clsuter
	// similar to oc new-app ...
	NewApp(ctx context.Context, namespace, dpTemplate string, envVar, params map[string]string) (string, error)

	// DeleteApp delete an app from the openshift cluster
	// similar to oc delete all -n <ns> -l <label>
	DeleteApp(ctx context.Context, namespace, label string) (string, error)
}
