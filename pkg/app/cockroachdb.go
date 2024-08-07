package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

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

const cReadyTimeout = 10 * time.Minute

type CockroachDB struct {
	name      string
	namespace string
	cacrt     string
	tlscrt    string
	tlskey    string
	cli       kubernetes.Interface
	chart     helm.ChartInfo
}

// NewCockroachDB Last tested working version "22.1.5"
func NewCockroachDB(name string) App {
	return &CockroachDB{
		name: name,
		chart: helm.ChartInfo{
			Release:  appendRandString(name),
			RepoName: helm.CockroachDBRepoName,
			RepoURL:  helm.CockroachDBRepoURL,
			Chart:    "cockroachdb",
			Values: map[string]string{
				"image.tag":        "v22.1.5",
				"image.pullPolicy": "IfNotPresent",
			},
		},
	}
}

func (c *CockroachDB) Init(context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	c.cli, err = kubernetes.NewForConfig(cfg)
	return err
}

func (c *CockroachDB) Install(ctx context.Context, namespace string) error { //nolint:dupl // Not a duplicate, common code already extracted
	log.Info().Print("Installing cockroachdb cluster helm chart.", field.M{"app": c.name})
	c.namespace = namespace

	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}

	if err = cli.AddRepo(ctx, c.chart.RepoName, c.chart.RepoURL); err != nil {
		return errkit.Wrap(err, "Failed to install helm repo.", "app", c.name, "repo", c.chart.RepoName)
	}

	_, err = cli.Install(ctx, fmt.Sprintf("%s/%s", c.chart.RepoName, c.chart.Chart), c.chart.Version, c.chart.Release, c.namespace, c.chart.Values, false, false)
	return errkit.Wrap(err, "Failed to install helm chart.", "app", c.name, "chart", c.chart.Chart, "release", c.chart.Release)
}

func (c *CockroachDB) IsReady(ctx context.Context) (bool, error) {
	log.Info().Print("Waiting for cockroachdb cluster to be ready.", field.M{"app": c.name, "namespace": c.namespace, "release": c.chart.Release})
	ctx, cancel := context.WithTimeout(ctx, cReadyTimeout)
	defer cancel()
	err := kube.WaitOnStatefulSetReady(ctx, c.cli, c.namespace, c.chart.Release)
	if err != nil {
		log.WithError(err).Print("Error Occurred --> ", field.M{"error": err.Error()})
		return false, err
	}
	log.Info().Print("Application instance is ready.", field.M{"app": c.name})

	// Get the secret that gets installed with Cockroach DB installation
	// and read the client certs from that secret.
	// These client certs are then stored in a directory in client pod so
	// that we can use them later to communicate with cockroach DB
	// cluster, and executing queries like Creating Database and Table,
	// Inserting Data, Setting up Garbage Collection Time,
	// Delete Database etc
	secretName := fmt.Sprintf("%s-client-secret", c.chart.Release)
	secret, err := c.cli.CoreV1().Secrets(c.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if _, exist := secret.Data["ca.crt"]; !exist {
		return false, errkit.New("Error: ca.crt not found in the cluster credential", "secret", secretName)
	}
	c.cacrt = string(secret.Data["ca.crt"])

	if _, exist := secret.Data["tls.crt"]; !exist {
		return false, errkit.New("Error: tls.crt not found in the cluster credential", "secret", secretName)
	}
	c.tlscrt = string(secret.Data["tls.crt"])

	if _, exist := secret.Data["tls.key"]; !exist {
		return false, errkit.New("Error: tls.key not found in the cluster credential", "secret", secretName)
	}
	c.tlskey = string(secret.Data["tls.key"])

	createCrtDirCmd := "mkdir -p /cockroach/cockroach-client-certs"
	createCrtDir := []string{"sh", "-c", createCrtDirCmd}
	_, stderr, err := c.execCommand(ctx, createCrtDir)
	if err != nil {
		return false, errkit.Wrap(err, "Error while Creating Cert Directory", "stderr", stderr)
	}

	createCaCrtCmd := fmt.Sprintf("echo '%s' >> /cockroach/cockroach-client-certs/ca.crt", c.cacrt)
	createCaCrt := []string{"sh", "-c", createCaCrtCmd}
	_, stderr, err = c.execCommand(ctx, createCaCrt)
	if err != nil {
		return false, errkit.Wrap(err, "Error while Creating ca.crt", "stderr", stderr)
	}

	createTLSCrtCmd := fmt.Sprintf("echo '%s'>> /cockroach/cockroach-client-certs/client.root.crt", c.tlscrt)
	createTLSCrt := []string{"sh", "-c", createTLSCrtCmd}
	_, stderr, err = c.execCommand(ctx, createTLSCrt)
	if err != nil {
		return false, errkit.Wrap(err, "Error while Creating tls.crt", "stderr", stderr)
	}

	createTLSKeyCmd := fmt.Sprintf("echo '%s' >> /cockroach/cockroach-client-certs/client.root.key", c.tlskey)
	createTLSKey := []string{"sh", "-c", createTLSKeyCmd}
	_, stderr, err = c.execCommand(ctx, createTLSKey)
	if err != nil {
		return false, errkit.Wrap(err, "Error while Creating tls.key", "stderr", stderr)
	}

	changeFilePermCmd := "cd /cockroach/cockroach-client-certs/ && chmod 0600 *"
	changeFilePerm := []string{"sh", "-c", changeFilePermCmd}
	_, stderr, err = c.execCommand(ctx, changeFilePerm)
	if err != nil {
		return false, errkit.Wrap(err, "Error while changing certificate file permissions", "stderr", stderr)
	}

	changeDefaultGCTimeCmd := "./cockroach sql --certs-dir=/cockroach/cockroach-client-certs -e 'ALTER RANGE default CONFIGURE ZONE USING gc.ttlseconds = 10;'"
	changeDefaultGCTime := []string{"sh", "-c", changeDefaultGCTimeCmd}
	_, stderr, err = c.execCommand(ctx, changeDefaultGCTime)
	if err != nil {
		return false, errkit.Wrap(err, "Error while setting up Garbage Collection time", "stderr", stderr)
	}

	return err == nil, err
}

func (c *CockroachDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "statefulset",
		Name:      c.chart.Release,
		Namespace: c.namespace,
	}
}

func (c *CockroachDB) Uninstall(ctx context.Context) error {
	cli, err := helm.NewCliClient()
	if err != nil {
		return errkit.Wrap(err, "failed to create helm client")
	}
	err = cli.Uninstall(ctx, c.chart.Release, c.namespace)
	if err != nil {
		log.WithError(err).Print("Failed to uninstall app, you will have to uninstall it manually.", field.M{"app": c.name})
		return err
	}
	log.Print("Uninstalled application.", field.M{"app": c.name})

	return nil
}

func (c *CockroachDB) GetClusterScopedResources(context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (c *CockroachDB) Ping(ctx context.Context) error {
	log.Print("Pinging the cockroachdb database.", field.M{"app": c.name})

	loginCmd := fmt.Sprintf("./cockroach sql --certs-dir=/cockroach/cockroach-client-certs --host=%s-public", c.chart.Release)
	login := []string{"sh", "-c", loginCmd}
	_, stderr, err := c.execCommand(ctx, login)
	if err != nil {
		return errkit.Wrap(err, "Error while pinging database", "stderr", stderr)
	}

	log.Print("Ping to the application was success.", field.M{"app": c.name})
	return nil
}

// Initialize is used to initialize the database or create schema
func (c *CockroachDB) Initialize(ctx context.Context) error {
	createDatabaseCMD := "./cockroach sql --certs-dir=/cockroach/cockroach-client-certs -e 'CREATE DATABASE bank; CREATE TABLE bank.accounts (id INT, balance DECIMAL);'"
	createDatabase := []string{"sh", "-c", createDatabaseCMD}
	_, stderr, err := c.execCommand(ctx, createDatabase)
	if err != nil {
		return errkit.Wrap(err, "Error while initializing", "stderr", stderr)
	}
	return nil
}

func (c *CockroachDB) Insert(ctx context.Context) error {
	log.Print("Inserting some records in  cockroachdb instance.", field.M{"app": c.name})

	insertRecordCMD := "./cockroach sql --certs-dir=/cockroach/cockroach-client-certs -e 'INSERT INTO bank.accounts VALUES (1, 1000.50);'"
	insertRecord := []string{"sh", "-c", insertRecordCMD}
	_, stderr, err := c.execCommand(ctx, insertRecord)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting the data into database", "stderr", stderr)
	}

	log.Print("Successfully inserted records in the application.", field.M{"app": c.name})
	return nil
}

func (c *CockroachDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting the records from the cockroachdb instance.", field.M{"app": c.name})

	selectRowsCMD := "./cockroach sql --certs-dir=/cockroach/cockroach-client-certs -e 'SELECT COUNT(*) FROM bank.accounts;'"
	selectRows := []string{"sh", "-c", selectRowsCMD}
	stdout, stderr, err := c.execCommand(ctx, selectRows)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while counting the data of the database", "stderr", stderr)
	}
	// output returned from above query is "count\n3"
	// get the returned count and convert it to int, to return
	rowsReturned, err := strconv.Atoi(strings.Split(stdout, "\n")[1])
	if err != nil {
		return 0, errkit.Wrap(err, "Error while converting row count to int.")
	}
	log.Print("Count that we received from application is.", field.M{"app": c.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (c *CockroachDB) Reset(ctx context.Context) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, cReadyTimeout)
	defer waitCancel()
	err := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		err := c.Ping(ctx)
		return err == nil, nil
	})

	if err != nil {
		return errkit.Wrap(err, "Error waiting for application to be ready to reset it", "app", c.name)
	}

	log.Print("Resetting the cockroachdb instance.", field.M{"app": "cockroachdb"})

	// delete all the data from the table
	deleteFromTableCMD := "./cockroach sql --certs-dir=/cockroach/cockroach-client-certs -e 'DROP DATABASE IF EXISTS bank;'"
	deleteFromTable := []string{"sh", "-c", deleteFromTableCMD}
	_, stderr, err := c.execCommand(ctx, deleteFromTable)
	if err != nil {
		return errkit.Wrap(err, "Error while dropping the table", "stderr", stderr)
	}
	// Even though the table is deleted from the database, it's present in the
	// descriptor table. We will have to wait for it to be deleted from there  as
	// well (using garbage collection), so that we can restore the snapshot in
	// the same DB cluster.
	err = poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		err = c.waitForGC(ctx)
		return err == nil, nil
	})
	log.Print("Reset of the application was successful.", field.M{"app": c.name})

	return nil
}

func (c *CockroachDB) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (c *CockroachDB) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"cockroachSecret": {
			Kind:      "Secret",
			Name:      c.chart.Release + "-client-secret",
			Namespace: c.namespace,
		},
	}
}

func (c *CockroachDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromStatefulSet(ctx, c.cli, c.namespace, c.chart.Release)
	if err != nil || podName == "" {
		return "", "", errkit.Wrap(err, "Error getting pod and container name.", "app", c.name)
	}
	return kube.Exec(ctx, c.cli, c.namespace, podName, containerName, command, nil)
}

func (c *CockroachDB) waitForGC(ctx context.Context) error {
	log.Info().Print("Getting Data from descriptor table", field.M{"app": c.name})
	getDescriptorCMD := "./cockroach sql --certs-dir=/cockroach/cockroach-client-certs -e 'SELECT * FROM system.descriptor;'"
	getDescriptor := []string{"sh", "-c", getDescriptorCMD}
	stdout, stderr, err := c.execCommand(ctx, getDescriptor)
	if err != nil {
		return errkit.Wrap(err, "Error while getiing descriptor table data", "stderr", stderr)
	}
	bankInDescriptor := strings.Contains(stdout, "bank") || strings.Contains(stdout, "account")
	log.Info().Print("bankInDescriptor:  ", field.M{"value": bankInDescriptor})
	if bankInDescriptor {
		return errkit.New("Bank Database exists. Waiting for garbage collector to run and remove the database")
	}
	return nil
}
