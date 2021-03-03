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
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	kafkaClusterWaitTimeout = 5 * time.Minute
)

type kubernetesClient struct {
}

// NewkubernetesClient returns kubernetes client
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
	createOperator := []string{"create", "-n", namespace, "-f", strimziYaml}
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
func (kubectl kubernetesClient) CreateConfigMap(ctx context.Context, configMapName, namespace, yamlFileRepo, kafkaConfigPath, sinkConfigPath, sourceConfigPath string) (string, error) {
	//	createConfig := []string{"create", "configmap", "s3config", fmt.Sprintf("--from-file=%s/%s --from-file=%s/%s --from-file=%s/%s --from-literal=timeinSeconds=1800", yamlFileRepo, sinkConfigPath, yamlFileRepo, sourceConfigPath, yamlFileRepo, sinkConfigPath), "-n", namespace}
	createConfig := []string{"create", "-n", namespace, "configmap", configMapName, fmt.Sprintf("--from-file=adobe-s3-sink.properties=%s/%s", yamlFileRepo, sinkConfigPath), fmt.Sprintf("--from-file=adobe-s3-source.properties=%s/%s", yamlFileRepo, sourceConfigPath), fmt.Sprintf("--from-file=adobe-kafkaConfiguration.properties=%s/%s", yamlFileRepo, kafkaConfigPath), "--from-literal=timeinSeconds=1800"}
	return helm.RunCmdWithTimeout(ctx, "kubectl", createConfig)
}
func (kubectl kubernetesClient) DeleteConfigMap(ctx context.Context, namespace, configMapName string) (string, error) {
	deleteConfig := []string{"delete", "-n", namespace, "configmap", configMapName}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteConfig)
}
func (kubectl kubernetesClient) DeleteOperator(ctx context.Context, namespace, yamlFileRepo, strimziYaml string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	deleteOperator := []string{"delete", "-n", namespace, "-f", strimziYaml}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteOperator)
}
func (kubectl kubernetesClient) DeleteKafka(ctx context.Context, namespace, yamlFileRepo, kafkaConfigPath string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	deleteKafka := []string{"delete", "-n", namespace, "-f", fmt.Sprintf("%s/%s", yamlFileRepo, kafkaConfigPath)}
	return helm.RunCmdWithTimeout(ctx, "kubectl", deleteKafka)
}

func (kubectl kubernetesClient) Ping(ctx context.Context, namespace string) (string, error) {
	// kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
	pingKafka := []string{"run", "-n", namespace, "kafka-ping", "-ti", "--rm=true", "--image=strimzi/kafka:0.20.0-kafka-2.6.0", "--restart=Never", "--", "bin/kafka-topics.sh", "--list", "--bootstrap-server=my-cluster-kafka-external-bootstrap:9094"}
	log.Print(pingKafka)
	return helm.RunCmdWithTimeout(ctx, "kubectl", pingKafka)
}
func (kubectl kubernetesClient) Insert(ctx context.Context, topic, namespace string) error {
	err := produce(ctx, topic)
	return err
}
func (kubectl kubernetesClient) Count(ctx context.Context, topic, namespace string) (int, error) {
	count, err := consume(ctx, topic)
	return count, err
}

// to produce messages to kafka
func produce(ctx context.Context, topic string) error {
	partition := 0
	forwarder, err := K8SServicePortForward(ctx, "my-cluster-kafka-external-bootstrap", "kafka", "9094")
	if err != nil {
		return err
	}

	defer forwarder.Close()
	ports, err := forwarder.GetPorts()
	if err != nil {
		log.Print("failed to get port:", err)
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
		log.Print("failed to close connection:", err)
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
	//	log.Print("Attempt to use InCluster config failed")

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
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}
	roundTripper, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return &portforward.PortForwarder{}, errors.Wrapf(err, "Failed to create RoundTripper for k8s config")
	}

	svc, err := clientset.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return &portforward.PortForwarder{}, errors.Wrapf(err, "Failed to get service for component")
	}
	for k, v := range svc.Spec.Selector {
		selector = fmt.Sprintf("%s%s=%s,", selector, k, v)
	}
	selector = strings.TrimSuffix(selector, ",")
	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return &portforward.PortForwarder{}, errors.Wrapf(err, "Failed to list pods for component")
	}
	if len(pods.Items) == 0 {
		return &portforward.PortForwarder{}, errors.Wrapf(err, "Empty pods list for component")
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", ns, pods.Items[0].Name)
	u, err := url.Parse(cfg.Host)
	if err != nil {
		return &portforward.PortForwarder{}, errors.Wrapf(err, "Failed to parse url struct from k8s config")
	}
	hostIP := fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	pre, pwe := io.Pipe()
	pro, pwo := io.Pipe()
	go func() {
		buf := bytes.NewBuffer(nil)
		if _, inErr := buf.ReadFrom(pre); inErr != nil {
			log.Print(inErr)
		}
	}()
	go func() {
		buf := bytes.NewBuffer(nil)
		if _, inErr := buf.ReadFrom(pro); inErr != nil {
			log.Print(inErr)
		}
	}()

	f, err := portforward.New(dialer, []string{fmt.Sprintf(":%s", pPort)}, ctx.Done(), readyChan, pwo, pwe)
	if err != nil {
		return &portforward.PortForwarder{}, errors.Wrapf(err, "Failed to create portforward")
	}

	go func() {
		errCh <- f.ForwardPorts()
	}()

	select {
	case <-readyChan:
		log.Print("PortForward is Ready")
	case err = <-errCh:
		return &portforward.PortForwarder{}, errors.Wrapf(err, "Failed to get Ports from forwarded")
	}

	return f, nil
}

func consume(ctx context.Context, topic string) (int, error) {
	partition := 0
	forwarder, err := K8SServicePortForward(ctx, "my-cluster-kafka-external-bootstrap", "kafka", "9094")
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
		log.Print("failed to get connection:", err)
		return 0, err
	}

	err = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		log.Print("failed to set ReadDeadline:", err)
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
	if messageEnd == false {
		if err := batch.Close(); err != nil {
			log.Print("failed to close batch:", err)
			return 0, err
		}
	}
	if err := conn.Close(); err != nil {
		log.Print("failed to close connection:", err)
		return 0, err
	}
	return count, nil
}
