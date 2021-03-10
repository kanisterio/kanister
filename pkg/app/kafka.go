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
	"github.com/kanisterio/kanister/pkg/helm"
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
	topic            string
	kubernetesClient k8s.KubeClient
	chart            helm.ChartInfo
}

func NewKafkaCluster(name string) App {
	return &KafkaCluster{
		name:             name,
		kubernetesClient: k8s.NewkubernetesClient(),
		sinkConfigPath:   "adobe-s3-sink.properties",
		sourceConfigPath: "adobe-s3-source.properties",
		kafkaConfigPath:  "adobe-kafkaConfiguration.properties",
		kafkaYaml:        "kafka-cluster.yaml",
		topic:            "blogs",
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoURL:  helm.KafkaOperatorRepoURL,
			Chart:    "strimzi-kafka-operator",
			RepoName: helm.KafkaOperatorRepoName,
		},
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
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}
	log.Print("Adding repo.", field.M{"app": kc.name})
	err = cli.AddRepo(ctx, kc.chart.RepoName, kc.chart.RepoURL)
	if err != nil {
		return errors.Wrapf(err, "Error helm repo for app %s.", kc.name)
	}

	log.Print("Installing kafka operator using helm.", field.M{"app": kc.name})
	err = cli.Install(ctx, kc.chart.RepoName+"/"+kc.chart.Chart, kc.chart.Version, kc.chart.Release, kc.namespace, kc.chart.Values)
	if err != nil {
		return errors.Wrapf(err, "Error intalling operator %s through helm.", kc.name)
	}

	kubectl := k8s.NewkubernetesClient()
	time.Sleep(1 * time.Second)
	out, err := kubectl.InstallKafka(ctx, namespace, yamlFileRepo, kc.kafkaYaml)
	if err != nil {
		return errors.Wrapf(err, "Error installing the application %s, %s", kc.name, out)
	}
	out, err = kubectl.CreateConfigMap(ctx, configMapName, namespace, yamlFileRepo, kc.kafkaConfigPath, kc.sinkConfigPath, kc.sourceConfigPath)
	if err != nil {
		return errors.Wrapf(err, "Error creating config Map %s, %s", kc.name, out)
	}
	log.Print("Application was installed successfully.", field.M{"app": kc.name})
	return nil
}

// Object return the configmap referred in blueprint
func (kc *KafkaCluster) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		APIVersion: "v1",
		Name:       configMapName,
		Namespace:  kc.namespace,
		Resource:   "configmaps",
	}
}
func (kc *KafkaCluster) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}
func (kc *KafkaCluster) Secrets() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (kc *KafkaCluster) Uninstall(ctx context.Context) error {
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}
	err = cli.Uninstall(ctx, kc.chart.Release, kc.namespace)
	if err != nil {
		log.WithError(err).Print("Failed to uninstall app, you will have to uninstall it manually.", field.M{"app": kc.name})
		return err
	}

	log.Print("Application Deleted successfully.", field.M{"app": kc.name})
	return nil
}

func (kc *KafkaCluster) Ping(ctx context.Context) error {
	log.Print("Pinging the application", field.M{"app": kc.name})
	kubectl := k8s.NewkubernetesClient()
	out, err := kubectl.Ping(ctx, kc.namespace)
	if err != nil {
		return errors.Wrapf(err, "Error Pinging the app for %s, %s.", kc.name, out)
	}
	log.Print("Ping to the application was successful.")
	return nil
}

func (kc *KafkaCluster) Insert(ctx context.Context) error {
	log.Print("Inserting some records in kafka topic.", field.M{"app": kc.name})
	kubectl := k8s.NewkubernetesClient()
	err := kubectl.Insert(ctx, kc.topic, kc.namespace)
	if err != nil {
		return errors.Wrapf(err, "Error Insert the record for %s", kc.name)
	}

	log.Print("Successfully inserted record in the application.", field.M{"app": kc.name})
	return nil
}

func (kc *KafkaCluster) IsReady(ctx context.Context) (bool, error) {
	log.Info().Print("Waiting for application to be ready.", field.M{"app": kc.name})
	ctx, cancel := context.WithTimeout(ctx, kafkaClusterWaitTimeout)
	defer cancel()
	err := kube.WaitOnStatefulSetReady(ctx, kc.cli, kc.namespace, "my-cluster-kafka")
	if err != nil {
		return false, err
	}
	err = kube.WaitOnStatefulSetReady(ctx, kc.cli, kc.namespace, "my-cluster-zookeeper")
	if err != nil {
		return false, err
	}

	log.Print("Application instance is ready.", field.M{"app": kc.name})
	return true, nil
}

func (kc *KafkaCluster) Count(ctx context.Context) (int, error) {
	log.Print("Counting records in kafka topic.", field.M{"app": kc.name})
	kubectl := k8s.NewkubernetesClient()
	count, err := kubectl.Count(ctx, kc.topic, kc.namespace)
	if err != nil {
		return 0, errors.Wrapf(err, "Error Insert the record for %s, %s.", kc.name, err)
	}

	log.Print("Count that we received from application is.", field.M{"app": kc.name, "count": count})
	return count, nil
}

func (kc *KafkaCluster) Reset(ctx context.Context) error {
	log.Print("Resetting the kafka topic is being handled in blueprint.", field.M{"app": kc.name})
	return nil
}

// Initialize is used to initialize the database or create schema
func (kc *KafkaCluster) Initialize(ctx context.Context) error {
	return nil
}
