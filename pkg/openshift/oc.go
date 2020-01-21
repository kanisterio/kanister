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
	"fmt"

	"github.com/kanisterio/kanister/pkg/helm"
)

type OpenShiftClient struct {
}

func NewOpenShiftClient() OSClient {
	return &OpenShiftClient{}
}

func (oc OpenShiftClient) CreateNamespace(ctx context.Context, namespace string) (string, error) {
	command := []string{"create", "namespace", namespace}
	return helm.RunCmdWithTimeout(ctx, "oc", command)
}

func (oc OpenShiftClient) NewApp(ctx context.Context, namespace, osAppImage string, envVar map[string]string) (string, error) {
	var formedVars []string
	for k, v := range envVar {
		formedVars = append(formedVars, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	command := append([]string{"new-app", "-n", namespace, osAppImage}, formedVars...)
	out, err := helm.RunCmdWithTimeout(ctx, "oc", command)

	return out, err
}
