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
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const pgReadyTimeout = 1 * time.Minute

type PostgresDB struct {
	name      string
	cli       kubernetes.Interface
	chart     helm.ChartInfo
	namespace string
}

func NewPostgresDB(name string) App {
	return &PostgresDB{
		name: name,
		chart: helm.ChartInfo{
			Release:  name,
			RepoName: helm.StableRepoName,
			RepoURL:  helm.StableRepoURL,
			Chart:    "postgresql",
			Version:  "7.6.0",
			Values: map[string]string{
				"image.repository":                      "kanisterio/postgresql",
				"image.tag":                             "0.22.0",
				"postgresqlPassword":                    "test@54321",
				"postgresqlExtendedConf.archiveCommand": "'envdir /bitnami/postgresql/data/env wal-e wal-push %p'",
				"postgresqlExtendedConf.archiveMode":    "true",
				"postgresqlExtendedConf.archiveTimeout": "60",
				"postgresqlExtendedConf.walLevel":       "archive",
			},
		},
	}
}

func (pdb *PostgresDB) getStatefulSetName() string {
	return fmt.Sprintf("%s-postgresql", pdb.chart.Release)
}

func (pdb *PostgresDB) Init(ctx context.Context) error {
	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	pdb.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

func (pdb *PostgresDB) Install(ctx context.Context, ns string) error {
	log.Info().Print("Installing helm chart.", field.M{"app": pdb.name, "release": pdb.chart.Release, "namespace": ns})
	pdb.namespace = ns

	// Create helm client
	cli := helm.NewCliClient(helm.V3)

	// Add helm repo and fetch charts
	if err := cli.AddRepo(ctx, pdb.chart.RepoName, pdb.chart.RepoURL); err != nil {
		return err
	}
	// Install helm chart
	return cli.Install(ctx, fmt.Sprintf("%s/%s", pdb.chart.RepoName, pdb.chart.Chart), pdb.chart.Version, pdb.chart.Release, pdb.namespace, pdb.chart.Values)
}

func (pdb *PostgresDB) IsReady(ctx context.Context) (bool, error) {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, pgReadyTimeout)
	defer cancel()

	if err := kube.WaitOnStatefulSetReady(ctx, pdb.cli, pdb.namespace, pdb.getStatefulSetName()); err != nil {
		return false, err
	}
	return true, nil
}

func (pdb *PostgresDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "statefulset",
		Name:      pdb.getStatefulSetName(),
		Namespace: pdb.namespace,
	}
}

func (pdb PostgresDB) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (pdb PostgresDB) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"postgresql": crv1alpha1.ObjectReference{
			Kind:      "secret",
			Name:      pdb.getStatefulSetName(),
			Namespace: pdb.namespace,
		},
	}
}

// Ping makes and tests DB connection
func (pdb *PostgresDB) Ping(ctx context.Context) error {
	cmd := "pg_isready -U 'postgres' -h 127.0.0.1 -p 5432"
	_, stderr, err := pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to ping postgresql DB. %s", stderr)
	}
	log.Info().Print("Connected to database.", field.M{"app": pdb.name})
	return nil
}

func (pdb PostgresDB) Insert(ctx context.Context) error {
	cmd := fmt.Sprintf("PGPASSWORD=${POSTGRES_PASSWORD} psql -d test -c \"INSERT INTO COMPANY (NAME,AGE,CREATED_AT) VALUES ('foo', 32, now());\"")
	_, stderr, err := pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create db in postgresql. %s", stderr)
	}
	log.Info().Print("Inserted a row in test db.", field.M{"app": pdb.name})
	return nil
}

func (pdb PostgresDB) Count(ctx context.Context) (int, error) {
	cmd := "PGPASSWORD=${POSTGRES_PASSWORD} psql -d test -c 'SELECT COUNT(*) FROM company;'"
	stdout, stderr, err := pdb.execCommand(ctx, []string{"sh", "-c", cmd})
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
	log.Info().Print("Counting rows in test db.", field.M{"app": pdb.name, "count": count})
	return count, nil
}

func (pdb PostgresDB) Reset(ctx context.Context) error {
	// Delete database if exists
	cmd := "PGPASSWORD=${POSTGRES_PASSWORD} psql -c 'DROP DATABASE IF EXISTS test;'"
	_, stderr, err := pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to drop db from postgresql. %s ", stderr)
	}

	// Create database
	cmd = "PGPASSWORD=${POSTGRES_PASSWORD} psql -c 'CREATE DATABASE test;'"
	_, stderr, err = pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create db in postgresql. %s ", stderr)
	}

	// Create table
	cmd = "PGPASSWORD=${POSTGRES_PASSWORD} psql -d test -c 'CREATE TABLE COMPANY(ID SERIAL PRIMARY KEY NOT NULL, NAME TEXT NOT NULL, AGE INT NOT NULL, CREATED_AT TIMESTAMP);'"
	_, stderr, err = pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create table in postgresql. %s ", stderr)
	}
	log.Info().Print("Database reset successful!", field.M{"app": pdb.name})
	return nil
}

func (pdb PostgresDB) Uninstall(ctx context.Context) error {
	log.Info().Print("Uninstalling helm chart.", field.M{"app": pdb.name, "release": pdb.chart.Release, "namespace": pdb.namespace})
	// Create helm client
	cli := helm.NewCliClient(helm.V3)

	// Uninstall helm chart
	return errors.Wrapf(cli.Uninstall(ctx, pdb.chart.Release, pdb.namespace), "Failed to uninstall %s helm release", pdb.chart.Release)
}

func (pdb PostgresDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	// Get pod and container name
	pod, container, err := getPodContainerFromStatefulSet(ctx, pdb.cli, pdb.namespace, pdb.getStatefulSetName())
	if err != nil {
		return "", "", err
	}
	return kube.Exec(pdb.cli, pdb.namespace, pod, container, command, nil)
}
