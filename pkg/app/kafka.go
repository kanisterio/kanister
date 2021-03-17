// Copyright 2021 The Kanister Authors.
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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	kafkaClusterWaitTimeout = 5 * time.Minute
	yamlFileRepo            = "../../examples/kafka/adobe-s3-connector"
	configMapName           = "s3config"
	sinkConfigPath          = "adobe-s3-sink.properties"
	sourceConfigPath        = "adobe-s3-source.properties"
	kafkaConfigPath         = "adobe-kafkaConfiguration.properties"
	crdYaml                 = "CRD.yaml"
	kafkaYaml               = "kafka-cluster.yaml"
	topic                   = "blogs"
	chart                   = "strimzi-kafka-operator"
)

type KafkaCluster struct {
	cli              kubernetes.Interface
	name             string
	namespace        string
	sinkConfigPath   string
	sourceConfigPath string
	kafkaConfigPath  string
	pathToYaml       string
	crdYaml          string
	kafkaYaml        string
	topic            string
	chart            helm.ChartInfo
}

func NewKafkaCluster(name, pathToYaml string) App {
	if pathToYaml == "" {
		pathToYaml = yamlFileRepo
	}
	return &KafkaCluster{
		name:             name,
		sinkConfigPath:   sinkConfigPath,
		sourceConfigPath: sourceConfigPath,
		kafkaConfigPath:  kafkaConfigPath,
		kafkaYaml:        kafkaYaml,
		pathToYaml:       pathToYaml,
		crdYaml:          crdYaml,
		topic:            topic,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoURL:  helm.KafkaOperatorRepoURL,
			Chart:    chart,
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
	createKafka := []string{"create", "-n", namespace, "-f", fmt.Sprintf("%s/%s", kc.pathToYaml, kc.kafkaYaml)}
	out, err := helm.RunCmdWithTimeout(ctx, "kubectl", createKafka)
	if err != nil {
		return errors.Wrapf(err, "Error installing the application %s, %s", kc.name, out)
	}
	createConfig := []string{"create", "-n", namespace, "configmap", configMapName, fmt.Sprintf("--from-file=adobe-s3-sink.properties=%s/%s", kc.pathToYaml, kc.sinkConfigPath), fmt.Sprintf("--from-file=adobe-s3-source.properties=%s/%s", kc.pathToYaml, kc.sourceConfigPath), fmt.Sprintf("--from-file=adobe-kafkaConfiguration.properties=%s/%s", kc.pathToYaml, kc.kafkaConfigPath), "--from-literal=timeinSeconds=1800"}
	out, err = helm.RunCmdWithTimeout(ctx, "kubectl", createConfig)
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
	deleteConfig := []string{"delete", "-n", kc.namespace, "configmap", configMapName}
	out, err := helm.RunCmdWithTimeout(ctx, "kubectl", deleteConfig)
	if err != nil {
		return errors.Wrapf(err, "Error deleting config Map %s, %s", kc.name, out)
	}
	deleteCRD := []string{"delete", "-f", fmt.Sprintf("%s/%s", yamlFileRepo, crdYaml)}
	out, err = helm.RunCmdWithTimeout(ctx, "kubectl", deleteCRD)
	if err != nil {
		return errors.Wrapf(err, "Error deleting CRD %s, %s", kc.name, out)
	}

	log.Print("Application Deleted successfully.", field.M{"app": kc.name})
	return nil
}

func (kc *KafkaCluster) Ping(ctx context.Context) error {
	log.Print("Pinging the application", field.M{"app": kc.name})
	pingKafka := []string{"run", "-n", kc.namespace, "kafka-ping", "-ti", "--rm=true", "--image=strimzi/kafka:0.20.0-kafka-2.6.0", "--restart=Never", "--", "bin/kafka-topics.sh", "--list", "--bootstrap-server=my-cluster-kafka-external-bootstrap:9094"}
	out, err := helm.RunCmdWithTimeout(ctx, "kubectl", pingKafka)
	if err != nil {
		return errors.Wrapf(err, "Error Pinging the app for %s, %s.", kc.name, out)
	}
	log.Print("Ping to the application was successful.")
	return nil
}

func (kc *KafkaCluster) Insert(ctx context.Context) error {
	log.Print("Inserting some records in kafka topic.", field.M{"app": kc.name})

	err := produce(ctx, kc.topic, kc.namespace)
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
	count, err := consume(ctx, kc.topic, kc.namespace)
	if err != nil {
		return 0, errors.Wrapf(err, "Error counting the records for %s, %s.", kc.name, err)
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

// to produce messages to kafka
func produce(ctx context.Context, topic string, namespace string) error {
	partition := 0
	forwarder, err := K8SServicePortForward(ctx, "my-cluster-kafka-external-bootstrap", namespace, "9094")
	if err != nil {
		return err
	}

	defer forwarder.Close()
	ports, err := forwarder.GetPorts()
	if err != nil {
		return err
	}
	uri := "localhost:" + fmt.Sprint(ports[0].Local)

	conn, err := kafka.DialLeader(ctx, "tcp", uri, topic, partition)
	if err != nil {
		return err
	}
	err = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return err
	}
	_, err = conn.WriteMessages(
		kafka.Message{Value: []byte("{'title':'The Matrix','year':1999,'cast':['Keanu Reeves','Laurence Fishburne','Carrie-Anne Moss','Hugo Weaving','Joe Pantoliano'],'genres':['Science Fiction']}")},
	)
	if err != nil {
		return err
	}

	if err := conn.Close(); err != nil {
		return err
	}
	return nil
}

// LoadConfig returns a kubernetes client config based on global settings.
func LoadConfig() (*rest.Config, error) {
	//	log.Print("Attempting to use InCluster config")
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	homeConfig := filepath.Join(os.Getenv("HOME"), ".kube/config")
	config, err = clientcmd.BuildConfigFromFlags("", homeConfig)
	if err == nil {
		return config, nil
	}
	return nil, errors.Wrapf(err, "Failed to load default k8s config")
}

// K8SServicePortForward creates a service port forwarding and returns the forwarder and the error if any
func K8SServicePortForward(ctx context.Context, svcName string, ns string, pPort string) (*portforward.PortForwarder, error) {
	var selector string
	errCh := make(chan error)
	readyChan := make(chan struct{}, 1)

	cfg, err := LoadConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to Load config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Clienset for k8s config")
	}
	roundTripper, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create RoundTripper for k8s config")
	}

	svc, err := clientset.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get service for component")
	}
	for k, v := range svc.Spec.Selector {
		selector = fmt.Sprintf("%s%s=%s,", selector, k, v)
	}
	selector = strings.TrimSuffix(selector, ",")
	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to list pods for component")
	}
	if len(pods.Items) == 0 {
		return nil, errors.Wrapf(err, "Empty pods list for component")
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", ns, pods.Items[0].Name)
	u, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse url struct from k8s config")
	}
	hostIP := fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	pre, pwe := io.Pipe()
	pro, pwo := io.Pipe()
	go func() {
		buf := bytes.NewBuffer(nil)
		if _, inErr := buf.ReadFrom(pre); inErr != nil {
			log.Print(inErr.Error())
		}
	}()
	go func() {
		buf := bytes.NewBuffer(nil)
		if _, inErr := buf.ReadFrom(pro); inErr != nil {
			log.Print(inErr.Error())
		}
	}()

	f, err := portforward.New(dialer, []string{fmt.Sprintf(":%s", pPort)}, ctx.Done(), readyChan, pwo, pwe)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create portforward")
	}

	go func() {
		errCh <- f.ForwardPorts()
	}()

	select {
	case <-readyChan:
		log.Print("PortForward is Ready")
	case err = <-errCh:
		return nil, errors.Wrapf(err, "Failed to get Ports from forwarded")
	}

	return f, nil
}

func consume(ctx context.Context, topic string, namespace string) (int, error) {
	partition := 0
	forwarder, err := K8SServicePortForward(ctx, "my-cluster-kafka-external-bootstrap", namespace, "9094")
	if err != nil {
		log.Print("error getting forwarder")
		return 0, err
	}

	defer forwarder.Close()
	ports, err := forwarder.GetPorts()
	if err != nil {
		return 0, err
	}
	uri := "localhost:" + fmt.Sprint(ports[0].Local)

	conn, err := kafka.DialLeader(ctx, "tcp", uri, topic, partition)
	if err != nil {
		return 0, err
	}

	err = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		//		log.Print("failed to set ReadDeadline:", err)
		return 0, err
	}
	batch := conn.ReadBatch(10e3, 1e6) // fetch 10KB min, 1MB max
	count := 0
	b := make([]byte, 10e3) // 10KB max per message
	messageEnd := false
	for {
		_, err := batch.Read(b)
		if err != nil {
			messageEnd = true
			break
		}
		count = count + 1
		fmt.Println(string(b))
	}
	if !messageEnd {
		if err := batch.Close(); err != nil {
			return 0, err
		}
	}
	if err := conn.Close(); err != nil {
		return 0, err
	}
	return count, nil
}
