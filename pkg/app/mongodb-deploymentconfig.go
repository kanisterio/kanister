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
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kanisterio/errkit"
	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/openshift"
)

const (
	mongoDepConfigName        = "mongodb"
	mongoDepConfigWaitTimeout = 5 * time.Minute
)

type MongoDBDepConfig struct {
	cli         kubernetes.Interface
	osCli       osversioned.Interface
	name        string
	namespace   string
	user        string
	osClient    openshift.OSClient
	params      map[string]string
	storageType storage
	// dbTemplateVersion will most probably match with the OCP version
	dbTemplateVersion DBTemplate
}

func NewMongoDBDepConfig(name string, templateVersion DBTemplate, storageType storage) App {
	return &MongoDBDepConfig{
		name: name,
		user: "admin",
		params: map[string]string{
			"MONGODB_ADMIN_PASSWORD": "secretpassword",
		},
		osClient:          openshift.NewOpenShiftClient(),
		storageType:       storageType,
		dbTemplateVersion: templateVersion,
	}
}

func (mongo *MongoDBDepConfig) Init(context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	mongo.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	mongo.osCli, err = osversioned.NewForConfig(cfg)

	return err
}

func (mongo *MongoDBDepConfig) Install(ctx context.Context, namespace string) error {
	mongo.namespace = namespace

	dbTemplate := getOpenShiftDBTemplate(mongoDepConfigName, mongo.dbTemplateVersion, mongo.storageType)

	_, err := mongo.osClient.NewApp(ctx, mongo.namespace, dbTemplate, nil, mongo.params)

	return errkit.Wrap(err, "Error installing app on openshift cluster.", "app", mongo.name)
}

func (mongo *MongoDBDepConfig) IsReady(ctx context.Context) (bool, error) {
	log.Info().Print("Waiting for application to be ready.", field.M{"app": mongo.name})
	ctx, cancel := context.WithTimeout(ctx, mongoDepConfigWaitTimeout)
	defer cancel()

	err := kube.WaitOnDeploymentConfigReady(ctx, mongo.osCli, mongo.cli, mongo.namespace, mongoDepConfigName)
	if err != nil {
		return false, err
	}

	log.Print("Application is ready", field.M{"application": mongo.name})
	return true, nil
}

func (mongo *MongoDBDepConfig) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "deploymentconfig",
		Name:      mongoDepConfigName,
		Namespace: mongo.namespace,
	}
}

func (mongo *MongoDBDepConfig) Uninstall(ctx context.Context) error {
	_, err := mongo.osClient.DeleteApp(ctx, mongo.namespace, getLabelOfApp(mongoDepConfigName, mongo.storageType))
	return err
}

func (mongo *MongoDBDepConfig) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (mongo *MongoDBDepConfig) Ping(ctx context.Context) error {
	log.Print("Pinging the application", field.M{"app": mongo.name})

	pingCMD := []string{"bash", "-c", fmt.Sprintf("mongo admin --authenticationDatabase admin -u %s -p $MONGODB_ADMIN_PASSWORD --quiet --eval \"rs.slaveOk(); db\"", mongo.user)}
	_, stderr, err := mongo.execCommand(ctx, pingCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while Pinging the database", "stderr", stderr)
	}
	log.Print("Ping to the application was successful.")
	return nil
}

func (mongo *MongoDBDepConfig) Insert(ctx context.Context) error {
	log.Print("Inserting documents into collection.", field.M{"app": mongo.name})
	insertCMD := []string{"bash", "-c", fmt.Sprintf("mongo admin --authenticationDatabase admin -u %s -p "+
		"$MONGODB_ADMIN_PASSWORD --quiet --eval \"db.restaurants.insert({'_id': '%s','name' : 'Tom', "+
		"'cuisine' : 'Hawaiian', 'id' : '8675309'})\"", mongo.user, uuid.New())}
	_, stderr, err := mongo.execCommand(ctx, insertCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting data data into mongodb collection.", "stderr", stderr)
	}

	log.Print("Insertion of documents into collection was successful.", field.M{"app": mongo.name})
	return nil
}

func (mongo *MongoDBDepConfig) Count(ctx context.Context) (int, error) {
	log.Print("Counting documents of collection.", field.M{"app": mongo.name})
	countCMD := []string{"bash", "-c", fmt.Sprintf("mongo admin --authenticationDatabase admin -u %s -p $MONGODB_ADMIN_PASSWORD --quiet --eval \"rs.slaveOk(); db.restaurants.count()\"", mongo.user)}
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

func (mongo *MongoDBDepConfig) Reset(ctx context.Context) error {
	log.Print("Resetting the application.", field.M{"app": mongo.name})
	// delete all the entries from the restaurants collection
	// we are not deleting the database because we are dealing with admin database here
	// and deletion admin database is prohibited
	deleteDBCMD := []string{"bash", "-c", fmt.Sprintf("mongo admin --authenticationDatabase admin -u %s -p $MONGODB_ADMIN_PASSWORD --quiet --eval \"db.restaurants.drop()\"", mongo.user)}
	stdout, stderr, err := mongo.execCommand(ctx, deleteDBCMD)
	return errkit.Wrap(err, "Error resetting the mongodb application.", "stdout", stdout, "stderr", stderr)
}

// Initialize is used to initialize the database or create schema
func (mongo *MongoDBDepConfig) Initialize(ctx context.Context) error {
	return nil
}

func (mongo *MongoDBDepConfig) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromDeploymentConfig(ctx, mongo.osCli, mongo.cli, mongo.namespace, mongoDepConfigName)
	if err != nil {
		return "", "", err
	}
	stdout, stderr, err := kube.Exec(ctx, mongo.cli, mongo.namespace, podName, containerName, command, nil)
	log.Print("Executing the command in pod and container", field.M{"pod": podName, "container": containerName, "cmd": command})

	return stdout, stderr, errkit.Wrap(err, "Error executing command in the pod")
}
