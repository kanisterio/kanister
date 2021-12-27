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

const pghaReadyTimeout = 1 * time.Minute

type PostgresDBHA struct {
	name      string
	cli       kubernetes.Interface
	chart     helm.ChartInfo
	namespace string
}

// Last tested chart version "10.12.3". Also we are using postgres version 13.4
func NewPostgresDBWithHA(name string, subPath string, storageclass string) App {
	return &PostgresDB{
		name: name,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoName: helm.BitnamiRepoName,
			RepoURL:  helm.BitnamiRepoURL,
			Chart:    "postgresql-ha",
			Values: map[string]string{
				"image.registry":            "ghcr.io",
				"image.repository":          "kanisterio/postgresql",
				"image.tag":                 "latest",
				"image.pullPolicy":          "Always",
				"postgresql.password":       "test@54321",
				"volumePermissions.enabled": "true",
				"persistence.storageClass":  storageclass,
			},
		},
	}
}

func (pdb *PostgresDBHA) getStatefulSetName() string {
	return fmt.Sprintf("%s-postgresql-ha-postgresql", pdb.chart.Release)
}

func (pdb *PostgresDBHA) getPGPoolDeploymentName() string {
	return fmt.Sprintf("%s-postgresql-ha-pgpool", pdb.chart.Release)
}

func (pdb *PostgresDBHA) Init(ctx context.Context) error {
	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	pdb.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

func (pdb *PostgresDBHA) Install(ctx context.Context, ns string) error {
	log.Info().Print("Installing helm chart.", field.M{"app": pdb.name, "release": pdb.chart.Release, "namespace": ns})
	pdb.namespace = ns

	// Create helm client
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}

	// Add helm repo and fetch charts
	if err = cli.AddRepo(ctx, pdb.chart.RepoName, pdb.chart.RepoURL); err != nil {
		return err
	}
	// Install helm chart
	return cli.Install(ctx, fmt.Sprintf("%s/%s", pdb.chart.RepoName, pdb.chart.Chart), pdb.chart.Version, pdb.chart.Release, pdb.namespace, pdb.chart.Values)
}

func (pdb *PostgresDBHA) IsReady(ctx context.Context) (bool, error) {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, pgReadyTimeout)
	defer cancel()

	if err := kube.WaitOnStatefulSetReady(ctx, pdb.cli, pdb.namespace, pdb.getStatefulSetName()); err != nil {
		return false, err
	}

	if err := kube.WaitOnDeploymentReady(ctx, pdb.cli, pdb.namespace, pdb.getPGPoolDeploymentName()); err != nil {
		return false, err
	}

	return true, nil
}

func (pdb *PostgresDBHA) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "statefulset",
		Name:      pdb.getStatefulSetName(),
		Namespace: pdb.namespace,
	}
}

func (pdb PostgresDBHA) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (pdb PostgresDBHA) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"postgresql": crv1alpha1.ObjectReference{
			Kind:      "secret",
			Name:      pdb.getStatefulSetName(),
			Namespace: pdb.namespace,
		},
		"pgpool": crv1alpha1.ObjectReference{
			Kind:      "secret",
			Name:      pdb.getPGPoolDeploymentName(),
			Namespace: pdb.namespace,
		},
	}
}

// Ping makes and tests DB connection
func (pdb *PostgresDBHA) Ping(ctx context.Context) error {
	cmd := "pg_isready -U 'postgres' -h 127.0.0.1 -p 5432"
	_, stderr, err := pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to ping postgresql DB. %s", stderr)
	}
	log.Info().Print("Connected to database.", field.M{"app": pdb.name})
	return nil
}

func (pdb PostgresDBHA) Insert(ctx context.Context) error {
	cmd := fmt.Sprintf("PGPASSWORD=${POSTGRES_PASSWORD} psql -d test -c \"INSERT INTO COMPANY (NAME,AGE,CREATED_AT) VALUES ('foo', 32, now());\"")
	_, stderr, err := pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create db in postgresql. %s", stderr)
	}
	log.Info().Print("Inserted a row in test db.", field.M{"app": pdb.name})
	return nil
}

func (pdb PostgresDBHA) Count(ctx context.Context) (int, error) {
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

func (pdb PostgresDBHA) Reset(ctx context.Context) error {
	// Delete database if exists
	cmd := "PGPASSWORD=${POSTGRES_PASSWORD} psql -c 'DROP DATABASE IF EXISTS test;'"
	_, stderr, err := pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to drop db from postgresql. %s ", stderr)
	}

	log.Info().Print("Database reset successful!", field.M{"app": pdb.name})
	return nil
}

// Initialize is used to initialize the database or create schema
func (pdb PostgresDBHA) Initialize(ctx context.Context) error {
	// Create database
	cmd := "PGPASSWORD=${POSTGRES_PASSWORD} psql -c 'CREATE DATABASE test;'"
	_, stderr, err := pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create db in postgresql. %s ", stderr)
	}

	// Create table
	cmd = "PGPASSWORD=${POSTGRES_PASSWORD} psql -d test -c 'CREATE TABLE COMPANY(ID SERIAL PRIMARY KEY NOT NULL, NAME TEXT NOT NULL, AGE INT NOT NULL, CREATED_AT TIMESTAMP);'"
	_, stderr, err = pdb.execCommand(ctx, []string{"sh", "-c", cmd})
	if err != nil {
		return errors.Wrapf(err, "Failed to create table in postgresql. %s ", stderr)
	}
	return nil
}

func (pdb PostgresDBHA) Uninstall(ctx context.Context) error {
	log.Info().Print("Uninstalling helm chart.", field.M{"app": pdb.name, "release": pdb.chart.Release, "namespace": pdb.namespace})

	// Create helm client
	cli, err := helm.NewCliClient()
	if err != nil {
		return errors.Wrap(err, "failed to create helm client")
	}

	// Uninstall helm chart
	return errors.Wrapf(cli.Uninstall(ctx, pdb.chart.Release, pdb.namespace), "Failed to uninstall %s helm release", pdb.chart.Release)
}

func (pdp *PostgresDBHA) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (pdb PostgresDBHA) execCommand(ctx context.Context, command []string) (string, string, error) {
	// Get pod and container name
	pod, container, err := kube.GetPodContainerFromStatefulSet(ctx, pdb.cli, pdb.namespace, pdb.getStatefulSetName())
	if err != nil {
		return "", "", err
	}
	return kube.Exec(pdb.cli, pdb.namespace, pod, container, command, nil)
}
