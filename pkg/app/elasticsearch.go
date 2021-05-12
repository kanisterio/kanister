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
	"encoding/json"
	"fmt"
	"time"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	esWaitTimeout = 3 * time.Minute
)

// ElasticsearchPingOutput struct gets mapped to the output of curl <es-host>:<es-port>/<index-name>/_search?pretty
// which actually returns details about all the documents of a specific ES index (index-name)
// if, due to any reason, there is change in how Elasticsearch responds to  above query, below
// struct is subject to change.
type ElasticsearchPingOutput struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
	} `json:"hits"`
}

type ElasticsearchInstance struct {
	cli              kubernetes.Interface
	namespace        string
	name             string
	indexname        string
	chart            helm.ChartInfo
	elasticsearchURL string
}

// Last tested on 7.10.1
func NewElasticsearchInstance(name string) App {
	return &ElasticsearchInstance{
		name:      name,
		namespace: "es-test",
		indexname: "testindex",
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoURL:  helm.ElasticRepoURL,
			Chart:    "elasticsearch",
			RepoName: helm.ElasticRepoName,
			Values: map[string]string{
				"antiAffinity": "soft",
				"replicas":     "1",
			},
		},
		elasticsearchURL: "localhost:9200",
	}
}

func (esi *ElasticsearchInstance) Init(ctx context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	esi.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

func (esi *ElasticsearchInstance) Install(ctx context.Context, namespace string) error {
	esi.namespace = namespace
	// Get the HELM cli
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}

	log.Print("Installing the application using helm.", field.M{"app": esi.name})
	err = cli.AddRepo(ctx, esi.chart.RepoName, esi.chart.RepoURL)
	if err != nil {
		return err
	}

	err = cli.Install(ctx, fmt.Sprintf("%s/%s", esi.chart.RepoName, esi.chart.Chart), esi.chart.Version, esi.chart.Release, esi.namespace, esi.chart.Values)
	if err != nil {
		return err
	}
	log.Print("Application was installed successfully.", field.M{"app": esi.name})
	return nil
}

func (esi *ElasticsearchInstance) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the application to be ready.", field.M{"app": esi.name})
	ctx, cancel := context.WithTimeout(ctx, esWaitTimeout)
	defer cancel()

	err := kube.WaitOnStatefulSetReady(ctx, esi.cli, esi.namespace, fmt.Sprintf("%s-master", esi.name))
	if err != nil {
		return false, err
	}

	log.Print("Application is ready.", field.M{"app": esi.name})
	return true, nil
}

func (esi *ElasticsearchInstance) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "StatefulSet",
		Name:      fmt.Sprintf("%s-master", esi.name),
		Namespace: esi.namespace,
	}
}

func (esi *ElasticsearchInstance) Uninstall(ctx context.Context) error {
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}

	log.Print("UnInstalling the application using helm.", field.M{"app": esi.name})
	err = cli.Uninstall(ctx, esi.chart.Release, esi.namespace)
	if err != nil {
		return errors.Wrapf(err, "Error uninstalling the application %s", esi.name)
	}

	return nil
}

func (esi *ElasticsearchInstance) Ping(ctx context.Context) error {
	log.Print("Pinging the application to check if its accessible.", field.M{"app": esi.name})

	pingCMD := []string{"sh", "-c", fmt.Sprintf("curl %s", esi.elasticsearchURL)}
	_, stderr, err := esi.execCommand(ctx, pingCMD)
	if err != nil {
		return errors.Wrapf(err, "Failed to ping the application. Error:%s", stderr)
	}

	log.Print("Ping to the application was successful.", field.M{"app": esi.name})
	return nil
}
func (esi *ElasticsearchInstance) Insert(ctx context.Context) error {
	addDocumentToIndexCMD := []string{"sh", "-c", fmt.Sprintf("curl -X POST %s/%s/_doc/?refresh=true -H 'Content-Type: application/json' -d'{\"appname\": \"kanister\" }'", esi.elasticsearchURL, esi.indexname)}
	_, stderr, err := esi.execCommand(ctx, addDocumentToIndexCMD)
	if err != nil {
		// even one insert failed we will have to return becasue
		// the count wont  match anyway and the test will fail
		return errors.Wrapf(err, "Error %s inserting document to an index %s.", stderr, esi.indexname)
	}

	log.Print("A document was inserted into the elastics search index.", field.M{"app": esi.name})
	return nil
}

func (esi *ElasticsearchInstance) Count(ctx context.Context) (int, error) {
	documentCountCMD := []string{"sh", "-c", fmt.Sprintf("curl %s/%s/_search?pretty", esi.elasticsearchURL, esi.indexname)}
	stdout, stderr, err := esi.execCommand(ctx, documentCountCMD)
	if err != nil {
		return 0, errors.Wrapf(err, "Error %s Counting the documents in an index.", stderr)
	}

	// convert the output to ElasticsearchPingOutput object so that we can get the document count
	pingOutput := ElasticsearchPingOutput{}
	err = json.Unmarshal([]byte(stdout), &pingOutput)
	if err != nil {
		return 0, err
	}

	log.Print("Hit count that we have in count is ", field.M{"app": esi.name, "count": pingOutput.Hits.Total.Value})
	return pingOutput.Hits.Total.Value, nil
}

func (esi *ElasticsearchInstance) Reset(ctx context.Context) error {
	log.Print("Resetting the application.", field.M{"app": esi.name})

	// delete the index and then create it, in order to reset the es application
	deleteIndexCMD := []string{"sh", "-c", fmt.Sprintf("curl -X DELETE %s/%s?pretty", esi.elasticsearchURL, esi.indexname)}
	_, stderr, err := esi.execCommand(ctx, deleteIndexCMD)
	if err != nil {
		return errors.Wrapf(err, "Error %s while deleting the index %s to reset the application.", stderr, esi.indexname)
	}

	return nil
}

// Initialize is used to initialize the database or create schema
func (esi *ElasticsearchInstance) Initialize(ctx context.Context) error {
	// create the index
	createIndexCMD := []string{"sh", "-c", fmt.Sprintf("curl -X PUT %s/%s?pretty", esi.elasticsearchURL, esi.indexname)}
	_, stderr, err := esi.execCommand(ctx, createIndexCMD)
	if err != nil {
		return errors.Wrapf(err, "Error %s: Resetting the application.", stderr)
	}
	return nil
}

func (esi *ElasticsearchInstance) execCommand(ctx context.Context, command []string) (string, string, error) {
	podname, containername, err := kube.GetPodContainerFromStatefulSet(ctx, esi.cli, esi.namespace, fmt.Sprintf("%s-master", esi.name))
	if err != nil || podname == "" {
		return "", "", errors.Wrapf(err, "Error getting the pod and container name %s.", esi.name)
	}
	return kube.Exec(esi.cli, esi.namespace, podname, containername, command, nil)
}
