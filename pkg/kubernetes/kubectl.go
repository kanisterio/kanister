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
	"log"
	"os"
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
	createOperator := []string{"create", "-n", namespace, "-f", fmt.Sprintf("%s", strimziYaml)}
	return helm.RunCmdWithTimeout(ctx, "kubectl", createOperator)
}
func (kubectl kubernetesClient) InstallKafka(ctx context.Context, namespace, yamlFileRepo, kafkaConfigPath string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	createKafka := []string{"create", "-n", namespace, "-f", fmt.Sprintf("%s/%s", yamlFileRepo, kafkaConfigPath)}
	file, err := os.Open(fmt.Sprintf("%s/%s", yamlFileRepo, kafkaConfigPath))
	if err == nil {
		log.Print(file)
	}
	return helm.RunCmdWithTimeout(ctx, "kubectl", createKafka)
}
func (kubectl kubernetesClient) CreateConfigMap(ctx context.Context, configMapName, namespace, yamlFileRepo, sinkConfigPath, sourceConfigPath, kafkaConfigPath string) (string, error) {
	//	createConfig := []string{"create", "configmap", "s3config", fmt.Sprintf("--from-file=%s/%s --from-file=%s/%s --from-file=%s/%s --from-literal=timeinSeconds=1800", yamlFileRepo, sinkConfigPath, yamlFileRepo, sourceConfigPath, yamlFileRepo, sinkConfigPath), "-n", namespace}
	createConfig := []string{"create", "-n", namespace, "configmap", configMapName, "--", "from-file=", fmt.Sprintf("%s/%s", yamlFileRepo, sinkConfigPath)}
	return helm.RunCmdWithTimeout(ctx, "kubectl", createConfig)
}
func (kubectl kubernetesClient) DeleteConfigMap(ctx context.Context, namespace, configMapName string) (string, error) {
	deleteConfig := []string{"delete", "-n", namespace, "configmap", configMapName}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteConfig)
}
func (kubectl kubernetesClient) DeleteOperator(ctx context.Context, namespace, yamlFileRepo, strimziYaml string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	deleteOperator := []string{"delete", "-n", namespace, "-f", fmt.Sprintf("%s", strimziYaml)}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteOperator)
}
func (kubectl kubernetesClient) DeleteKafka(ctx context.Context, namespace, yamlFileRepo, kafkaConfigPath string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	deleteKafka := []string{"delete", "-n", namespace, "-f", fmt.Sprintf("%s/%s", yamlFileRepo, kafkaConfigPath)}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteKafka)
}

func (kubectl kubernetesClient) Ping(ctx context.Context, namespace string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	pingKafka := []string{"run", "-n", namespace, "kafka-producer", "-ti", "--rm=true", "--image=strimzi/kafka:0.20.0-kafka-2.6.0", "--restart=Never", "--", "bin/kafka-topics.sh", "--create", "--topic", "integration-test", "--bootstrap-server=my-cluster-kafka-bootstrap:9092"}
	log.Print(pingKafka)
	return helm.RunCmdWithTimeout(ctx, "kubectl", pingKafka)
}
func (kubectl kubernetesClient) Insert(ctx context.Context, namespace string) (string, error) {

	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	insertKafka := []string{"run", "-n", namespace, "kafka-producer", "-ti", "--rm=true", "--image=strimzi/kafka:0.20.0-kafka-2.6.0", "--restart=Never", "--", "bin/kafka-console-producer.sh", "--topic", "integration-test", "--broker-list=my-cluster-kafka-bootstrap:9092"}
	log.Print(insertKafka)
	return helm.RunCmdWithInput(ctx, "kubectl", insertKafka, "abcd")

}
