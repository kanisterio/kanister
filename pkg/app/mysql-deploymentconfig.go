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
	"strconv"
	"strings"
	"time"

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
	mysqlDepConfigWaitTimeout = 5 * time.Minute
	mysqlDepConfigName        = "mysql"
)

type MysqlDepConfig struct {
	cli         kubernetes.Interface
	osCli       osversioned.Interface
	name        string
	namespace   string
	params      map[string]string
	storageType storage
	osClient    openshift.OSClient
	// dbTemplateVersion will most probably match with the OCP version
	dbTemplateVersion DBTemplate
}

func NewMysqlDepConfig(name string, templateVersion DBTemplate, storageType storage, tag string) App {
	return &MysqlDepConfig{
		name: name,
		params: map[string]string{
			"MYSQL_ROOT_PASSWORD": "secretpassword",
			"MYSQL_VERSION":       tag,
		},
		storageType:       storageType,
		osClient:          openshift.NewOpenShiftClient(),
		dbTemplateVersion: templateVersion,
	}
}

func (mdep *MysqlDepConfig) Init(context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	mdep.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	mdep.osCli, err = osversioned.NewForConfig(cfg)

	return err
}

func (mdep *MysqlDepConfig) Install(ctx context.Context, namespace string) error {
	mdep.namespace = namespace

	dbTemplate := getOpenShiftDBTemplate(mysqlDepConfigName, mdep.dbTemplateVersion, mdep.storageType)

	oc := openshift.NewOpenShiftClient()
	_, err := oc.NewApp(ctx, mdep.namespace, dbTemplate, nil, mdep.params)

	return errkit.Wrap(err, "Error installing app on openshift cluster.", "app", mdep.name)
}

func (mdep *MysqlDepConfig) IsReady(ctx context.Context) (bool, error) {
	log.Info().Print("Waiting for application to be ready.", field.M{"app": mdep.name})
	ctx, cancel := context.WithTimeout(ctx, mysqlDepConfigWaitTimeout)
	defer cancel()

	err := kube.WaitOnDeploymentConfigReady(ctx, mdep.osCli, mdep.cli, mdep.namespace, mysqlDepConfigName)
	if err != nil {
		return false, err
	}

	log.Print("Application is ready", field.M{"application": mdep.name})
	return true, nil
}

func (mdep *MysqlDepConfig) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "deploymentconfig",
		Name:      mysqlDepConfigName,
		Namespace: mdep.namespace,
	}
}

func (mdep *MysqlDepConfig) Uninstall(ctx context.Context) error {
	_, err := mdep.osClient.DeleteApp(ctx, mdep.namespace, getLabelOfApp(mysqlDepConfigName, mdep.storageType))
	return err
}

func (mdep *MysqlDepConfig) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (mdep *MysqlDepConfig) Ping(ctx context.Context) error {
	log.Print("Pinging the application", field.M{"app": mdep.name})

	pingCMD := []string{"bash", "-c", "mysql -u root -e 'show databases;'"}
	_, stderr, err := mdep.execCommand(ctx, pingCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while Pinging the database", "stderr", stderr)
	}
	log.Print("Ping to the application was successful.")
	return nil
}

func (mdep *MysqlDepConfig) Insert(ctx context.Context) error {
	log.Print("Inserting some records in  mysql instance.", field.M{"app": mdep.name})

	insertRecordCMD := []string{"bash", "-c", "mysql -u root -e 'use testdb; INSERT INTO pets VALUES (\"Puffball\",\"Diane\",\"hamster\",\"f\",\"1999-03-30\",NULL); '"}
	_, stderr, err := mdep.execCommand(ctx, insertRecordCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting the data into msyql deployment config database", "stderr", stderr)
	}

	log.Print("Successfully inserted record in the application.", field.M{"app": mdep.name})
	return nil
}

func (mdep *MysqlDepConfig) Count(ctx context.Context) (int, error) {
	log.Print("Counting the records from the mysql instance.", field.M{"app": mdep.name})

	selectRowsCMD := []string{"bash", "-c", "mysql -u root -e 'use testdb; select count(*) from pets; '"}
	stdout, stderr, err := mdep.execCommand(ctx, selectRowsCMD)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while counting the data of the database", "stderr", stderr)
	}

	// get the returned count and convert it to int, to return
	rowsReturned, err := strconv.Atoi((strings.Split(stdout, "\n")[1]))
	if err != nil {
		return 0, errkit.Wrap(err, "Error while converting row count to int.")
	}

	log.Print("Count that we received from application is.", field.M{"app": mdep.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (mdep *MysqlDepConfig) Reset(ctx context.Context) error {
	log.Print("Resetting the mysql instance.", field.M{"app": "mysql"})

	// delete all the data from the table
	deleteCMD := []string{"bash", "-c", "mysql -u root -e 'DROP DATABASE IF EXISTS testdb'"}
	_, stderr, err := mdep.execCommand(ctx, deleteCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while dropping the mysql table", "stderr", stderr)
	}

	// create the database and a pets table
	createCMD := []string{"bash", "-c", "mysql -u root -e 'create database testdb; use testdb;  CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);'"}
	_, stderr, err = mdep.execCommand(ctx, createCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while creating the mysql table", "stderr", stderr)
	}

	log.Print("Reset of the application was successful.", field.M{"app": mdep.name})
	return nil
}

// Initialize is used to initialize the database or create schema
func (mdep *MysqlDepConfig) Initialize(ctx context.Context) error {
	return nil
}

func (mdep *MysqlDepConfig) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromDeploymentConfig(ctx, mdep.osCli, mdep.cli, mdep.namespace, mysqlDepConfigName)
	if err != nil {
		return "", "", err
	}
	stdout, stderr, err := kube.Exec(ctx, mdep.cli, mdep.namespace, podName, containerName, command, nil)
	if err != nil {
		return stdout, stderr, errkit.Wrap(err, "Error executing command in the pod.")
	}
	return stdout, stderr, err
}
