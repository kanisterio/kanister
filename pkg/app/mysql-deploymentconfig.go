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
	"strings"
	"time"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/openshift"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	mysqlDepConfigWaitTimeout = 2 * time.Minute
	mysqlToolPodName          = "mysql-tools-pod"
	mysqlToolContainerName    = "mysql-tools-container"
	mysqlToolImage            = "kanisterio/mysql-sidecar:0.26.0"
	mysqlDepConfigName        = "mysql"
)

type MysqlDepConfig struct {
	cli        kubernetes.Interface
	osCli      osversioned.Interface
	name       string
	namespace  string
	dbTemplate string
	envVar     map[string]string
}

func NewMysqlDepConfig(name string) App {
	return &MysqlDepConfig{
		name:       name,
		dbTemplate: getOpenShiftDBTemplate(mysqlDepConfigName),
		envVar: map[string]string{
			"MYSQL_ROOT_PASSWORD": "secretpassword",
		},
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

	oc := openshift.NewOpenShiftClient()
	_, err := oc.NewApp(ctx, mdep.namespace, mdep.dbTemplate, mdep.envVar)
	if err != nil {
		return errors.Wrapf(err, "Error installing app %s on openshift cluster.", mdep.name)
	}

	// we are creating a tool pod will be executing the commands from this pod
	// because we are not able to login to deployment config mysql instance using the env var
	// MYSQL_ROOT_PASSWORD that gets set
	err = mdep.createMySQLToolsPod(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error creating mysql tools pod")
	}

	return mdep.createMySQLSecret(ctx)
}

// createMySQLSecret creates a secret that will be used to login to running MYSQL instance
func (mdep *MysqlDepConfig) createMySQLSecret(ctx context.Context) error {
	mysqlSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      mdep.name,
			Namespace: mdep.namespace,
		},
		Data: map[string][]byte{
			"mysql-root-password": []byte(mdep.envVar["MYSQL_ROOT_PASSWORD"]),
		},
	}

	_, err := mdep.cli.CoreV1().Secrets(mdep.namespace).Create(mysqlSecret)

	return errors.Wrapf(err, "Error creating secret for mysqldepconf app.")
}

// createMySQLToolsPod creates a pod that will be use to run command into mysql instance
// we had to do this because, the secret to login to mysql instance doesnt get created
// when we deploy mysql using deploymentConfig on openshift cluster.
func (mdep *MysqlDepConfig) createMySQLToolsPod(ctx context.Context) error {
	mysqlToolPod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: mysqlToolPodName,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:      mysqlToolContainerName,
					Image:     mysqlToolImage,
					Resources: v1.ResourceRequirements{},
					Env: []v1.EnvVar{
						v1.EnvVar{
							Name: "MYSQL_ROOT_PASSWORD",
							ValueFrom: &v1.EnvVarSource{
								SecretKeyRef: &v1.SecretKeySelector{
									LocalObjectReference: v1.LocalObjectReference{Name: mdep.name},
									Key:                  "mysql-root-password",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := mdep.cli.CoreV1().Pods(mdep.namespace).Create(mysqlToolPod)

	return errors.Wrapf(err, "Error creating mysql tools pod")
}

func (mdep *MysqlDepConfig) IsReady(ctx context.Context) (bool, error) {
	log.Info().Print("Waiting for application to be ready.", field.M{"app": mdep.name})
	ctx, cancel := context.WithTimeout(ctx, mysqlDepConfigWaitTimeout)
	defer cancel()

	err := kube.WaitOnDeploymentConfigReady(ctx, mdep.osCli, mdep.cli, mdep.namespace, mysqlDepConfigName)
	if err != nil {
		return false, err
	}

	log.Print("Waiting for the MySQL tools pod to be ready.", field.M{"app": mdep.name})
	// wait for the tools pod to be ready
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		running, err := kube.IsPodRunning(mdep.cli, mysqlToolPodName, mdep.namespace)
		return running, err
	})

	if err != nil {
		return false, errors.Wrapf(err, "Error while waiting for mysql tools pod to get ready.")
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
	// delete MySQL tools pod
	err := mdep.cli.CoreV1().Pods(mdep.namespace).Delete(mysqlToolPodName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf("Error deleting pod %s from namespace %s while uninstalling application %s.", mysqlToolPodName, mdep.namespace, mdep.name))
	}
	// delete secret
	err = mdep.cli.CoreV1().Secrets(mdep.namespace).Delete(mdep.name, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf("Error deleting secret %s from namespace %s while uninstalling appliation %s.", mdep.name, mdep.namespace, mdep.name))
	}
	// deleting deployment config that is running the mysql instance
	return mdep.osCli.AppsV1().DeploymentConfigs(mdep.namespace).Delete(mysqlDepConfigName, &metav1.DeleteOptions{})
}

func (mdep *MysqlDepConfig) Ping(ctx context.Context) error {
	log.Print("Pinging the application", field.M{"app": mdep.name})

	pingCMD := []string{"sh", "-c", fmt.Sprintf("mysql -u root --password=$MYSQL_ROOT_PASSWORD -h %s -e 'show databases;'", mysqlDepConfigName)}
	_, stderr, err := mdep.execCommand(ctx, pingCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while Pinging the database %s, %s", stderr, err)
	}
	log.Print("Ping to the application was successful.")
	return nil
}

func (mdep *MysqlDepConfig) Insert(ctx context.Context) error {
	log.Print("Inserting some records in  mysql instance.", field.M{"app": mdep.name})

	insertRecordCMD := []string{"sh", "-c", fmt.Sprintf("mysql -u root --password=$MYSQL_ROOT_PASSWORD -h %s -e 'use testdb; INSERT INTO pets VALUES (\"Puffball\",\"Diane\",\"hamster\",\"f\",\"1999-03-30\",NULL); '", mysqlDepConfigName)}
	_, stderr, err := mdep.execCommand(ctx, insertRecordCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while inserting the data into msyql deployment config database: %s", stderr)
	}

	log.Print("Successfully inserted record in the application.", field.M{"app": mdep.name})
	return nil
}

func (mdep *MysqlDepConfig) Count(ctx context.Context) (int, error) {
	log.Print("Counting the records from the mysql instance.", field.M{"app": mdep.name})

	selectRowsCMD := []string{"sh", "-c", fmt.Sprintf("mysql -u root --password=$MYSQL_ROOT_PASSWORD -h %s -e 'use testdb; select count(*) from pets; '", mysqlDepConfigName)}
	stdout, stderr, err := mdep.execCommand(ctx, selectRowsCMD)
	if err != nil {
		return 0, errors.Wrapf(err, "Error while counting the data of the database: %s", stderr)
	}

	// get the returned count and convert it to int, to return
	rowsReturned, err := strconv.Atoi((strings.Split(stdout, "\n")[1]))
	if err != nil {
		return 0, errors.Wrapf(err, "Error while converting row count to int.")
	}

	log.Print("Count that we received from application is.", field.M{"app": mdep.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (mdep *MysqlDepConfig) Reset(ctx context.Context) error {
	log.Print("Resetting the mysql instance.", field.M{"app": "mysql"})

	// delete all the data from the table
	deleteCMD := []string{"sh", "-c", fmt.Sprintf("mysql -u root --password=$MYSQL_ROOT_PASSWORD -h %s -e 'DROP DATABASE IF EXISTS testdb'", mysqlDepConfigName)}
	_, stderr, err := mdep.execCommand(ctx, deleteCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while dropping the mysql table: %s", stderr)
	}

	// create the database and a pets dummy table
	createCMD := []string{"sh", "-c", fmt.Sprintf("mysql -u root --password=$MYSQL_ROOT_PASSWORD -h %s -e 'create database testdb; use testdb;  CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);'", mysqlDepConfigName)}
	_, stderr, err = mdep.execCommand(ctx, createCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while creating the mysql table: %s", stderr)
	}

	log.Print("Reset of the application was successful.", field.M{"app": mdep.name})
	return nil
}

func (mdep *MysqlDepConfig) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (mdep *MysqlDepConfig) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"mysql": crv1alpha1.ObjectReference{
			Kind:      "Secret",
			Name:      mdep.name,
			Namespace: mdep.namespace,
		},
	}
}

func (mdep *MysqlDepConfig) execCommand(ctx context.Context, command []string) (string, string, error) {
	stdout, stderr, err := kube.Exec(mdep.cli, mdep.namespace, mysqlToolPodName, mysqlToolContainerName, command, nil)
	if err != nil {
		return stdout, stderr, errors.Wrapf(err, "Error executing command in the pod.")
	}
	return stdout, stderr, err
}
<<<<<<< HEAD
=======

// getDepConfName returns the name of the deployment config that is running the mysql instance
// it gets generated on the openshift image that we have provided thats why we are getting the
// name from app image
// func (mdep *MysqlDepConfig) getDepConfName(ctx context.Context) string {
// 	return strings.Split(mdep.osAppImage, "/")[1]
// }
>>>>>>> Add another way to install databases on openshift cluster
