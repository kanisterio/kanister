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

	"github.com/pkg/errors"
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

type MongoDB struct {
	cli       kubernetes.Interface
	namespace string
	username  string
	name      string
	chart     helm.ChartInfo
}

func NewMongoDB(name string) App {
	return &MongoDB{
		username: "root",
		name:     name,
		chart: helm.ChartInfo{
			Release:  name,
			RepoURL:  helm.StableRepoURL,
			Chart:    "mongodb",
			RepoName: helm.StableRepoName,
			Version:  "7.4.6",
			Values: map[string]string{
				"replicaSet.enabled":  "true",
				"image.repository":    "kanisterio/mongodb",
				"image.tag":           "0.22.0",
				"mongodbRootPassword": "secretpassword",
			},
		},
	}
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
	cli := helm.NewCliClient(helm.V3)
	log.Print("Adding repo for the application.", field.M{"app": mongo.name})

	err := cli.AddRepo(ctx, mongo.chart.RepoName, mongo.chart.RepoURL)
	if err != nil {
		return err
	}

	log.Print("Installing application using helm.", field.M{"app": mongo.name})
	err = cli.Install(ctx, fmt.Sprintf("%s/%s", mongo.chart.RepoName, mongo.chart.Chart), mongo.chart.Version, mongo.name, mongo.namespace, mongo.chart.Values)
	if err != nil {
		return err
	}
	log.Print("Application was installed successfully.", field.M{"app": mongo.name})
	return nil
}

func (mongo *MongoDB) IsReady(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, mongoWaitTimeout)
	defer cancel()

	statefSets := []string{"mongodb-primary", "mongodb-secondary", "mongodb-arbiter"}
	for _, resource := range statefSets {
		err := kube.WaitOnStatefulSetReady(ctx, mongo.cli, mongo.namespace, fmt.Sprintf("%s-%s", mongo.name, resource))
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
		Name:      fmt.Sprintf("%s-mongodb-primary", mongo.name),
		Namespace: mongo.namespace,
	}
}

func (mongo *MongoDB) Uninstall(ctx context.Context) error {
	cli := helm.NewCliClient(helm.V3)
	log.Print("Uninstalling application.", field.M{"app": mongo.name})
	err := cli.Uninstall(ctx, mongo.name, mongo.namespace)
	return errors.Wrapf(err, "Error while uninstalling the application.")
}

func (mongo *MongoDB) Ping(ctx context.Context) error {
	log.Print("Pinging the application.", field.M{"app": mongo.name})
	pingCMD := []string{"sh", "-c", fmt.Sprintf("mongo admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"rs.slaveOk(); db\"", mongo.username)}
	_, stderr, err := mongo.execCommand(ctx, pingCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while pinging the mongodb application %s", stderr)
	}

	// even after ping is successful, it takes some time for primary pod to becomd the master
	// we will have to wait for that so that the write subsequent write requests wont fail.
	isMasterCMD := []string{"sh", "-c", fmt.Sprintf(" mongo admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"db.isMaster()\"", mongo.username)}
	isMasterStdout, stderr, err := mongo.execCommand(ctx, isMasterCMD)
	if err != nil {
		return errors.Wrapf(err, "Error %s checking if the pod is master.", stderr)
	}

	// extract the ismaster field from the output and make sure
	// this ismaster field is true
	re := regexp.MustCompile("\"ismaster\" : ([a-z]*),")
	match := re.FindStringSubmatch(isMasterStdout)
	if len(match) < 2 {
		return errors.Wrap(errors.New("ismaster not in the output while checking if the monogdb is node."), "Error while checking if the mongodb node is master:")
	}

	isMaster, err := strconv.ParseBool(match[1])
	if err != nil {
		return errors.Wrapf(err, "Error converting ismaster value.")
	}
	if !isMaster {
		return errors.New("Error while getting ismaster value from output.")
	}

	log.Print("Ping was successful to application.", field.M{"app": mongo.name})
	return nil
}

func (mongo *MongoDB) Insert(ctx context.Context) error {
	log.Print("Inserting documents into collection.", field.M{"app": mongo.name})
	insertCMD := []string{"sh", "-c", fmt.Sprintf("mongo admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"db.restaurants.insert({'name' : 'Tom', 'cuisine' : 'Hawaiian', 'id' : '8675309'})\"", mongo.username)}
	_, stderr, err := mongo.execCommand(ctx, insertCMD)
	if err != nil {
		return errors.Wrapf(err, "Error %s while inserting data data into mongodb collection.", stderr)
	}

	log.Print("Insertion of documents into collection was successful.", field.M{"app": mongo.name})
	return nil
}
func (mongo *MongoDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting documents of collection.", field.M{"app": mongo.name})
	countCMD := []string{"sh", "-c", fmt.Sprintf("mongo admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"rs.slaveOk(); db.restaurants.count()\"", mongo.username)}
	stdout, stderr, err := mongo.execCommand(ctx, countCMD)
	if err != nil {
		return 0, errors.Wrapf(err, "Error %s while counting the data in mongodb collection.", stderr)
	}

	noOfRecords, err := strconv.Atoi(stdout)
	if err != nil {
		return 0, err
	}

	log.Print("Count that we are returning from count method is.", field.M{"app": "mongodb", "count": noOfRecords})
	return noOfRecords, nil
}
func (mongo *MongoDB) Reset(ctx context.Context) error {
	log.Print("Resetting the application.", field.M{"app": mongo.name})
	// delete all the entries from the restaurants dummy colleciton
	// we are not deleting the database becasue we are dealing with admin database here
	// and deletion admin database is prohibited
	deleteDBCMD := []string{"sh", "-c", fmt.Sprintf("mongo admin --authenticationDatabase admin -u %s -p $MONGODB_ROOT_PASSWORD --quiet --eval \"rs.slaveOk(); db.restaurants.drop()\"", mongo.username)}
	stdout, stderr, err := mongo.execCommand(ctx, deleteDBCMD)
	return errors.Wrapf(err, "Error %s, resetting the mongodb application. stdout is %s", stderr, stdout)
}

func (mongo *MongoDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := getPodContainerFromStatefulSet(ctx, mongo.cli, mongo.namespace, fmt.Sprintf("%s-mongodb-primary", mongo.name))
	if err != nil || podName == "" {
		return "", "", err
	}
	return kube.Exec(mongo.cli, mongo.namespace, podName, containerName, command, nil)
}
