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

// NewApp install a new application in the openshift
// cluster using "oc new-app" command
func (oc OpenShiftClient) NewApp(ctx context.Context, namespace, dpTemplate string, envVar, params map[string]string) (string, error) {
	var formedVars, formedParams []string
	for k, v := range envVar {
		formedVars = append(formedVars, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range params {
		formedParams = append(formedParams, "-p", fmt.Sprintf("%s=%s", k, v))
	}

	// append env variables in the command
	command := append([]string{"new-app", "-n", namespace, dpTemplate}, formedVars...)
	// append parameters in the command
	command = append(command, formedParams...)
	return helm.RunCmdWithTimeout(ctx, "oc", command)
}

// DeleteApp deletes an application that is deployed on OpenShift cluster
// oc delete all -n <ns-name> -l <label>
// openshift adds a specific label to all the resources, while deploying an application
func (oc OpenShiftClient) DeleteApp(ctx context.Context, namespace, label string) (string, error) {
	delCommand := []string{"delete", "all", "-n", namespace, "-l", label}
	return helm.RunCmdWithTimeout(ctx, "oc", delCommand)
}
