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
	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/openshift"
)

const (
	postgresDepConfigName          = "postgresql"
	postgreSQLDepConfigWaitTimeout = 5 * time.Minute
)

type PostgreSQLDepConfig struct {
	name           string
	cli            kubernetes.Interface
	osCli          osversioned.Interface
	namespace      string
	opeshiftClient openshift.OSClient
	envVar         map[string]string
	params         map[string]string
	storageType    storage
	// dbTemplateVersion will most probably match with the OCP version
	dbTemplateVersion DBTemplate
}

func NewPostgreSQLDepConfig(name string, templateVersion DBTemplate, storageType storage) App {
	return &PostgreSQLDepConfig{
		name:           name,
		opeshiftClient: openshift.NewOpenShiftClient(),
		envVar: map[string]string{
			"POSTGRESQL_ADMIN_PASSWORD": "secretpassword",
		},
		params: map[string]string{
			"POSTGRESQL_VERSION":  "13-el8",
			"POSTGRESQL_DATABASE": "postgres",
		},
		storageType:       storageType,
		dbTemplateVersion: templateVersion,
	}
}

func (pgres *PostgreSQLDepConfig) Init(ctx context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	pgres.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	pgres.osCli, err = osversioned.NewForConfig(cfg)

	return err
}

func (pgres *PostgreSQLDepConfig) Install(ctx context.Context, namespace string) error {
	pgres.namespace = namespace

	dbTemplate := getOpenShiftDBTemplate(postgresDepConfigName, pgres.dbTemplateVersion, pgres.storageType)

	_, err := pgres.opeshiftClient.NewApp(ctx, pgres.namespace, dbTemplate, pgres.envVar, pgres.params)
	if err != nil {
		return errkit.Wrap(err, "Error installing application on openshift cluster", "app", pgres.name)
	}
	// The secret that get created after installation doesnt have the creds that are mentioned in the
	// POSTGRESQL_ADMIN_PASSWORD above, we are creating another secret that will have this detail

	return pgres.createPostgreSQLSecret(ctx)
}

func (pgres *PostgreSQLDepConfig) createPostgreSQLSecret(ctx context.Context) error {
	postgreSQLSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", postgresDepConfigName, pgres.namespace),
			Namespace: pgres.namespace,
		},
		Data: map[string][]byte{
			"postgresql_admin_password": []byte(pgres.envVar["POSTGRESQL_ADMIN_PASSWORD"]),
		},
	}

	_, err := pgres.cli.CoreV1().Secrets(pgres.namespace).Create(ctx, postgreSQLSecret, metav1.CreateOptions{})

	return errkit.Wrap(err, "Error creating secret for mysqldepconf app.")
}

func (pgres *PostgreSQLDepConfig) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for application to be ready.", field.M{"app": pgres.name})
	ctx, cancel := context.WithTimeout(ctx, postgreSQLDepConfigWaitTimeout)
	defer cancel()

	err := kube.WaitOnDeploymentConfigReady(ctx, pgres.osCli, pgres.cli, pgres.namespace, postgresDepConfigName)
	if err != nil {
		return false, errkit.Wrap(err, "Error waiting for application to be ready.", "app", pgres.name)
	}

	log.Print("Application is ready", field.M{"app": pgres.name})
	return true, nil
}

func (pgres *PostgreSQLDepConfig) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "deploymentconfig",
		Name:      postgresDepConfigName,
		Namespace: pgres.namespace,
	}
}

func (pgres *PostgreSQLDepConfig) Uninstall(ctx context.Context) error {
	_, err := pgres.opeshiftClient.DeleteApp(ctx, pgres.namespace, getLabelOfApp(postgresDepConfigName, pgres.storageType))
	return err
}

func (pgres *PostgreSQLDepConfig) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (pgres *PostgreSQLDepConfig) Ping(ctx context.Context) error {
	cmd := "pg_isready -U 'postgres' -h 127.0.0.1 -p 5432"
	_, stderr, err := pgres.execCommand(ctx, []string{"bash", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to ping postgresql deployment config DB", "stderr", stderr)
	}
	log.Info().Print("Connected to database.", field.M{"app": pgres.name})
	return nil
}

func (pgres *PostgreSQLDepConfig) Insert(ctx context.Context) error {
	cmd := "psql -d test -c \"INSERT INTO COMPANY (NAME,AGE,CREATED_AT) VALUES ('foo', 32, now());\""
	_, stderr, err := pgres.execCommand(ctx, []string{"bash", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to create db in postgresql deployment config", "stderr", stderr)
	}
	log.Info().Print("Inserted a row in test db.", field.M{"app": pgres.name})
	return nil
}

func (pgres *PostgreSQLDepConfig) Count(ctx context.Context) (int, error) {
	cmd := "psql -d test -c 'SELECT COUNT(*) FROM company;'"
	stdout, stderr, err := pgres.execCommand(ctx, []string{"bash", "-c", cmd})
	if err != nil {
		return 0, errkit.Wrap(err, "Failed to count db entries in postgresql deployment config", "stderr", stderr)
	}

	out := strings.Fields(stdout)
	if len(out) < 4 {
		return 0, errkit.New("unknown response for count query")
	}
	count, err := strconv.Atoi(out[2])
	if err != nil {
		return 0, errkit.Wrap(err, "Failed to count db entries in postgresql deployment config", "stderr", stderr)
	}
	log.Info().Print("Counting rows in test db.", field.M{"app": pgres.name, "count": count})
	return count, nil
}

func (pgres *PostgreSQLDepConfig) Reset(ctx context.Context) error {
	cmd := "psql -c 'DROP DATABASE IF EXISTS test;'"
	_, stderr, err := pgres.execCommand(ctx, []string{"bash", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to drop db from postgresql deployment config", "stderr", stderr)
	}

	log.Info().Print("Database reset successful!", field.M{"app": pgres.name})
	return nil
}

// Initialize is used to initialize the database or create schema
func (pgres *PostgreSQLDepConfig) Initialize(ctx context.Context) error {
	// Create database
	cmd := "psql -c 'CREATE DATABASE test;'"
	_, stderr, err := pgres.execCommand(ctx, []string{"bash", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to create db in postgresql deployment config", "stderr", stderr)
	}

	// Create table
	cmd = "psql -d test -c 'CREATE TABLE COMPANY(ID SERIAL PRIMARY KEY NOT NULL, NAME TEXT NOT NULL, AGE INT NOT NULL, CREATED_AT TIMESTAMP);'"
	_, stderr, err = pgres.execCommand(ctx, []string{"bash", "-c", cmd})
	if err != nil {
		return errkit.Wrap(err, "Failed to create table in postgresql deployment config", "stderr", stderr)
	}
	return nil
}

func (pgres *PostgreSQLDepConfig) execCommand(ctx context.Context, command []string) (string, string, error) {
	// Get pod and container name
	pod, container, err := kube.GetPodContainerFromDeploymentConfig(ctx, pgres.osCli, pgres.cli, pgres.namespace, postgresDepConfigName)
	if err != nil {
		return "", "", err
	}
	return kube.Exec(ctx, pgres.cli, pgres.namespace, pod, container, command, nil)
}
