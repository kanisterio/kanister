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
	"time"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/openshift"
)

const (
	postgresDepConfigName          = "postgresql"
	postgreSQLDepConfigWaitTimeout = 2 * time.Minute
)

type PostgreSQLDepConfig struct {
	name           string
	cli            kubernetes.Interface
	osCli          osversioned.Interface
	namespace      string
	opeshiftClient openshift.OSClient
	dbTemplate     string
}

func NewPostgreSQLDepConfig(name string) App {
	return &PostgreSQLDepConfig{
		name:           name,
		opeshiftClient: openshift.NewOpenShiftClient(),
		dbTemplate:     getOpenShiftDBTemplate(postgresDepConfigName),
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

	_, err := pgres.opeshiftClient.NewApp(ctx, pgres.namespace, pgres.dbTemplate, nil)

	return errors.Wrap(err, "Error while installing the application.")
}

func (pgres *PostgreSQLDepConfig) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for application to be ready.", field.M{"app": pgres.name})
	ctx, cancel := context.WithTimeout(ctx, postgreSQLDepConfigWaitTimeout)
	defer cancel()

	err := kube.WaitOnDeploymentConfigReady(ctx, pgres.osCli, pgres.cli, pgres.namespace, postgresDepConfigName)
	if err != nil {
		return false, errors.Wrapf(err, "Error %s waiting for application to be ready.", pgres.name)
	}

	log.Print("Application is ready", field.M{"app": pgres.name})
	return false, nil
}

func (pgres *PostgreSQLDepConfig) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "deploymentconfig",
		Name:      postgresDepConfigName,
		Namespace: pgres.namespace,
	}
}

func (pgres *PostgreSQLDepConfig) Uninstall(context.Context) error {
	return nil
}

func (pgres *PostgreSQLDepConfig) Ping(context.Context) error {
	return nil
}

func (pgres *PostgreSQLDepConfig) Insert(ctx context.Context) error {
	return nil
}

func (pgres *PostgreSQLDepConfig) Count(context.Context) (int, error) {
	return 0, nil
}

func (pgres *PostgreSQLDepConfig) Reset(context.Context) error {
	return nil
}
