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
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kanisterio/errkit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	cbReadyTimeout = 5 * time.Minute
	numPings       = 1
)

// Regex to extract result from cb query response
var countResp = regexp.MustCompile(`(?m){"\$1":([\d]+)},`)

type CouchbaseDB struct {
	name      string
	namespace string
	username  string
	password  string
	cli       kubernetes.Interface
	chart     helm.ChartInfo
}

// NewCouchbaseDB initialises an instance of Couchbase DB
// Last tested woking version "2.3.0"
func NewCouchbaseDB(name string) App {
	return &CouchbaseDB{
		name: name,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoName: helm.CouchbaseRepoName,
			RepoURL:  helm.CouchbaseRepoURL,
			Chart:    "couchbase-operator",
			Values: map[string]string{
				"cluster.servers.default.size":        "1",
				"cluster.servers.default.services[0]": "data",
				"cluster.servers.default.services[1]": "query",
				"cluster.servers.default.services[2]": "index",
			},
		},
	}
}

func (cb *CouchbaseDB) Init(ctx context.Context) error {
	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	cb.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

func (cb *CouchbaseDB) Install(ctx context.Context, ns string) error { //nolint:dupl // Not a duplicate, common code already extracted
	log.Info().Print("Installing couchbase operator and cluster helm chart.", field.M{"app": cb.name})
	cb.namespace = ns

	// Create helm client
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}

	// Add helm repo and fetch charts
	if err = cli.AddRepo(ctx, cb.chart.RepoName, cb.chart.RepoURL); err != nil {
		return errkit.Wrap(err, "Failed to install helm repo.", "app", cb.name, "repo", cb.chart.RepoName)
	}

	// Install cb operator, admission controller and cluster
	_, err = cli.Install(ctx, fmt.Sprintf("%s/%s", cb.chart.RepoName, cb.chart.Chart), cb.chart.Version, cb.chart.Release, cb.namespace, cb.chart.Values, true, false)
	return errkit.Wrap(err, "Failed to install helm chart.", "app", cb.name, "chart", cb.chart.Chart, "release", cb.chart.Release)
}

func (cb *CouchbaseDB) IsReady(ctx context.Context) (bool, error) {
	log.Info().Print("Waiting for couchbase cluster to be ready.", field.M{"app": cb.name})
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, cbReadyTimeout)
	defer cancel()

	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		_, err := cb.getRunningCBPod()
		return err == nil, nil
	})
	if err != nil {
		return false, err
	}

	// Read cluster creds from Secret
	secret, err := cb.cli.CoreV1().Secrets(cb.namespace).Get(ctx, fmt.Sprintf("auth-%s-couchbase-cluster", cb.chart.Release), metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if _, exist := secret.Data["username"]; exist {
		cb.username = string(secret.Data["username"])
	}
	if _, exist := secret.Data["password"]; exist {
		cb.password = string(secret.Data["password"])
	}
	return err == nil, err
}

func (cb *CouchbaseDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		APIVersion: "v2",
		Group:      "couchbase.com",
		Name:       cb.chart.Release,
		Namespace:  cb.namespace,
		Resource:   "couchbaseclusters",
	}
}

// Ping makes and tests DB connection
func (cb *CouchbaseDB) Ping(ctx context.Context) error {
	log.Info().Print("Pinging database.", field.M{"app": cb.name})
	cmd := fmt.Sprintf("cbc-ping -u %s -P %s -c %d", cb.username, cb.password, numPings)
	_, stderr, err := cb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to ping couchbase DB", "stderr", stderr)
	}
	log.Info().Print("Connected to database.", field.M{"app": cb.name})
	return nil
}

func (cb CouchbaseDB) Insert(ctx context.Context) error {
	log.Info().Print("Inserting data into default backet.", field.M{"app": cb.name})
	c, err := cb.Count(ctx)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("cbc-create -u %s -P %s %s -V '{\"name\":\"test\", \"age\": 25}'", cb.username, cb.password, uuid.New().String())
	_, stderr, err := cb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to add document in couchbase default bucket", "stderr", stderr)
	}

	// We'll wait till count correct result
	err = cb.waitForCount(ctx, c+1)
	if err != nil {
		return err
	}

	log.Info().Print("Inserted a document in default couchbase bucket.", field.M{"app": cb.name})
	return nil
}

func (cb CouchbaseDB) Count(ctx context.Context) (int, error) {
	cmd := fmt.Sprintf("cbc-n1ql -u %s -P %s 'select count(*) from default'", cb.username, cb.password)
	stdout, stderr, err := cb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return 0, errkit.Wrap(err, "Failed to count db entries in couchbase", "stderr", stderr)
	}

	// Parse output
	// Output format:
	// ---> Encoded query: {"statement":"select count(*) from default"}
	//
	// {"$1":4},
	// ---> Query response finished
	// {
	// "requestID": "6c235ff9-6e88-46f5-8531-18a42d753841",
	// "signature": {"$1":"number"},
	// "results": [
	// ],
	// "status": "success",
	// "metrics": {"elapsedTime": "34.274958ms","executionTime": "33.329635ms","resultCount": 1,"resultSize": 8}
	// }
	matched := countResp.FindAllStringSubmatch(stdout, -1)
	if len(matched) != 1 || len(matched[0]) != 2 {
		return 0, nil
	}

	count, err := strconv.Atoi(matched[0][1])
	if err != nil {
		return 0, errkit.Wrap(err, "Failed to count db entries in couchbase", "stderr", stderr)
	}
	log.Info().Print("Counting rows in test db.", field.M{"app": cb.name, "count": count})
	return count, nil
}

func (cb CouchbaseDB) Reset(ctx context.Context) error {
	// Flush default bucket in couchbase cluster
	log.Info().Print("Delete all documents from default bucket", field.M{"app": cb.name})

	// Create index
	cmd := fmt.Sprintf("cbc-n1ql -u %s -P %s 'create primary index on default'", cb.username, cb.password)
	_, stderr, err := cb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to create index on default", "stderr", stderr, "app", cb.name)
	}

	// Delete all documents
	cmd = fmt.Sprintf("cbc-n1ql -u %s -P %s 'delete from default'", cb.username, cb.password)
	_, stderr, err = cb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to delete documents from default bucket", "stderr", stderr, "app", cb.name)
	}

	// We'll wait till count returns zero
	return cb.waitForCount(ctx, 0)
}

// Initialize is used to initialize the database or create schema
func (cb CouchbaseDB) Initialize(ctx context.Context) error {
	return nil
}

func (cb CouchbaseDB) Uninstall(ctx context.Context) error {
	// Create helm client
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}

	// Uninstall couchbase-operator helm chart
	log.Info().Print("Uninstalling helm charts.", field.M{"app": cb.name, "release": cb.chart.Release, "namespace": cb.namespace})
	err = cli.Uninstall(ctx, cb.chart.Release, cb.namespace)
	return errkit.Wrap(err, "Failed to uninstall helm release", "release", cb.chart.Release)
}

func (cb CouchbaseDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (cb CouchbaseDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	// Get pod and container name
	podName, err := cb.getRunningCBPod()
	if err != nil {
		return "", "", err
	}

	container, err := kube.PodContainers(ctx, cb.cli, cb.namespace, podName)
	if err != nil || len(container) == 0 {
		return "", "", err
	}
	return kube.Exec(ctx, cb.cli, cb.namespace, podName, container[0].Name, command, nil)
}

// getRunningCBPod name of running couchbase cluster pod if its in ready state
func (cb CouchbaseDB) getRunningCBPod() (string, error) {
	podName := fmt.Sprintf("%s-0000", cb.chart.Release)
	pod, err := cb.cli.CoreV1().Pods(cb.namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if len(pod.Status.ContainerStatuses) == 0 {
		return "", errkit.New("Could not find ready pod.", "name", podName, "namespace", cb.namespace)
	}
	if !pod.Status.ContainerStatuses[0].Ready {
		return "", errkit.New("Could not find ready pod.", podName, cb.namespace)
	}

	return pod.GetName(), nil
}

// Couchbase cluster takes some time to replicate data
// We'll wait till count query returns expected result
func (cb CouchbaseDB) waitForCount(ctx context.Context, result int) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		count, err := cb.Count(ctx)
		return count == result, err
	})
	return errkit.Wrap(err, "Timed out while waiting for Couchbase cluster to be in sync")
}
