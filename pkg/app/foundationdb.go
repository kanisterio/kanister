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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kanisterio/errkit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	fdbReadyTimeout = 15 * time.Minute

	// Format of the name of the pods that get generated is of form helmReleaseName-suffix-index
	// If we run fdb cluster in double redundancy mode, 7 pods get spinned up.
	// We are using the second pod, because of certain issues in the first pod
	// that gets spinned up. podNameSuffix stores the suffix-index part
	podNameSuffix = "sample-2"
)

// FoundationDB has fields of foundationdb instance
type FoundationDB struct {
	name           string
	namespace      string
	cli            kubernetes.Interface
	oprReleaseName string
	fdbReleaseName string
}

// NewFoundationDB initializes and returns the foundation db instance
func NewFoundationDB(name string) App {
	return &FoundationDB{
		name:           name,
		oprReleaseName: appendRandString("fdb-operator"),
		fdbReleaseName: appendRandString("fdb-instance"),
	}
}

// Init initialises the kubernetes config details
func (fdb *FoundationDB) Init(ctx context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	fdb.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

// Install is used to install the database
func (fdb *FoundationDB) Install(ctx context.Context, namespace string) error {
	fdb.namespace = namespace

	helmVersion, err := helm.FindVersion()
	if err != nil {
		return errkit.Wrap(err, "Couldn't find the helm version.")
	}

	var oprARG, instARG []string
	switch helmVersion {
	case helm.V2:
		oprARG = []string{"install", "../../helm/fdb-operator/", "--name=" + fdb.oprReleaseName, "-n", fdb.namespace}
		instARG = []string{"install", "../../helm/fdb-instance", "--name=" + fdb.fdbReleaseName, "-n", fdb.namespace}
	case helm.V3:
		oprARG = []string{"install", fdb.oprReleaseName, "../../helm/fdb-operator/", "-n", fdb.namespace, "--wait"}
		instARG = []string{"install", fdb.fdbReleaseName, "../../helm/fdb-instance", "-n", fdb.namespace, "--wait"}
	}

	out, err := helm.RunCmdWithTimeout(ctx, helm.GetHelmBinName(), oprARG)
	if err != nil {
		return errkit.Wrap(err, "Error installing the operator.", "app", fdb.name, "out", out)
	}

	out, err = helm.RunCmdWithTimeout(ctx, helm.GetHelmBinName(), instARG)
	if err != nil {
		return errkit.Wrap(err, "Error installing the application", "app", fdb.name, "out", out)
	}

	log.Print("Application was installed successfully.", field.M{"app": fdb.name})
	return nil
}

// IsReady is used to check the database that is installed is ready or not
func (fdb *FoundationDB) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for application to be ready.", field.M{"app": fdb.name})

	ctx, cancel := context.WithTimeout(ctx, fdbReadyTimeout)
	defer cancel()

	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		_, err := fdb.getRunningFDBPod()
		return err == nil, nil
	})
	if err != nil {
		return false, err
	}

	log.Print("Application is ready.", field.M{"app": fdb.name})
	return true, nil
}

// Object will return the controller reference that is installed to run the database
func (fdb *FoundationDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		APIVersion: "v1beta1",
		Group:      "apps.foundationdb.org",
		Name:       fmt.Sprintf("%s-sample", fdb.fdbReleaseName),
		Namespace:  fdb.namespace,
		Resource:   "foundationdbclusters",
	}
}

// Uninstall used to uninstall the database
func (fdb *FoundationDB) Uninstall(ctx context.Context) error {
	unintFDB := []string{"delete", "-n", fdb.namespace, fdb.fdbReleaseName}
	out, err := helm.RunCmdWithTimeout(ctx, helm.GetHelmBinName(), unintFDB)
	if err != nil {
		return errkit.Wrap(err, "Error uninstalling the fdb instance.", "out", out)
	}

	uninstOpr := []string{"delete", "-n", fdb.namespace, fdb.oprReleaseName}
	out, err = helm.RunCmdWithTimeout(ctx, helm.GetHelmBinName(), uninstOpr)

	return errkit.Wrap(err, "Error uninstalling the operator.", "out", out)
}

func (fdb *FoundationDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (fdb *FoundationDB) getRunningFDBPod() (string, error) {
	// Format of the name of the pods that get generated is
	// helmReleaseName-sample-index
	podName := fmt.Sprintf("%s-%s", fdb.fdbReleaseName, podNameSuffix)
	pod, err := fdb.cli.CoreV1().Pods(fdb.namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if len(pod.Status.ContainerStatuses) == 0 {
		return "", errkit.New("Couldn't find ready pod.", "name", fdb.name, "namespace", fdb.namespace)
	}
	if !pod.Status.ContainerStatuses[0].Ready {
		return "", errkit.New("Couldn't find ready pod.", "name", fdb.name, "namespace", fdb.namespace)
	}

	return pod.GetName(), nil
}

// Ping is used to check if the database is able to accept the request
func (fdb *FoundationDB) Ping(ctx context.Context) error {
	log.Print("Pinging the appliation to check if its accessible.", field.M{"app": fdb.name})

	pingCMD := []string{"sh", "-c", "fdbcli"}
	stdout, stderr, err := fdb.execCommand(ctx, pingCMD)
	if err != nil {
		return errkit.Wrap(err, "Error while pinging the database.", "app", fdb.name, "stderr", stderr)
	}

	// This is how we get the output of fdbcli
	// Using cluster file `/var/dynamic-conf/fdb.cluster'.
	// The database is available.
	// Welcome to the fdbcli. For help, type `help'.

	if !strings.Contains(stdout, "The database is available") {
		return errkit.New("Database is still not ready.", "name", fdb.name)
	}

	return nil
}

// Insert is used to insert some records into the database
func (fdb *FoundationDB) Insert(ctx context.Context) error {
	// generate a random key
	insertCMD := []string{"sh", "-c", fmt.Sprintf("fdbcli --exec 'writemode on; set %s vivek; '", uuid.New())}
	_, stderr, err := fdb.execCommand(ctx, insertCMD)

	return errkit.Wrap(err, "Error inserting data into the database.", "stderr", stderr)
}

// Count is used to count the number of records
func (fdb *FoundationDB) Count(ctx context.Context) (int, error) {
	countCMD := []string{"sh", "-c", "fdbcli --exec \"getrangekeys '' \xFF \""}
	stdout, stderr, err := fdb.execCommand(ctx, countCMD)
	if err != nil {
		return 0, errkit.Wrap(err, "Error counting the records in the database.", "app", fdb.name, "stderr", stderr)
	}

	// Below is how we get the output of getrangekeys
	// Range limited to 25 keys
	// `1'
	// `2'
	// `3'
	count := len(strings.Split(stdout, "\n")) - 1

	return count, nil
}

// Reset is used to reset the database
func (fdb *FoundationDB) Reset(ctx context.Context) error {
	resetCMD := []string{"sh", "-c", "fdbcli --exec \"writemode on; clearrange '' \xFF\" "}
	stdout, stderr, err := fdb.execCommand(ctx, resetCMD)

	return errkit.Wrap(err, "Error resetting the database.", "stderr", stderr, "app", fdb.name, "stdout", stdout)
}

// Initialize is used to initialize the database or create schema
func (fdb *FoundationDB) Initialize(ctx context.Context) error {
	return nil
}

func (fdb *FoundationDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, err := fdb.getRunningFDBPod()
	if err != nil {
		return "", "", err
	}

	containers, err := kube.PodContainers(ctx, fdb.cli, fdb.namespace, podName)
	if err != nil || len(containers) == 0 {
		return "", "", err
	}

	return kube.Exec(ctx, fdb.cli, fdb.namespace, podName, containers[0].Name, command, nil)
}
