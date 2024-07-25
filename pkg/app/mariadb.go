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

	"github.com/kanisterio/errkit"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	mariaWaitTimeout = 1 * time.Minute
	mariaDBSTSSuffix = "mariadb"
)

type MariaDB struct {
	cli       kubernetes.Interface
	namespace string
	name      string
	chart     helm.ChartInfo
}

func NewMariaDB(name string) App {
	return &MariaDB{
		name: name,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoURL:  helm.BitnamiRepoURL,
			Chart:    "mariadb",
			RepoName: helm.BitnamiRepoName,
			Values: map[string]string{
				"auth.rootPassword": "mysecretpassword",
				"image.pullPolicy":  "Always",
			},
		},
	}
}

func (m *MariaDB) Init(context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	m.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	return nil
}

func (m *MariaDB) Install(ctx context.Context, namespace string) error { //nolint:dupl // Not a duplicate, common code already extracted
	m.namespace = namespace
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}
	log.Print("Adding repo.", field.M{"app": m.name})
	err = cli.AddRepo(ctx, m.chart.RepoName, m.chart.RepoURL)
	if err != nil {
		return errkit.Wrap(err, "Error adding helm repo for app.", "app", m.name)
	}

	log.Print("Installing maria instance using helm.", field.M{"app": m.name})
	_, err = cli.Install(ctx, m.chart.RepoName+"/"+m.chart.Chart, m.chart.Version, m.chart.Release, m.namespace, m.chart.Values, true, false)
	if err != nil {
		return errkit.Wrap(err, "Error intalling application through helm.", "app", m.name)
	}

	return nil
}

func (m *MariaDB) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the maria instance to be ready.", field.M{"app": m.name})
	ctx, cancel := context.WithTimeout(ctx, mariaWaitTimeout)
	defer cancel()
	err := kube.WaitOnStatefulSetReady(ctx, m.cli, m.namespace, mariaDBSTSName(m.chart.Release))
	if err != nil {
		return false, err
	}
	log.Print("Application instance is ready.", field.M{"app": m.name})
	return true, nil
}

func (m *MariaDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "statefulset",
		Name:      mariaDBSTSName(m.chart.Release),
		Namespace: m.namespace,
	}
}

func (m *MariaDB) Uninstall(ctx context.Context) error {
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}
	err = cli.Uninstall(ctx, m.chart.Release, m.namespace)
	if err != nil {
		log.WithError(err).Print("Failed to uninstall app, you will have to uninstall it manually.", field.M{"app": m.name})
		return err
	}
	log.Print("Uninstalled application.", field.M{"app": m.name})
	return nil
}

func (m *MariaDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (m *MariaDB) Ping(ctx context.Context) error {
	log.Print("Pinging the maria database.", field.M{"app": m.name})

	// exec into the pod and create the test database, read password from secret
	loginMaria := []string{"sh", "-c", "mysql -u root --password=$MARIADB_ROOT_PASSWORD"}
	_, stderr, err := m.execCommand(ctx, loginMaria)
	if err != nil {
		return errkit.Wrap(err, "Error while Pinging the database", "stderr", stderr)
	}

	log.Print("Ping to the application was success.", field.M{"app": m.name})
	return nil
}

func (m *MariaDB) Insert(ctx context.Context) error {
	log.Print("Inserting some records in  maria instance.", field.M{"app": m.name})

	insertRecordCMD := []string{"sh", "-c", "mysql -u root --password=$MARIADB_ROOT_PASSWORD -e 'use testdb; INSERT INTO pets VALUES (\"Puffball\",\"Diane\",\"hamster\",\"f\",\"1999-03-30\",NULL); '"}
	_, stderr, err := m.execCommand(ctx, insertRecordCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting the data into msyql database", "stderr", stderr)
	}

	log.Print("Successfully inserted records in the application.", field.M{"app": m.name})
	return nil
}

func (m *MariaDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting the records from the maria instance.", field.M{"app": m.name})

	selectRowsCMD := []string{"sh", "-c", "mysql -u root --password=$MARIADB_ROOT_PASSWORD -e 'use testdb; select count(*) from pets; '"}
	stdout, stderr, err := m.execCommand(ctx, selectRowsCMD)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while counting the data of the database", "stderr", stderr)
	}
	// get the returned count and convert it to int, to return
	rowsReturned, err := strconv.Atoi((strings.Split(stdout, "\n")[1]))
	if err != nil {
		return 0, errkit.Wrap(err, "Error while converting row count to int.")
	}
	log.Print("Count that we received from application is.", field.M{"app": m.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (m *MariaDB) Reset(ctx context.Context) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, mariaWaitTimeout)
	defer waitCancel()
	err := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		err := m.Ping(ctx)
		return err == nil, nil
	})

	if err != nil {
		return errkit.Wrap(err, "Error waiting for application to be ready to reset it", "app", m.name)
	}

	log.Print("Resetting the maria instance.", field.M{"app": m.name})

	// delete all the data from the table
	deleteFromTableCMD := []string{"sh", "-c", "mysql -u root --password=$MARIADB_ROOT_PASSWORD -e 'DROP DATABASE IF EXISTS testdb'"}
	_, stderr, err := m.execCommand(ctx, deleteFromTableCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while dropping the maria table", "stderr", stderr)
	}

	log.Print("Reset of the application was successful.", field.M{"app": m.name})
	return nil
}

func (m *MariaDB) Initialize(ctx context.Context) error {
	// create the database and a pets table
	createTableCMD := []string{"sh", "-c", "mysql -u root --password=$MARIADB_ROOT_PASSWORD " +
		"-e 'create database testdb; use testdb; " +
		"CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), " +
		"birth DATE, death DATE);'"}
	_, stderr, err := m.execCommand(ctx, createTableCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while creating the maria table", "stderr", stderr)
	}
	return nil
}

func (m *MariaDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podname, containername, err := kube.GetPodContainerFromStatefulSet(ctx, m.cli, m.namespace, mariaDBSTSName(m.chart.Release))
	if err != nil || podname == "" {
		return "", "", errkit.Wrap(err, "Error  getting pod and containername.", "app", m.name)
	}
	return kube.Exec(ctx, m.cli, m.namespace, podname, containername, command, nil)
}

func mariaDBSTSName(release string) string {
	return fmt.Sprintf("%s-%s", release, mariaDBSTSSuffix)
}
