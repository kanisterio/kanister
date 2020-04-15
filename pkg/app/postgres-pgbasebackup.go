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

	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"k8s.io/client-go/kubernetes"
)

type PostgresSQLDB struct {
	name      string
	cli       kubernetes.Interface
	chart     helm.ChartInfo
	namespace string
}

const (
	postgresUser = "postgres"
	waitToCount  = 2 * time.Minute
	// this conf, allow us to run pg_basebackup and psql form anywhere witout password
	// TODO: make this work with md5 instead of trust
	pgHbaConf = `host all all 0.0.0.0/0 trust
					host all postgres 0.0.0.0/0 trust
					local all postgres trust
					host replication postgres 0.0.0.0/0 trust
					`
)

func NewPostgresSQLDB(name string) App {
	return &PostgresSQLDB{
		name: name,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			Chart:    "postgresql",
			RepoName: helm.StableRepoName,
			RepoURL:  helm.StableRepoURL,
			Version:  "8.6.4",
			Values: map[string]string{
				"image.repository":     "kanisterio/postgresql",
				"image.tag":            "0.28.0",
				"pgHbaConfiguration":   pgHbaConf,
				"postgresqlPassword":   "secretpassword",
				"replication.password": "secretreplpassword",
			},
		},
	}
}

func (pg *PostgresSQLDB) getStatefulSetName() string {
	return fmt.Sprintf("%s-%s", pg.chart.Release, "postgresql")
}

func (pg *PostgresSQLDB) Init(context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	pg.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

func (pg *PostgresSQLDB) Install(ctx context.Context, namespace string) error {
	log.Print("Installing the application ", field.M{"app": pg.name})

	pg.namespace = namespace

	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrapf(err, "failed to created helm client")
	}

	if err = cli.AddRepo(ctx, pg.chart.RepoName, pg.chart.RepoURL); err != nil {
		return err
	}

	return cli.Install(ctx, fmt.Sprintf("%s/%s", pg.chart.RepoName, pg.chart.Chart), pg.chart.Version, pg.chart.Release, pg.namespace, pg.chart.Values)
}

func (pg *PostgresSQLDB) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the application to be ready", field.M{"app": pg.name})
	ctx, cancel := context.WithTimeout(ctx, pgReadyTimeout)
	defer cancel()

	if err := kube.WaitOnStatefulSetReady(ctx, pg.cli, pg.namespace, pg.getStatefulSetName()); err != nil {
		return false, err
	}

	log.Print("The application is ready", field.M{"app": pg.name})
	return true, nil
}

func (pg *PostgresSQLDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "statefulset",
		Name:      pg.getStatefulSetName(),
		Namespace: pg.namespace,
	}
}

func (pg *PostgresSQLDB) Uninstall(ctx context.Context) error {
	log.Info().Print("Uninstalling helm chart.", field.M{"app": pg.name, "release": pg.chart.Release, "namespace": pg.namespace})

	// Create helm client
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}

	// Uninstall helm chart
	return errors.Wrapf(cli.Uninstall(ctx, pg.chart.Release, pg.namespace), "Failed to uninstall %s helm release", pg.chart.Release)
}

func (pg *PostgresSQLDB) Ping(ctx context.Context) error {
	cmd := "pg_isready -U 'postgres' -h 127.0.0.1 -p 5432"
	_, stderr, err := pg.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to ping postgresql DB. %s", stderr)
	}
	log.Info().Print("Connected to database.", field.M{"app": pg.name})
	return nil
}

func (pg *PostgresSQLDB) Insert(ctx context.Context) error {
	cmd := fmt.Sprintf("PGPASSWORD=${POSTGRES_PASSWORD} psql -d test -c \"INSERT INTO COMPANY (NAME,AGE,CREATED_AT) VALUES ('foo', 32, now());\" -U %s", postgresUser)
	_, stderr, err := pg.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create db in postgresql. %s", stderr)
	}
	log.Info().Print("Inserted a row in test db.", field.M{"app": pg.name})
	return nil
}

func (pg *PostgresSQLDB) Count(ctx context.Context) (int, error) {
	// When we restore the backup PostgreSQL pod gets restarted wait for that pod
	// to be running again.
	// Unfortunately we can not poll to wait for pod to be ready.

	time.Sleep(waitToCount)

	cmd := fmt.Sprintf("PGPASSWORD=${POSTGRES_PASSWORD} psql -d test -c 'SELECT COUNT(*) FROM company;' -U %s", postgresUser)
	stdout, stderr, err := pg.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to count db entries in postgresql. %s ", stderr)
	}

	out := strings.Fields(stdout)
	if len(out) < 4 {
		return 0, fmt.Errorf("Unknown response for count query")
	}
	count, err := strconv.Atoi(out[2])
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to count db entries in postgresql. %s ", stderr)
	}
	log.Info().Print("Counting rows in test db.", field.M{"app": pg.name, "count": count})
	return count, nil
}

func (pg *PostgresSQLDB) Reset(ctx context.Context) error {
	// Delete database if exists
	cmd := fmt.Sprintf("PGPASSWORD=${POSTGRES_PASSWORD} psql -c 'DROP DATABASE IF EXISTS test;' -U %s", postgresUser)
	_, stderr, err := pg.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to drop db from postgresql. %s ", stderr)
	}

	// Create database
	cmd = fmt.Sprintf("PGPASSWORD=${POSTGRES_PASSWORD} psql -c 'CREATE DATABASE test;' -U %s", postgresUser)
	_, stderr, err = pg.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create db in postgresql. %s ", stderr)
	}

	// Create table
	cmd = fmt.Sprintf("PGPASSWORD=${POSTGRES_PASSWORD} psql -d test -c 'CREATE TABLE COMPANY(ID SERIAL PRIMARY KEY NOT NULL, NAME TEXT NOT NULL, AGE INT NOT NULL, CREATED_AT TIMESTAMP);' -U %s", postgresUser)
	_, stderr, err = pg.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create table in postgresql. %s ", stderr)
	}
	log.Info().Print("Database reset successful!", field.M{"app": pg.name})
	return nil
}

func (pg *PostgresSQLDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	// Get pod and container name
	pod, container, err := kube.GetPodContainerFromStatefulSet(ctx, pg.cli, pg.namespace, pg.getStatefulSetName())
	if err != nil {
		return "", "", err
	}
	return kube.Exec(pg.cli, pg.namespace, pod, container, command, nil)
}
