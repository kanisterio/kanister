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

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	mysqlWaitTimeout = 1 * time.Minute
)

type MysqlDB struct {
	cli       kubernetes.Interface
	namespace string
	name      string
	chart     helm.ChartInfo
}

func NewMysqlDB(name string) App {
	return &MysqlDB{
		name: name,
		chart: helm.ChartInfo{
			Release:  name,
			RepoURL:  helm.StableRepoURL,
			Chart:    "mysql",
			RepoName: helm.StableRepoName,
			Version:  "1.4.0",
			Values: map[string]string{
				"mysqlRootPassword":   "mysecretpassword",
				"persistence.enabled": "false",
				"imagePullPolicy":     "Always",
			},
		},
	}
}

func (mdb *MysqlDB) Init(ctx context.Context) error {

	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	mdb.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	return nil
}

func (mdb *MysqlDB) Install(ctx context.Context, namespace string) error {

	mdb.namespace = namespace

	cli := helm.NewCliClient(helm.V3)
	log.Print("Adding repo.", field.M{"app": mdb.name})
	err := cli.AddRepo(ctx, mdb.chart.RepoName, mdb.chart.RepoURL)
	if err != nil {
		return errors.Wrapf(err, "Error helm repo for app %s.", mdb.name)
	}

	log.Print("Installing mysql instance using helm.", field.M{"app": mdb.name})
	err = cli.Install(ctx, mdb.chart.RepoName+"/"+mdb.chart.Chart, mdb.chart.Version, mdb.chart.Release, mdb.namespace, mdb.chart.Values)
	if err != nil {
		return errors.Wrapf(err, "Error intalling application %s through helm.", mdb.name)
	}

	return nil
}

func (mdb *MysqlDB) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the mysql instance to be ready.", field.M{"app": mdb.name})
	ctx, cancel := context.WithTimeout(ctx, mysqlWaitTimeout)
	defer cancel()
	err := kube.WaitOnDeploymentReady(ctx, mdb.cli, mdb.namespace, mdb.name)
	if err != nil {
		return false, err
	}

	log.Print("Application instance is ready.", field.M{"app": mdb.name})
	return true, nil

}

func (mdb *MysqlDB) Object() crv1alpha1.ObjectReference {

	return crv1alpha1.ObjectReference{
		Kind:      "deployment",
		Name:      mdb.name,
		Namespace: mdb.namespace,
	}
}

func (mdb *MysqlDB) Uninstall(ctx context.Context) error {
	cli := helm.NewCliClient(helm.V3)

	err := cli.Uninstall(ctx, mdb.name, mdb.namespace)
	if err != nil {
		log.WithError(err).Print("Failed to uninstall app, you will have to uninstall it manually.", field.M{"app": mdb.name})
		return err
	}
	log.Print("Uninstalled application.", field.M{"app": mdb.name})

	return nil
}

func (mdb *MysqlDB) Ping(ctx context.Context) error {
	log.Print("Pinging the mysql database.", field.M{"app": mdb.name})

	// exec into the pod and create the test database, read password from secret
	loginMysql := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD"}
	_, stderr, err := mdb.execCommand(ctx, loginMysql)
	if err != nil {
		return errors.Wrapf(err, "Error while Pinging the database %s", stderr)
	}

	log.Print("Ping to the application was success.", field.M{"app": mdb.name})
	return nil
}

func (mdb *MysqlDB) Insert(ctx context.Context) error {
	log.Print("Inserting some records in  mysql instance.", field.M{"app": mdb.name})

	insertRecordCMD := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD -e 'use testdb; INSERT INTO pets VALUES (\"Puffball\",\"Diane\",\"hamster\",\"f\",\"1999-03-30\",NULL); '"}
	_, stderr, err := mdb.execCommand(ctx, insertRecordCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while inserting the data into msyql database: %s", stderr)
	}

	log.Print("Successfully inserted recored in the application.", field.M{"app": mdb.name})
	return nil
}

func (mdb *MysqlDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting the records from the mysql isntance.", field.M{"app": mdb.name})

	selectRowsCMD := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD -e 'use testdb; select count(*) from pets; '"}
	stdout, stderr, err := mdb.execCommand(ctx, selectRowsCMD)
	if err != nil {
		return 0, errors.Wrapf(err, "Error while counting the data of the database: %s", stderr)
	}
	// get the returned cound and convert it to int, to return
	rowsReturned, err := strconv.Atoi((strings.Split(stdout, "\n")[1]))
	if err != nil {
		return 0, errors.Wrapf(err, "Error while converting row count to int.")
	}
	log.Print("Count that we received from application is.", field.M{"app": mdb.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (mdb *MysqlDB) Reset(ctx context.Context) error {
	log.Print("Resetting the mysql instance.", field.M{"app": "mysql"})

	// delete all the data from the table
	deleteFromTableCMD := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD -e 'DROP DATABASE IF EXISTS testdb'"}
	_, stderr, err := mdb.execCommand(ctx, deleteFromTableCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while dropping the mysql table: %s", stderr)
	}

	// create the database and a pets dummy table
	createTableCMD := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD -e 'create database testdb; use testdb;  CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);'"}
	_, stderr, err = mdb.execCommand(ctx, createTableCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while creating the mysql table: %s", stderr)
	}

	log.Print("Rest of the application was successful.", field.M{"app": mdb.name})
	return nil
}

func (mdb *MysqlDB) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (mdb *MysqlDB) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"mysql": crv1alpha1.ObjectReference{
			Kind:      "Secret",
			Name:      mdb.name,
			Namespace: mdb.namespace,
		},
	}
}

func (mdb *MysqlDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podname, containername, err := GetPodContainerFromDeployment(ctx, mdb.cli, mdb.namespace, mdb.name)
	if err != nil || podname == "" {
		return "", "", errors.Wrapf(err, "Error  getting pod and containername %s.", mdb.name)
	}
	return kube.Exec(mdb.cli, mdb.namespace, podname, containername, command, nil)
}
