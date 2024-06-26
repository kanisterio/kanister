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
)

const (
	// cassandra timeout for waiting it to be ready.
	casWaitTimeout = 10 * time.Minute
	// cql (cassandra query language) timesout in 10 seconds by default
	// this is to support custom timeout so that queries will not fail.
	cqlTimeout = "300"
)

// CassandraInstance defines structure a cassandra databse application
type CassandraInstance struct {
	cli       kubernetes.Interface
	namespace string
	name      string
	chart     helm.ChartInfo
}

// NewCassandraInstance returns new cassandra application
func NewCassandraInstance(name string) App {
	return &CassandraInstance{
		name: name,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoURL:  helm.BitnamiRepoURL,
			Chart:    "cassandra",
			RepoName: helm.BitnamiRepoName,
			Values: map[string]string{
				"image.registry":   "ghcr.io",
				"image.repository": "kanisterio/cassandra",
				"image.tag":        "v9.99.9-dev",
				"image.pullPolicy": "Always",
				"replicaCount":     "1",
				"resourcesPreset":  "xlarge",
			},
		},
	}
}

// Init initializes the database application
func (cas *CassandraInstance) Init(context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	cas.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

// Install is used to install the initialized application
func (cas *CassandraInstance) Install(ctx context.Context, namespace string) error {
	cas.namespace = namespace

	log.Print("Installing application.", field.M{"app": cas.name})
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}
	err = cli.AddRepo(ctx, cas.chart.RepoName, cas.chart.RepoURL)
	if err != nil {
		return err
	}
	_, err = cli.Install(ctx, fmt.Sprintf("%s/%s", cas.chart.RepoName, cas.chart.Chart), cas.chart.Version, cas.chart.Release, cas.namespace, cas.chart.Values, true, false)
	if err != nil {
		return err
	}
	log.Print("Application was installed successfully.", field.M{"app": cas.name})
	return nil
}

// IsReady waits for the application to be ready
func (cas *CassandraInstance) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for application to be ready.", field.M{"app": cas.name})
	ctx, cancel := context.WithTimeout(ctx, casWaitTimeout)
	defer cancel()

	err := kube.WaitOnStatefulSetReady(ctx, cas.cli, cas.namespace, cas.chart.Release)
	if err != nil {
		return false, err
	}

	log.Print("Application is ready.", field.M{"app": cas.name})
	return true, nil
}

// Object returns the kubernetes resource that is being referred by db application installation
func (cas *CassandraInstance) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "StatefulSet",
		Name:      cas.chart.Release,
		Namespace: cas.namespace,
	}
}

// Uninstall us used to remove the database application
func (cas *CassandraInstance) Uninstall(ctx context.Context) error {
	log.Print("Uninstalling application.", field.M{"app": cas.name})
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}
	err = cli.Uninstall(ctx, cas.chart.Release, cas.namespace)
	if err != nil {
		return errkit.Wrap(err, "Error uninstalling cassandra app.")
	}
	log.Print("Application was uninstalled successfully.", field.M{"app": cas.name})
	return nil
}

func (cas *CassandraInstance) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

// Ping is used to ping the application to check the database connectivity
func (cas *CassandraInstance) Ping(ctx context.Context) error {
	log.Print("Pinging the application.", field.M{"app": cas.name})

	pingCMD := []string{"sh", "-c", "cqlsh -u cassandra -p $CASSANDRA_PASSWORD -e 'describe cluster'"}
	_, stderr, err := cas.execCommand(ctx, pingCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while pinging the database.", "stderr", stderr)
	}
	log.Print("Ping to the application was successful.", field.M{"app": cas.name})
	return nil
}

// Insert is used to insert the  records into the database
func (cas *CassandraInstance) Insert(ctx context.Context) error {
	log.Print("Inserting records into the database.", field.M{"app": cas.name})
	insertCMD := []string{"sh", "-c", fmt.Sprintf("cqlsh -u cassandra -p $CASSANDRA_PASSWORD -e \"insert into "+
		"restaurants.guests (id, firstname, lastname, birthday)  values (uuid(), 'Tom', 'Singh', "+
		"'2015-02-18');\" --request-timeout=%s", cqlTimeout)}
	_, stderr, err := cas.execCommand(ctx, insertCMD)
	if err != nil {
		return errkit.Wrap(err, "Error inserting records into the database.", "stderr", stderr)
	}
	return nil
}

// Count will return the number of records, there are inside the database's table
func (cas *CassandraInstance) Count(ctx context.Context) (int, error) {
	countCMD := []string{"sh", "-c", "cqlsh -u cassandra -p $CASSANDRA_PASSWORD -e \"select count(*) from restaurants.guests;\" "}
	stdout, stderr, err := cas.execCommand(ctx, countCMD)
	if err != nil {
		return 0, errkit.Wrap(err, "Error counting the number of records in the database.", "stderr", stderr)
	}
	// parse stdout to get the number of rows in the table
	// the count output from cqlsh is in below format
	// count
	// -------
	// 	3
	// (1 rows)

	count, err := strconv.Atoi(strings.Trim(strings.Split(stdout, "\n")[2], " "))
	if err != nil {
		return 0, errkit.Wrap(err, "Error, converting count value into int.")
	}
	log.Print("Counted number of records in the database.", field.M{"app": cas.name, "count": count})
	return count, nil
}

// Reset is used to reset or imitate disaster, in the database
func (cas *CassandraInstance) Reset(ctx context.Context) error {
	// delete keyspace and table if they already exist
	delRes := []string{"sh", "-c", fmt.Sprintf("cqlsh -u cassandra -p $CASSANDRA_PASSWORD -e "+
		"'drop table if exists restaurants.guests; drop keyspace if exists restaurants;' --request-timeout=%s", cqlTimeout)}
	_, stderr, err := cas.execCommand(ctx, delRes)
	if err != nil {
		return errkit.Wrap(err, "Error deleting resources while reseting application.", "stderr", stderr)
	}
	return nil
}

// Initialize is used to initialize the database or create schema
func (cas *CassandraInstance) Initialize(ctx context.Context) error {
	// create the keyspace
	createKS := []string{"sh", "-c", fmt.Sprintf("cqlsh -u cassandra -p $CASSANDRA_PASSWORD -e \"create keyspace "+
		"restaurants with replication  = {'class':'SimpleStrategy', 'replication_factor': 1};\" --request-timeout=%s", cqlTimeout)}
	_, stderr, err := cas.execCommand(ctx, createKS)
	if err != nil {
		return errkit.Wrap(err, "Error while creating the keyspace for application.", "stderr", stderr)
	}

	// create the table
	createTab := []string{"sh", "-c", fmt.Sprintf("cqlsh -u cassandra -p $CASSANDRA_PASSWORD -e \"create table "+
		"restaurants.guests (id UUID primary key, firstname text, lastname text, birthday timestamp);\" --request-timeout=%s", cqlTimeout)}
	_, stderr, err = cas.execCommand(ctx, createTab)
	if err != nil {
		return errkit.Wrap(err, "Error creating table.", "stderr", stderr)
	}
	return nil
}

func (cas *CassandraInstance) execCommand(ctx context.Context, command []string) (string, string, error) {
	podname, containername, err := kube.GetPodContainerFromStatefulSet(ctx, cas.cli, cas.namespace, cas.chart.Release)
	if err != nil || podname == "" {
		return "", "", errkit.Wrap(err, "Error getting the pod and container name.", "app", cas.name)
	}
	return kube.Exec(ctx, cas.cli, cas.namespace, podname, containername, command, nil)
}
