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
	"net/http"
	"net/url"
	"time"

	"github.com/kanisterio/errkit"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

const consumeTimeout = 5 * time.Minute

const (
	kafkaClusterWaitTimeout    = 5 * time.Minute
	s3ConnectorYamlFileRepo    = "../../examples/kafka/adobe-s3-connector"
	configMapName              = "s3config"
	s3SinkConfigPath           = "adobe-s3-sink.properties"
	s3SourceConfigPath         = "adobe-s3-source.properties"
	kafkaConfigPath            = "adobe-kafkaConfiguration.properties"
	kafkaYaml                  = "kafka-cluster.yaml"
	topic                      = "blogs"
	chart                      = "strimzi-kafka-operator"
	strimziImage               = "strimzi/kafka:0.20.0-kafka-2.6.0"
	bootstrapServerHost        = "my-cluster-kafka-bootstrap"
	bootstrapServerPort        = "9092"
	bridgeServiceName          = "kafka-bridge-service"
	bridgeServicePort          = "8080"
	strimziOperatorDeployment  = "strimzi-cluster-operator"
	kafkaClusterOperator       = "my-cluster-entity-operator"
	kafkaStatefulSet           = "my-cluster-kafka"
	zookeeperStatefulset       = "my-cluster-zookeeper"
	kafkaBridgeDeployment      = "kafka-bridge"
	kafkaBridge                = "kafka-bridge.yaml"
	subscriptionURLPathFormat  = "/consumers/%s-consumer-group/instances/%s-consumer/subscription"
	consumerGroupURLPathFormat = "/consumers/%s-consumer-group"
	consumeTopicMessage        = "/consumers/%s-consumer-group/instances/%s-consumer/records"
)

type KafkaCluster struct {
	cli                kubernetes.Interface
	dynClient          dynamic.Interface
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
	if err != nil {
		return errkit.Wrap(err, "failed to get a k8s client")
	}
	kc.dynClient, err = dynamic.NewForConfig(cfg)
	if err != nil {
		return errkit.Wrap(err, "failed to get a k8s dynamic client")
	}
	return nil
}

func (kc *KafkaCluster) Install(ctx context.Context, namespace string) error {
	kc.namespace = namespace
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}
	log.Print("Adding repo.", field.M{"app": kc.name})
	err = cli.AddRepo(ctx, kc.chart.RepoName, kc.chart.RepoURL)
	if err != nil {
		return errkit.Wrap(err, "Error adding helm repo for app.", "app", kc.name)
	}
	log.Print("Installing kafka operator using helm.", field.M{"app": kc.name})
	_, err = cli.Install(ctx, kc.chart.RepoName+"/"+kc.chart.Chart, kc.chart.Version, kc.chart.Release, kc.namespace, kc.chart.Values, true, false)
	if err != nil {
		return errkit.Wrap(err, "Error installing operator through helm.", "app", kc.name)
	}
	createKafka := []string{
		"create",
		"-n", namespace,
		"-f", fmt.Sprintf("%s/%s", kc.pathToYaml, kc.kafkaYaml),
	}
	out, err := helm.RunCmdWithTimeout(ctx, "kubectl", createKafka)
	if err != nil {
		return errkit.Wrap(err, "Error installing the application", "app", kc.name, "out", out)
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
		return errkit.Wrap(err, "Error creating ConfigMap", "app", kc.name, "out", out)
	}
	createKafkaBridge := []string{
		"create",
		"-n", namespace,
		"-f", fmt.Sprintf("%s/%s", kc.pathToYaml, kafkaBridge),
	}
	out, err = helm.RunCmdWithTimeout(ctx, "kubectl", createKafkaBridge)

	if err != nil {
		return errkit.Wrap(err, "Error installing the application", "app", kc.name, "out", out)
	}
	log.Print("Application was installed successfully.", field.M{"app": kc.name})
	return nil
}

func (kc *KafkaCluster) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return []crv1alpha1.ObjectReference{
		// ClusterRoles
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-kafka-client",
			Resource:   "clusterroles",
		},
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-cluster-operator-global",
			Resource:   "clusterroles",
		},
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-cluster-operator-namespaced",
			Resource:   "clusterroles",
		},
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-entity-operator",
			Resource:   "clusterroles",
		},
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-topic-operator",
			Resource:   "clusterroles",
		},
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-kafka-broker",
			Resource:   "clusterroles",
		},

		// ClusterRoleBindings
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-cluster-operator",
			Resource:   "clusterrolebindings",
		},
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-cluster-operator-kafka-broker-delegation",
			Resource:   "clusterrolebindings",
		},
		{
			APIVersion: "v1",
			Group:      "rbac.authorization.k8s.io",
			Name:       "strimzi-cluster-operator-kafka-client-delegation",
			Resource:   "clusterrolebindings",
		},

		// CRDs
		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkabridges.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},
		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkaconnectors.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},
		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkaconnects.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},
		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkamirrormaker2s.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},
		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkamirrormakers.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},
		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkarebalances.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},
		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkas.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},

		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkatopics.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},
		{
			APIVersion: "v1",
			Group:      "apiextensions.k8s.io",
			Name:       "kafkausers.kafka.strimzi.io",
			Resource:   "customresourcedefinitions",
		},
	}
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
		return errkit.Wrap(err, "failed to create helm client")
	}

	err = kc.cli.CoreV1().ConfigMaps(kc.namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Error deleting ConfigMap", "configMap", configMapName)
	}

	err = cli.Uninstall(ctx, kc.chart.Release, kc.namespace)
	if err != nil {
		log.WithError(err).Print("Failed to uninstall app, you will have to uninstall it manually.", field.M{"app": kc.name})
		return err
	}

	var gvr schema.GroupVersionResource
	ClusterLevelResources := kc.GetClusterScopedResources(ctx)
	// delete ClusterScopedResources if present
	for _, clr := range ClusterLevelResources {
		gvr = schema.GroupVersionResource{Group: clr.Group, Version: clr.APIVersion, Resource: clr.Resource}
		err = kc.dynClient.Resource(gvr).Delete(ctx, clr.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			log.WithError(err).Print("Failed to delete cluster resource", field.M{"app": kc.name})
			return err
		}
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
		return errkit.Wrap(err, "Error Pinging the app", "app", kc.name, "out", out)
	}
	log.Print("Ping to the application was successful.")
	return nil
}

func (kc *KafkaCluster) Insert(ctx context.Context) error {
	log.Print("Inserting some records in kafka topic.", field.M{"app": kc.name})

	err := kc.InsertRecord(ctx, kc.namespace)
	if err != nil {
		return errkit.Wrap(err, "Error inserting the record for", "app", kc.name)
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
	err = kube.WaitOnDeploymentReady(ctx, kc.cli, kc.namespace, kafkaBridgeDeployment)
	if err != nil {
		return false, err
	}
	log.Print("Application instance is ready.", field.M{"app": kc.name})
	return true, nil
}

func (kc *KafkaCluster) Count(ctx context.Context) (int, error) {
	log.Print("Counting records in kafka topic.", field.M{"app": kc.name})

	count, err := consumeTopic(ctx, kc.namespace)
	if err != nil {
		return 0, errkit.Wrap(err, "Error counting the records.", "app", kc.name)
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

// Message describes the response we get after consuming a topic
type Message struct {
	Topic     string `json:"topic"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Partition int    `json:"partition"`
	Offset    int    `json:"offset"`
}

// InsertPayload is a list of Records passed to a topic
type InsertPayload struct {
	Records []Records `json:"records"`
}

// Records takes value and place that to a partition in a topic based on hash of the key
type Records struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Payload sets the consumer configuration
type Payload struct {
	Name                     string `json:"name"`
	AutoOffsetReset          string `json:"auto.offset.reset"`
	Format                   string `json:"format"`
	EnableAutoCommit         bool   `json:"enable.auto.commit"`
	FetchMinBytes            int    `json:"fetch.min.bytes"`
	ConsumerRequestTimeoutMs int    `json:"consumer.request.timeout.ms"`
}

// SubscriptionPayload takes the list of topic names to subscribe
type SubscriptionPayload struct {
	Topics []string `json:"topics"`
}

// InsertRecord creates a topic and insert a record
func (kc *KafkaCluster) InsertRecord(ctx context.Context, namespace string) error {
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

	req, err := http.NewRequest("POST", uri+"/topics/"+topic, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/vnd.kafka.json.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errkit.New("Error inserting records in topic")
	}
	defer resp.Body.Close() //nolint:errcheck
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Debug().Print(string(bytes))
	return nil
}

// K8SServicePortForward creates a service port forwarding and returns the forwarder and error if any
func K8SServicePortForward(ctx context.Context, svcName string, ns string, pPort string) (*portforward.PortForwarder, error) {
	errCh := make(chan error)
	readyChan := make(chan struct{}, 1)

	cfg, err := kube.LoadConfig()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to Load config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)

	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Clienset for k8s config")
	}
	roundTripper, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create RoundTripper for k8s config")
	}

	svc, err := clientset.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get service for component")
	}

	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(metav1.SetAsLabelSelector(svc.Spec.Selector)),
	})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to list pods for component")
	}
	if len(pods.Items) == 0 {
		return nil, errkit.Wrap(err, "Empty pods list for component")
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", ns, pods.Items[0].Name)
	u, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to parse url struct from k8s config")
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
		return nil, errkit.Wrap(err, "Failed to create port forward")
	}

	go func() {
		errCh <- f.ForwardPorts()
	}()

	select {
	case <-readyChan:
		log.Print("PortForward is Ready")
	case err = <-errCh:
		return nil, errkit.Wrap(err, "Failed to get ports from forwarded ports")
	}

	return f, nil
}

// createConsumerGroup creates a consumer group in kafka cluster
func createConsumerGroup(uri string) error {
	data := Payload{
		Name:                     topic + "-consumer",
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
	req, err := http.NewRequest("POST", uri+fmt.Sprintf(consumerGroupURLPathFormat, topic), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/vnd.kafka.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 && resp.StatusCode != 409 {
		// we are checking if consumer is already present and if not present it should be created
		return errkit.New("Error creating consumer")
	}
	defer resp.Body.Close() //nolint:errcheck
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Debug().Print(string(bytes))
	return nil
}

// subscribe subscribes a consumer to topic provided in kafka cluster
func subscribe(uri string) error {
	data := SubscriptionPayload{
		Topics: []string{topic},
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", uri+fmt.Sprintf(subscriptionURLPathFormat, topic, topic), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/vnd.kafka.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return errkit.New("Error subscribing to the topic")
	}
	defer resp.Body.Close() //nolint:errcheck
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Debug().Print(string(bytes))
	return nil
}

// consumeMessage consumes the message from a topic
func consumeMessage(uri string) (int, error) {
	req, err := http.NewRequest("GET", uri+fmt.Sprintf(consumeTopicMessage, topic, topic), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.kafka.json.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		return 0, errkit.New("Error consuming records from topic")
	}
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	responseBody := string(bytes)
	var Message []Message
	err = json.Unmarshal([]byte(responseBody), &Message)
	if err != nil {
		return 0, err
	}
	if len(Message) > 0 {
		log.Debug().Print(string(bytes))
	}
	return len(Message), nil
}

func consumeTopic(ctx context.Context, namespace string) (int, error) {
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
	err = createConsumerGroup(uri)
	if err != nil {
		return 0, err
	}
	err = subscribe(uri)
	if err != nil {
		return 0, err
	}
	recordCount := 0
	ctx, cancel := context.WithTimeout(ctx, consumeTimeout)
	defer cancel()
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		for {
			recordCount, err = consumeMessage(uri)
			if err != nil {
				return false, err
			}
			if recordCount > 0 {
				return true, nil
			}
		}
	})
	if err != nil {
		return 0, err
	}
	return recordCount, err
}
