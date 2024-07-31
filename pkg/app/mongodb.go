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
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kanisterio/errkit"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	mongoWaitTimeout = 5 * time.Minute
)

// IsMasterOutput struct gets mapped to the output of the mongo command that checks if node is master or not.
type IsMasterOutput struct {
	Ismaster bool `json:"ismaster"`
}

var _ HelmApp = &MongoDB{}

type MongoDB struct {
	cli       kubernetes.Interface
	namespace string
	username  string
	name      string
	chart     helm.ChartInfo
}

// NewMongoDB initialises an instance of Mongo DB
// Last tested working version "9.0.0"
func NewMongoDB(name string) HelmApp {
	return &MongoDB{
		username: "root",
		name:     name,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoURL:  helm.BitnamiRepoURL,
			RepoName: helm.BitnamiRepoName,
			Chart:    "mongodb",
			Version:  "14.11.1",
			Values: map[string]string{
				"architecture":     "replicaset",
				"image.pullPolicy": "Always",
			},
		},
	}
}

func (mongo *MongoDB) Chart() *helm.ChartInfo {
	return &mongo.chart
}

func (mongo *MongoDB) SetChart(chart helm.ChartInfo) {
	mongo.chart = chart
}

func (mongo *MongoDB) Init(ctx context.Context) error {
	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	mongo.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

func (mongo *MongoDB) Install(ctx context.Context, namespace string) error {
	mongo.namespace = namespace
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}

	log.Print("Adding repo for the application.", field.M{"app": mongo.name})
	err = cli.AddRepo(ctx, mongo.chart.RepoName, mongo.chart.RepoURL)
	if err != nil {
		return err
	}

	log.Print("Installing application using helm.", field.M{"app": mongo.name})
	_, err = cli.Install(ctx, fmt.Sprintf("%s/%s", mongo.chart.RepoName, mongo.chart.Chart), mongo.chart.Version, mongo.chart.Release, mongo.namespace, mongo.chart.Values, true, false)
	if err != nil {
		return err
	}
	log.Print("Application was installed successfully.", field.M{"app": mongo.name})
	return nil
}

func (mongo *MongoDB) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for application to be ready", field.M{"app": mongo.name})
	ctx, cancel := context.WithTimeout(ctx, mongoWaitTimeout)
	defer cancel()

	statefSets := []string{fmt.Sprintf("%s-mongodb", mongo.chart.Release), fmt.Sprintf("%s-mongodb-arbiter", mongo.chart.Release)}
	for _, resource := range statefSets {
		err := kube.WaitOnStatefulSetReady(ctx, mongo.cli, mongo.namespace, resource)
		if err != nil {
			return false, err
		}
	}

	log.Print("Application is ready.", field.M{"app": mongo.name})
	return true, nil
}

func (mongo *MongoDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "StatefulSet",
		Name:      fmt.Sprintf("%s-mongodb", mongo.chart.Release),
		Namespace: mongo.namespace,
	}
}

func (mongo *MongoDB) Uninstall(ctx context.Context) error {
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}
	log.Print("Uninstalling application.", field.M{"app": mongo.name})
	err = cli.Uninstall(ctx, mongo.chart.Release, mongo.namespace)
	return errkit.Wrap(err, "Error while uninstalling the application.")
}

func (mongo *MongoDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (mongo *MongoDB) Ping(ctx context.Context) error {
	log.Print("Pinging the application.", field.M{"app": mongo.name})
	pingCMD := []string{"sh", "-c", fmt.Sprintf("mongosh admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"db\"", mongo.username)}
	_, stderr, err := mongo.execCommand(ctx, pingCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while pinging the mongodb application", "stderr", stderr)
	}

	// even after ping is successful, it takes some time for primary pod to becomd the master
	// we will have to wait for that so that the write subsequent write requests wont fail.
	isMasterCMD := []string{"sh", "-c", fmt.Sprintf("mongosh admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"JSON.stringify(db.isMaster())\"", mongo.username)}
	stdout, stderr, err := mongo.execCommand(ctx, isMasterCMD)
	if err != nil {
		return errkit.Wrap(err, "Error checking if the pod is master.", "stderr", stderr)
	}

	// convert the mongo's output to go struct so that we can check if the pod has become master or not.
	op := IsMasterOutput{}
	err = json.Unmarshal([]byte(stdout), &op)
	if err != nil {
		return errkit.Wrap(err, "Error unmarshalling the ismaster ouptut.")
	}
	if !op.Ismaster {
		return errkit.New("the pod is not master yet")
	}

	log.Print("Ping was successful to application.", field.M{"app": mongo.name})
	return nil
}

func (mongo *MongoDB) Insert(ctx context.Context) error {
	log.Print("Inserting documents into collection.", field.M{"app": mongo.name})
	insertCMD := []string{"sh", "-c", fmt.Sprintf("mongosh admin --authenticationDatabase admin -u %s -p "+
		"$MONGODB_ROOT_PASSWORD --quiet --eval \"db.restaurants.insertOne({'_id': '%s','name' : 'Tom', "+
		"'cuisine' : 'Hawaiian', 'id' : '8675309'})\"", mongo.username, uuid.New())}
	_, stderr, err := mongo.execCommand(ctx, insertCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting data data into mongodb collection.", "stderr", stderr)
	}

	log.Print("Insertion of documents into collection was successful.", field.M{"app": mongo.name})
	return nil
}

func (mongo *MongoDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting documents of collection.", field.M{"app": mongo.name})
	countCMD := []string{"sh", "-c", fmt.Sprintf("mongosh admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"db.restaurants.countDocuments()\"", mongo.username)}
	stdout, stderr, err := mongo.execCommand(ctx, countCMD)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while counting the data in mongodb collection.", "stderr", stderr)
	}

	count, err := strconv.Atoi(stdout)
	if err != nil {
		return 0, err
	}

	log.Print("Count that we are returning from count method is.", field.M{"app": "mongodb", "count": count})
	return count, nil
}
func (mongo *MongoDB) Reset(ctx context.Context) error {
	log.Print("Resetting the application.", field.M{"app": mongo.name})
	// delete all the entries from the restaurants collection
	// we are not deleting the database because we are dealing with admin database here
	// and deletion admin database is prohibited
	deleteDBCMD := []string{"sh", "-c", fmt.Sprintf("mongosh admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"db.restaurants.drop()\"", mongo.username)}
	stdout, stderr, err := mongo.execCommand(ctx, deleteDBCMD)
	return errkit.Wrap(err, "Error resetting the mongodb application.", "stdout", stdout, "stderr", stderr)
}

// Initialize is used to initialize the database or create schema
func (mongo *MongoDB) Initialize(ctx context.Context) error {
	return nil
}

func (mongo *MongoDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromStatefulSet(ctx, mongo.cli, mongo.namespace, fmt.Sprintf("%s-mongodb", mongo.chart.Release))
	if err != nil || podName == "" {
		return "", "", err
	}
	return kube.Exec(ctx, mongo.cli, mongo.namespace, podName, containerName, command, nil)
}
