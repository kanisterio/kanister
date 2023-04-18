// Copyright 2023 The Kanister Authors.
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

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

type KopiaRepositoryServerMysqlDB struct {
	cli       kubernetes.Interface
	namespace string
	name      string
	chart     helm.ChartInfo
}

var _ HelmApp = &KopiaRepositoryServerMysqlDB{}

// NewKopiaRepositoryServerMysqlDB Last tested working version "6.14.11"
func NewKopiaRepositoryServerMysqlDB(name string) HelmApp {
	return &KopiaRepositoryServerMysqlDB{
		name: name,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoURL:  helm.BitnamiRepoURL,
			Chart:    "mysql",
			RepoName: helm.BitnamiRepoName,
			Values: map[string]string{
				"auth.rootPassword": "mysecretpassword",
				"image.pullPolicy":  "Always",
				"image.tag":         "v69",
				"image.repository":  "r4rajat/mysql",
			},
		},
	}
}

func (mdb *KopiaRepositoryServerMysqlDB) Chart() *helm.ChartInfo {
	return &mdb.chart
}

func (mdb *KopiaRepositoryServerMysqlDB) SetChart(chart helm.ChartInfo) {
	mdb.chart = chart
}

func (mdb *KopiaRepositoryServerMysqlDB) Init(ctx context.Context) error {
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

func (mdb *KopiaRepositoryServerMysqlDB) Install(ctx context.Context, namespace string) error {
	mdb.namespace = namespace
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}
	log.Print("Adding repo.", field.M{"app": mdb.name})
	err = cli.AddRepo(ctx, mdb.chart.RepoName, mdb.chart.RepoURL)
	if err != nil {
		return errors.Wrapf(err, "Error adding helm repo for app %s.", mdb.name)
	}

	log.Print("Installing mysql instance using helm.", field.M{"app": mdb.name})
	err = cli.Install(ctx, mdb.chart.RepoName+"/"+mdb.chart.Chart, mdb.chart.Version, mdb.chart.Release, mdb.namespace, mdb.chart.Values, true)
	if err != nil {
		return errors.Wrapf(err, "Error intalling application %s through helm.", mdb.name)
	}

	return nil
}

func (mdb *KopiaRepositoryServerMysqlDB) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the mysql instance to be ready.", field.M{"app": mdb.name})
	ctx, cancel := context.WithTimeout(ctx, mysqlWaitTimeout)
	defer cancel()
	err := kube.WaitOnStatefulSetReady(ctx, mdb.cli, mdb.namespace, mdb.chart.Release)
	if err != nil {
		return false, err
	}
	log.Print("Application instance is ready.", field.M{"app": mdb.name})
	return true, nil
}

func (mdb *KopiaRepositoryServerMysqlDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "statefulset",
		Name:      mdb.chart.Release,
		Namespace: mdb.namespace,
	}
}

func (mdb *KopiaRepositoryServerMysqlDB) Uninstall(ctx context.Context) error {
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}
	err = cli.Uninstall(ctx, mdb.chart.Release, mdb.namespace)
	if err != nil {
		log.WithError(err).Print("Failed to uninstall app, you will have to uninstall it manually.", field.M{"app": mdb.name})
		return err
	}
	log.Print("Uninstalled application.", field.M{"app": mdb.name})

	return nil
}

func (mdb *KopiaRepositoryServerMysqlDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (mdb *KopiaRepositoryServerMysqlDB) Ping(ctx context.Context) error {
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

func (mdb *KopiaRepositoryServerMysqlDB) Insert(ctx context.Context) error {
	log.Print("Inserting some records in  mysql instance.", field.M{"app": mdb.name})

	insertRecordCMD := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD -e 'use testdb; INSERT INTO pets VALUES (\"Puffball\",\"Diane\",\"hamster\",\"f\",\"1999-03-30\",NULL); '"}
	_, stderr, err := mdb.execCommand(ctx, insertRecordCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while inserting the data into msyql database: %s", stderr)
	}

	log.Print("Successfully inserted records in the application.", field.M{"app": mdb.name})
	return nil
}

func (mdb *KopiaRepositoryServerMysqlDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting the records from the mysql instance.", field.M{"app": mdb.name})

	selectRowsCMD := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD -e 'use testdb; select count(*) from pets; '"}
	stdout, stderr, err := mdb.execCommand(ctx, selectRowsCMD)
	if err != nil {
		return 0, errors.Wrapf(err, "Error while counting the data of the database: %s", stderr)
	}
	// get the returned count and convert it to int, to return
	rowsReturned, err := strconv.Atoi(strings.Split(stdout, "\n")[1])
	if err != nil {
		return 0, errors.Wrapf(err, "Error while converting row count to int.")
	}
	log.Print("Count that we received from application is.", field.M{"app": mdb.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (mdb *KopiaRepositoryServerMysqlDB) Reset(ctx context.Context) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, mysqlWaitTimeout)
	defer waitCancel()
	err := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		err := mdb.Ping(ctx)
		return err == nil, nil
	})

	if err != nil {
		return errors.Wrapf(err, "Error waiting for application %s to be ready to reset it", mdb.name)
	}

	log.Print("Resetting the mysql instance.", field.M{"app": "mysql"})

	// delete all the data from the table
	deleteFromTableCMD := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD -e 'DROP DATABASE IF EXISTS testdb'"}
	_, stderr, err := mdb.execCommand(ctx, deleteFromTableCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while dropping the mysql table: %s", stderr)
	}

	log.Print("Reset of the application was successful.", field.M{"app": mdb.name})
	return nil
}

// Initialize is used to initialize the database or create schema
func (mdb *KopiaRepositoryServerMysqlDB) Initialize(ctx context.Context) error {
	// create the database and a pets table
	createTableCMD := []string{"sh", "-c", "mysql -u root --password=$MYSQL_ROOT_PASSWORD -e 'create database testdb; use testdb;  CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);'"}
	_, stderr, err := mdb.execCommand(ctx, createTableCMD)
	if err != nil {
		return errors.Wrapf(err, "Error while creating the mysql table: %s", stderr)
	}
	return nil
}

func (mdb *KopiaRepositoryServerMysqlDB) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (mdb *KopiaRepositoryServerMysqlDB) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"kopia-repository-server-mysql": {
			Kind:      "Secret",
			Name:      mdb.chart.Release,
			Namespace: mdb.namespace,
		},
	}
}

func (mdb *KopiaRepositoryServerMysqlDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podname, containername, err := kube.GetPodContainerFromStatefulSet(ctx, mdb.cli, mdb.namespace, mdb.chart.Release)
	if err != nil || podname == "" {
		return "", "", errors.Wrapf(err, "Error  getting pod and containername %s.", mdb.name)
	}
	return kube.Exec(mdb.cli, mdb.namespace, podname, containername, command, nil)
}
