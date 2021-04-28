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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	//"github.com/segmentio/kafka-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	kafkaClusterWaitTimeout   = 5 * time.Minute
	s3ConnectorYamlFileRepo   = "../../examples/kafka/adobe-s3-connector"
	configMapName             = "s3config"
	s3SinkConfigPath          = "adobe-s3-sink.properties"
	s3SourceConfigPath        = "adobe-s3-source.properties"
	kafkaConfigPath           = "adobe-kafkaConfiguration.properties"
	kafkaYaml                 = "kafka-cluster.yaml"
	topic                     = "blogs"
	chart                     = "strimzi-kafka-operator"
	strimziImage              = "strimzi/kafka:0.20.0-kafka-2.6.0"
	bootstrapServerHost       = "my-cluster-kafka-bootstrap"
	bootstrapServerPort       = "9092"
	bridgeServiceName         = "kafka-bridge-service"
	bridgeServicePort         = "8080"
	strimziOperatorDeployment = "strimzi-cluster-operator"
	kafkaClusterOperator      = "my-cluster-entity-operator"
	kafkaStatefulSet          = "my-cluster-kafka"
	zookeeperStatefulset      = "my-cluster-zookeeper"
)

type KafkaCluster struct {
	cli                kubernetes.Interface
	name               string
	namespace          string
	s3SinkConfigPath   string
	s3SourceConfigPath string
	kafkaConfigPath    string
	pathToYaml         string
	kafkaYaml          string
	topic              string
	chart              helm.ChartInfo
}

func NewKafkaCluster(name, pathToYaml string) App {
	if pathToYaml == "" {
		pathToYaml = s3ConnectorYamlFileRepo
	}
	return &KafkaCluster{
		name:               name,
		s3SinkConfigPath:   s3SinkConfigPath,
		s3SourceConfigPath: s3SourceConfigPath,
		kafkaConfigPath:    kafkaConfigPath,
		kafkaYaml:          kafkaYaml,
		pathToYaml:         pathToYaml,
		topic:              topic,
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
		return errors.Wrapf(err, "Error installing operator %s through helm.", kc.name)
	}
	createKafka := []string{
		"create",
		"-n", namespace,
		"-f", fmt.Sprintf("%s/%s", kc.pathToYaml, kc.kafkaYaml),
	}
	out, err := helm.RunCmdWithTimeout(ctx, "kubectl", createKafka)
	if err != nil {
		return errors.Wrapf(err, "Error installing the application %s, %s", kc.name, out)
	}
	createConfig := []string{
		"create",
		"-n", namespace,
		"configmap", configMapName,
		fmt.Sprintf("--from-file=adobe-s3-sink.properties=%s/%s", kc.pathToYaml, kc.s3SinkConfigPath),
		fmt.Sprintf("--from-file=adobe-s3-source.properties=%s/%s", kc.pathToYaml, kc.s3SourceConfigPath),
		fmt.Sprintf("--from-file=adobe-kafkaConfiguration.properties=%s/%s", kc.pathToYaml, kc.kafkaConfigPath),
		"--from-literal=timeinSeconds=1800",
	}
	out, err = helm.RunCmdWithTimeout(ctx, "kubectl", createConfig)
	if err != nil {
		return errors.Wrapf(err, "Error creating ConfigMap %s, %s", kc.name, out)
	}
	createKafkaBridge := []string{
		"create",
		"-n", namespace,
		"-f", fmt.Sprintf("%s/%s", kc.pathToYaml, "kafka-bridge.yaml"),
	}
	out, err = helm.RunCmdWithTimeout(ctx, "kubectl", createKafkaBridge)

	if err != nil {
		return errors.Wrapf(err, "Error installing the application %s, %s", kc.name, out)
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

	deleteConfig := []string{"delete", "-n", kc.namespace, "configmap", configMapName}
	out, err := helm.RunCmdWithTimeout(ctx, "kubectl", deleteConfig)
	if err != nil {
		return errors.Wrapf(err, "Error deleting ConfigMap %s, %s", kc.name, out)
	}

	err = cli.Uninstall(ctx, kc.chart.Release, kc.namespace)
	if err != nil {
		log.WithError(err).Print("Failed to uninstall app, you will have to uninstall it manually.", field.M{"app": kc.name})
		return err
	}

	log.Print("Application deleted successfully.", field.M{"app": kc.name})
	return nil
}

func (kc *KafkaCluster) Ping(ctx context.Context) error {
	log.Print("Pinging the application", field.M{"app": kc.name})
	pingKafka := []string{
		"run",
		"-n", kc.namespace,
		"kafka-ping",
		"-ti",
		"--rm=true",
		fmt.Sprintf("--image=%s", strimziImage),
		"--restart=Never",
		"--",
		"bin/kafka-topics.sh",
		"--list",
		fmt.Sprintf("--bootstrap-server=%s:%s", bootstrapServerHost, bootstrapServerPort),
	}
	out, err := helm.RunCmdWithTimeout(ctx, "kubectl", pingKafka)
	if err != nil {
		return errors.Wrapf(err, "Error Pinging the app for %s, %s.", kc.name, out)
	}
	log.Print("Ping to the application was successful.")
	return nil
}

func (kc *KafkaCluster) Insert(ctx context.Context) error {
	log.Print("Inserting some records in kafka topic.", field.M{"app": kc.name})

	err := InsertRecord(ctx, kc.namespace)
	if err != nil {
		return errors.Wrapf(err, "Error inserting the record for %s", kc.name)
	}

	log.Print("Successfully inserted record in the application.", field.M{"app": kc.name})
	return nil
}

func (kc *KafkaCluster) IsReady(ctx context.Context) (bool, error) {
	log.Info().Print("Waiting for application to be ready.", field.M{"app": kc.name})
	ctx, cancel := context.WithTimeout(ctx, kafkaClusterWaitTimeout)
	defer cancel()
	err := kube.WaitOnDeploymentReady(ctx, kc.cli, kc.namespace, strimziOperatorDeployment)
	if err != nil {
		return false, err
	}
	err = kube.WaitOnDeploymentReady(ctx, kc.cli, kc.namespace, kafkaClusterOperator)
	if err != nil {
		return false, err
	}
	err = kube.WaitOnStatefulSetReady(ctx, kc.cli, kc.namespace, kafkaStatefulSet)
	if err != nil {
		return false, err
	}
	err = kube.WaitOnStatefulSetReady(ctx, kc.cli, kc.namespace, zookeeperStatefulset)
	if err != nil {
		return false, err
	}
	err = kube.WaitOnDeploymentReady(ctx, kc.cli, kc.namespace, "kafka-bridge")
	if err != nil {
		return false, err
	}
	log.Print("Application instance is ready.", field.M{"app": kc.name})
	return true, nil
}

func (kc *KafkaCluster) Count(ctx context.Context) (int, error) {
	log.Print("Counting records in kafka topic.", field.M{"app": kc.name})

	count, err := ConsumeRecord(ctx, kc.namespace)
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

type Message struct {
	Topic     string `json:"topic"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Partition int    `json:"partition"`
	Offset    int    `json:"offset"`
}

// curl -X POST http://localhost:8080/topics/bridge-quickstart-topic -H 'content-type: application/vnd.kafka.json.v2+json' -d '{"records": [{"key": "my-key","value": "sales-lead-0001"}]}'
type InsertPayload struct {
	Records []Records `json:"records"`
}
type Records struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func InsertRecord(ctx context.Context, namespace string) error {
	log.Print("Inserting record")

	forwarder, err := K8SServicePortForward(ctx, bridgeServiceName, namespace, bridgeServicePort)
	if err != nil {
		return err
	}
	defer forwarder.Close()
	ports, err := forwarder.GetPorts()
	if err != nil {
		return err
	}
	uri := "http://localhost:" + fmt.Sprint(ports[0].Local)

	data := InsertPayload{
		Records: []Records{
			{
				Value: "sales-lead-0001",
			},
		},
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", uri+"/topics/blogs", body)
	log.Print("Inserting record")

	if err != nil {
		fmt.Println("error in craeting request")
		return err
	}
	req.Header.Set("Content-Type", "application/vnd.kafka.json.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("error in sending response")
		return err
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error in raeding bytes")
		return err
	}

	log.Print(string(bytes))

	defer resp.Body.Close()
	return nil
}

// K8SServicePortForward creates a service port forwarding and returns the forwarder and error if any
func K8SServicePortForward(ctx context.Context, svcName string, ns string, pPort string) (*portforward.PortForwarder, error) {
	errCh := make(chan error)
	readyChan := make(chan struct{}, 1)

	cfg, err := kube.LoadConfig()
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

	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(metav1.SetAsLabelSelector(svc.Spec.Selector)),
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
		return nil, errors.Wrapf(err, "Failed to create port forward")
	}

	go func() {
		errCh <- f.ForwardPorts()
	}()

	select {
	case <-readyChan:
		log.Print("PortForward is Ready")
	case err = <-errCh:
		return nil, errors.Wrapf(err, "Failed to get ports from forwarded ports")
	}

	return f, nil
}

// curl -X POST http://localhost:8080/consumers/bridge-quickstart-consumer-group -H 'content-type: application/vnd.kafka.v2+json' -d '{"name": "bridge-quickstart-consumer","auto.offset.reset": "earliest","format": "json","enable.auto.commit": false,"fetch.min.bytes": 512,"consumer.request.timeout.ms": 30000}'

type Payload struct {
	Name                     string `json:"name"`
	AutoOffsetReset          string `json:"auto.offset.reset"`
	Format                   string `json:"format"`
	EnableAutoCommit         bool   `json:"enable.auto.commit"`
	FetchMinBytes            int    `json:"fetch.min.bytes"`
	ConsumerRequestTimeoutMs int    `json:"consumer.request.timeout.ms"`
}

func createConsumerGroup(uri string) error {
	data := Payload{
		Name:                     "blogs-consumer",
		AutoOffsetReset:          "earliest",
		Format:                   "json",
		EnableAutoCommit:         true,
		FetchMinBytes:            512,
		ConsumerRequestTimeoutMs: 30000,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", uri+"/consumers/blogs-consumer-group", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/vnd.kafka.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))
	return nil
}

// curl -X POST http://localhost:8080/consumers/bridge-quickstart-consumer-group/instances/bridge-quickstart-consumer/subscription -H 'content-type: application/vnd.kafka.v2+json' -d '{"topics": ["bridge-quickstart-topic"]}'

type SubscriptionPayload struct {
	Topics []string `json:"topics"`
}

func Subscribe(uri string) error {
	data := SubscriptionPayload{
		// fill struct
		Topics: []string{"blogs"},
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", uri+"/consumers/blogs-consumer-group/instances/blogs-consumer/subscription", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/vnd.kafka.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))
	return nil
}

// curl -X GET http://localhost:8080/consumers/bridge-quickstart-consumer-group/instances/bridge-quickstart-consumer/records   -H 'accept: application/vnd.kafka.json.v2+json'

func getBody(uri string) (int, error) {

	req, err := http.NewRequest("GET", uri+"/consumers/blogs-consumer-group/instances/blogs-consumer/records", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.kafka.json.v2+json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	responseBody := string(bytes)
	var Message []Message
	_ = json.Unmarshal([]byte(responseBody), &Message)
	log.Print(responseBody)

	fmt.Printf("Message : %+v", Message)
	return len(Message), nil
}

func ConsumeRecord(ctx context.Context, namespace string) (int, error) {
	forwarder, err := K8SServicePortForward(ctx, bridgeServiceName, namespace, bridgeServicePort)
	if err != nil {
		return 0, err
	}
	defer forwarder.Close()
	ports, err := forwarder.GetPorts()
	if err != nil {
		return 0, err
	}
	uri := "http://localhost:" + fmt.Sprint(ports[0].Local)
	fmt.Println("consuming records")
	createConsumerGroup(uri)
	fmt.Println("subscribing")
	Subscribe(uri)
	fmt.Println("getting records")
	a, _ := getBody(uri)
	fmt.Println("gotting the records second time")
	for {
		a, _ = getBody(uri)
		if a > 0 {
			log.Print(string(a))
			break
		}
	}
	return a, nil
}
