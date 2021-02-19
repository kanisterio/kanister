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
package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/kanisterio/kanister/pkg/helm"
)

const (
	kafkaClusterWaitTimeout = 5 * time.Minute
)

type kubernetesClient struct {
}

func NewkubernetesClient() KubeClient {
	return &kubernetesClient{}
}

// CreateNamespace creates required namespace in k8s cluster
func (kubectl kubernetesClient) CreateNamespace(ctx context.Context, namespace string) (string, error) {
	command := []string{"create", "namespace", namespace}
	return helm.RunCmdWithTimeout(ctx, "kubectl", command)
}

// InstallOPerator installs strimzi operator
func (kubectl kubernetesClient) InstallOperator(ctx context.Context, namespace, yamlFileRepo, strimziYaml string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	createOperator := []string{"create", "-f", fmt.Sprintf("%s/%s", yamlFileRepo, strimziYaml), "-n", namespace}
	return helm.RunCmdWithTimeout(ctx, "kubectl", createOperator)
}
func (kubectl kubernetesClient) InstallKafka(ctx context.Context, namespace, yamlFileRepo, kafkaConfigPath string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	createKafka := []string{"create", "-f", fmt.Sprintf("%s/%s", yamlFileRepo, kafkaConfigPath), "-n", namespace}
	return helm.RunCmdWithTimeout(ctx, "kubectl", createKafka)
}
func (kubectl kubernetesClient) CreateConfigMap(ctx context.Context, namespace, yamlFileRepo, sinkConfigPath, sourceConfigPath, kafkaConfigPath string) (string, error) {
	createConfig := []string{"create configmap", "s3config", "--from-file=", fmt.Sprintf("%s/%s", yamlFileRepo, sinkConfigPath), "--from-file=", fmt.Sprintf("%s/%s", yamlFileRepo, sourceConfigPath), "--from-file=", fmt.Sprintf("%s/%s", yamlFileRepo, sinkConfigPath), "--from-literal=timeinSeconds=1800", "-n", namespace}
	return helm.RunCmdWithTimeout(ctx, "kubectl", createConfig)
}
func (kubectl kubernetesClient) DeleteConfigMap(ctx context.Context, namespace, configMapName string) (string, error) {
	deleteConfig := []string{"delete configmap", configMapName, "-n", namespace}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteConfig)
}
func (kubectl kubernetesClient) DeleteOperator(ctx context.Context, namespace, yamlFileRepo, strimziYaml string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	deleteOperator := []string{"delete", "-f", fmt.Sprintf("%s/%s", yamlFileRepo, strimziYaml), "-n", namespace}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteOperator)
}
func (kubectl kubernetesClient) DeleteKafka(ctx context.Context, namespace, yamlFileRepo, kafkaConfigPath string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	deleteKafka := []string{"create", "-f", fmt.Sprintf("%s/%s", yamlFileRepo, kafkaConfigPath), "-n", namespace}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteKafka)
}
