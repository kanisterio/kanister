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
package app

import (
	"context"
	"time"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	k8s "github.com/kanisterio/kanister/pkg/kubernetes"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	kafkaClusterWaitTimeout = 5 * time.Minute
	yamlFileRepo            = "../../examples/kafka/adobe-s3-connector"
	configMapName           = "s3config"
)

type KafkaCluster struct {
	cli              kubernetes.Interface
	name             string
	namespace        string
	sinkConfigPath   string
	sourceConfigPath string
	kafkaConfigPath  string
	kafkaYaml        string
	strimziYaml      string
	kubernetesClient k8s.KubeClient
}

func NewKafkaCluster(name string) App {
	return &KafkaCluster{
		name:             name,
		kubernetesClient: k8s.NewkubernetesClient(),
		sinkConfigPath:   "adobe-s3-sink.properties",
		sourceConfigPath: "adobe-s3-source.properties",
		kafkaConfigPath:  "adobe-kafkaConfiguration.properties",
		kafkaYaml:        "kafka-cluster.yaml",
		strimziYaml:      "https://strimzi.io/install/latest?namespace=kafka",
	}
}
func (kc *KafkaCluster) Init(context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	kc.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	return err
}
func (kc *KafkaCluster) Install(ctx context.Context, namespace string) error {
	kc.namespace = namespace
	kubectl := k8s.NewkubernetesClient()
	out, err := kubectl.InstallOperator(ctx, namespace, yamlFileRepo, kc.strimziYaml)
	if err != nil {
		return errors.Wrapf(err, "Error installing the operator for %s, error is %s.", kc.name, out)
	}
	out, err = kubectl.InstallKafka(ctx, namespace, yamlFileRepo, kc.kafkaYaml)
	if err != nil {
		return errors.Wrapf(err, "Error installing the application %s, error is %s", kc.name, out)
	}
	out, err = kubectl.CreateConfigMap(ctx, namespace, yamlFileRepo, kc.kafkaConfigPath, kc.sinkConfigPath, kc.sourceConfigPath)
	if err != nil {
		return errors.Wrapf(err, "Error creating config Map %s, error is %s", kc.name, out)
	}
	log.Print("Application was installed successfully.", field.M{"app": kc.name})
	return nil
}

// Object return the configmap referred in blueprint
func (kc *KafkaCluster) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "configMap",
		Name:      configMapName,
		Namespace: kc.namespace,
	}
}
func (kc *KafkaCluster) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}
func (kc *KafkaCluster) Secrets() map[string]crv1alpha1.ObjectReference {
	return nil
}

// TODO
func (kc *KafkaCluster) Uninstall(ctx context.Context) error {
	kubectl := k8s.NewkubernetesClient()
	out, err := kubectl.DeleteConfigMap(ctx, kc.namespace, configMapName)
	if err != nil {
		return errors.Wrapf(err, "Error creating config Map %s, error is %s", kc.name, out)
	}
	out, err = kubectl.DeleteKafka(ctx, kc.namespace, yamlFileRepo, kc.kafkaYaml)
	if err != nil {
		return errors.Wrapf(err, "Error installing the application %s, error is %s", kc.name, out)
	}
	out, err = kubectl.DeleteOperator(ctx, kc.namespace, yamlFileRepo, kc.strimziYaml)
	if err != nil {
		return errors.Wrapf(err, "Error installing the operator for %s, error is %s.", kc.name, out)
	}
	log.Print("Application Deleted successfully.", field.M{"app": kc.name})
	return nil
}
func (kc *KafkaCluster) Ping(ctx context.Context) error {
	log.Print("Pinging the application", field.M{"app": kc.name})
	listTopic := []string{"sh", "-c"}
	_, stderr, err := kc.execCommand(ctx, listTopic)
	if err != nil {
		return errors.Wrapf(err, "Error while Pinging the database %s", stderr)
	}
	log.Print("Ping to the application was successful.")
	return nil
}

// TODO
func (kc *KafkaCluster) Insert(ctx context.Context) error {
	log.Print("Inserting some records in kafka topic.", field.M{"app": kc.name})
	log.Print("Successfully inserted record in the application.", field.M{"app": kc.name})
	return nil
}

// TODO
func (kc *KafkaCluster) IsReady(ctx context.Context) (bool, error) {
	log.Info().Print("Waiting for application to be ready.", field.M{"app": kc.name})
	ctx, cancel := context.WithTimeout(ctx, kafkaClusterWaitTimeout)
	defer cancel()
	log.Print("Application is ready", field.M{"application": kc.name})
	return true, nil
}

// TODO
func (kc *KafkaCluster) Count(ctx context.Context) (int, error) {
	log.Print("Counting the records from the mysql instance.", field.M{"app": kc.name})
	log.Print("Count that we received from application is.", field.M{"app": kc.name, "count": nil})
	return 0, nil
}

// TODO
func (kc *KafkaCluster) Reset(ctx context.Context) error {
	log.Print("Resetting the mysql instance.", field.M{"app": "mysql"})
	// delete all the data from the table
	log.Print("Reset of the application was successful.", field.M{"app": kc.name})
	return nil
}

// TODO
// Initialize is used to initialize the database or create schema
func (kc *KafkaCluster) Initialize(ctx context.Context) error {
	return nil
}
func (kc *KafkaCluster) execCommand(ctx context.Context, command []string) (string, string, error) {
	return kube.Exec(kc.cli, kc.namespace, "kafka-producer", "kafka-producer", command, nil)
}
