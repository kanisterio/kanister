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
	"strings"
	"time"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ElasticsearchPingOutput struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore interface{}   `json:"max_score"`
		Hits     []interface{} `json:"hits"`
	} `json:"hits"`
}

type chartInfo struct {
	helmrepoURL string
	helmchart   string
	helmAppname string
}

type ElasticsearchInstance struct {
	cli              kubernetes.Interface
	namespace        string
	indexname        string
	chart            chartInfo
	elasticsearchURL string
}

func NewElasticsearchInstance(helmrepoURL, helmChart, helmAppname string) App {
	return &ElasticsearchInstance{
		namespace: "es-test",
		indexname: "testindex",
		chart: chartInfo{
			helmrepoURL: helmrepoURL,
			helmchart:   helmChart,
			helmAppname: helmAppname,
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
	if err != nil {
		return err
	}

	return nil
}

func (esi *ElasticsearchInstance) Install(ctx context.Context, namespace string) error {
	esi.namespace = namespace

	// Get the HELM cli
	cli := helm.NewCliClient()

	log.Print("Installing the application using helm.", field.M{"app": "elasticsearch"})
	err := cli.AddRepo(ctx, esi.chart.helmchart, esi.chart.helmrepoURL)
	if err != nil {
		log.WithError(err).Print("Error while adding help repo.", field.M{"app": "elasticsearch"})
		return err
	}
	err = cli.Install(ctx, fmt.Sprintf("%s/%s", esi.chart.helmchart, esi.chart.helmAppname), "elasticsearch", esi.namespace, map[string]string{"antiAffinity": "soft", "replicas": "1"})
	if err != nil {
		log.WithError(err).Print("Error while installing the instance.", field.M{"app": "elasticsearch"})
		return err
	}

	return nil
}

func (esi *ElasticsearchInstance) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the application to be ready.", field.M{"app": "elasticsearch"})
	ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	err := kube.WaitOnStatefulSetReady(ctx, esi.cli, esi.namespace, "elasticsearch-master")
	if err != nil {
		log.WithError(err).Print("Error while waiting for statefulset to be ready.", field.M{"app": "elasticsearch"})
		return false, err
	}

	log.Print("Application is ready.", field.M{"app": "elasticsearch "})
	return true, nil
}

func (esi *ElasticsearchInstance) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "StatefulSet",
		Name:      "elasticsearch-master",
		Namespace: esi.namespace,
	}
}

func (esi *ElasticsearchInstance) Uninstall(ctx context.Context) error {

	cli := helm.NewCliClient()

	log.Print("UnInstalling the application using helm.", field.M{"app": "elasticsearch"})

	err := cli.Uninstall(ctx, "elasticsearch", esi.namespace)
	if err != nil {
		log.WithError(err).Print("Error while UnInstalling the instance.", field.M{"app": "elasticsearch"})
		return err
	}

	return nil
}

func (esi *ElasticsearchInstance) Ping(ctx context.Context) error {
	log.Print("Pinging the application to check if its accessible.", field.M{"app": "elasticsearch"})
	podname, containername, err := esi.GetPodAndContianerName()
	if err != nil {
		log.WithError(err).Print("Error getting container to ping the application.", field.M{"app": "elasticsearch"})
		return err
	}

	// since we will be EXECing into the pod where ES is running, wec an use localhost to query it
	pingCMD := []string{"sh", "-c", fmt.Sprintf("curl %s", esi.elasticsearchURL)}
	stdout, stderr, err := kube.Exec(esi.cli, esi.namespace, podname, containername, pingCMD, nil)

	format.Log(podname, containername, stdout)
	format.Log(podname, containername, stderr)
	// check the stdout then return
	if err != nil {
		log.WithError(err).Print("Error while pinginng the application.", field.M{"app": "elasticsearch"})
		return err
	}

	return nil
}
func (esi *ElasticsearchInstance) Insert(ctx context.Context, n int) error {
	podname, containername, err := esi.GetPodAndContianerName()
	if err != nil {
		log.WithError(err).Print("Error getting pod and container to insert the documents in the index.", field.M{"app": "elasticsearch"})
		return err
	}
	log.Print("Inserting document into the elastics search index.", field.M{"app": "elaticsearch"})
	addDocumentToIndexCMD := []string{"sh", "-c", fmt.Sprintf("curl -X POST %s/%s/_doc/?refresh=true -H 'Content-Type: application/json' -d'{\"appname\": \"kanister\" }'", esi.elasticsearchURL, esi.indexname)}
	for i := 0; i < n; i++ {
		stdout, stderr, err := kube.Exec(esi.cli, esi.namespace, podname, containername, addDocumentToIndexCMD, nil)
		format.Log(podname, containername, stdout)
		format.Log(podname, containername, stderr)
		if err != nil {
			log.WithError(err).Print("Error while inserting a document into index.", field.M{"app": "elasticsearch", "index": esi.indexname})
			// even and insert failed we will have to return becasue
			// the count wont  match anyway and the test will fail
			return err
		}
		log.Print("After inserting a document to the ES index the index is ", field.M{"app": "elasticsearch", "output": stdout})
	}
	return nil
}

func (esi *ElasticsearchInstance) Count(context.Context) (int, error) {
	podname, containername, err := esi.GetPodAndContianerName()
	if err != nil {
		log.WithError(err).Print("Error getting pod and container name to get the count of the documents.", field.M{"app": "elasticsearch", "index": esi.indexname})
		return 0, err
	}
	documentCountCMD := []string{"sh", "-c", fmt.Sprintf("curl %s/%s/_search?pretty", esi.elasticsearchURL, esi.indexname)}
	stdout, stderr, err := kube.Exec(esi.cli, esi.namespace, podname, containername, documentCountCMD, nil)
	format.Log(podname, containername, stdout)
	format.Log(podname, containername, stderr)
	if err != nil {
		log.WithError(err).Print("Error counting the ES documents.", field.M{"app": "elasticsearch", "index": esi.indexname})
		return 0, err
	}

	// convert the output to ElasticsearchPingOutput object so that we can get the document count
	pingOutput := ElasticsearchPingOutput{}
	json.Unmarshal([]byte(stdout), &pingOutput)

	log.Print("Hit count that we have in count is ", field.M{"app": "elasticsearch", "count": pingOutput.Hits.Total.Value})

	return pingOutput.Hits.Total.Value, nil
}

func (esi *ElasticsearchInstance) Reset(ctx context.Context) error {
	log.Print("Resetting the application.", field.M{"app": "elasticsearch"})

	// delete the index and then create it, in order to reset the es application
	podname, containername, err := esi.GetPodAndContianerName()
	if err != nil {
		log.WithError(err).Print("Error while getting pod and container to reset the application.", field.M{"app": "elasticsearch"})
		return err
	}

	deleteIndexCMD := []string{"sh", "-c", fmt.Sprintf("curl -X DELETE %s/%s?pretty", esi.elasticsearchURL, esi.indexname)}
	stdout, stderr, err := kube.Exec(esi.cli, esi.namespace, podname, containername, deleteIndexCMD, nil)
	// check the stdout
	log.Print("Output that we have after deleting the index ", field.M{"app": "elasticsearch", "output": stdout})
	format.Log(podname, containername, stdout)
	format.Log(podname, containername, stderr)
	if err != nil {
		log.WithError(err).Print("Error while deleting the index to reset the application.", field.M{"app": "elasticsearch", "index": esi.indexname})
		return err
	}

	// create the index
	createIndexCMD := []string{"sh", "-c", fmt.Sprintf("curl -X PUT %s/%s?pretty", esi.elasticsearchURL, esi.indexname)}
	stdout, stderr, err = kube.Exec(esi.cli, esi.namespace, podname, containername, createIndexCMD, nil)
	format.Log(podname, containername, stdout)
	format.Log(podname, containername, stderr)
	log.Print("Output that we have after creating the index ", field.M{"app": "elasticsearch", "output": stdout})
	if err != nil {
		log.WithError(err).Print("Error while creating the index.", field.M{"app": "elasticsearch"})
		return err
	}

	return nil
}

// GetPodAndContianerName takes namespace as input and returns the pod and container that is running in
// that namespace for deployment that was created through helm
// use the function after the PR https://github.com/kanisterio/kanister/pull/418/
func (esi *ElasticsearchInstance) GetPodAndContianerName() (string, string, error) {
	statefulset, err := esi.cli.AppsV1().StatefulSets(esi.namespace).Get("elasticsearch-master", metav1.GetOptions{})
	if err != nil {
		log.WithError(err).Print("Error getting statefulset to ping.", field.M{"app": "elasticsearch"})
		return "", "", err
	}
	statefulsetSelector := statefulset.Spec.Selector.MatchLabels
	var podLabelSelector []string
	for key, value := range statefulsetSelector {
		podLabelSelector = append(podLabelSelector, key+"="+value)
	}

	pods, err := esi.cli.CoreV1().Pods(esi.namespace).List(metav1.ListOptions{
		LabelSelector: strings.Join(podLabelSelector, ","),
	})
	if err != nil {
		return "", "", fmt.Errorf("Error while getting pods of the deployment %s", err.Error())
	}

	return pods.Items[0].Name, pods.Items[0].Spec.Containers[0].Name, nil
}
