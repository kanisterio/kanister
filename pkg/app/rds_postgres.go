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
	"os"
	"strconv"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsrds "github.com/aws/aws-sdk-go/service/rds"
	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/aws/ec2"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

type RDSPostgresDB struct {
	name                     string
	cli                      kubernetes.Interface
	namespace                string
	id                       string
	host                     string
	databases                []string
	dbSubnetGroup            string
	username                 string
	password                 string
	accessID                 string
	secretKey                string
	region                   string
	sessionToken             string
	securityGroupID          string
	securityGroupName        string
	configMapName            string
	secretName               string
	bastionDebugWorkloadName string
	publicAccess             bool
	vpcID                    string
}

const (
	dbInstanceType           = "db.t3.micro"
	postgresConnectionString = "PGPASSWORD=%s psql -h %s -p 5432 -U %s -d %s -t -c"
	subnetGroupDescription   = "kanister-test-subnet-group"
)

func NewRDSPostgresDB(name string, customRegion string) App {
	return &RDSPostgresDB{
		name:              name,
		id:                fmt.Sprintf("test-%s", name),
		securityGroupName: fmt.Sprintf("%s-sg", name),
		databases:         []string{"postgres", "template1"},
		username:          "master",
		password:          "secret99",
		region:            customRegion,
		configMapName:     fmt.Sprintf("%s-config", name),
		secretName:        fmt.Sprintf("%s-secret", name),
		publicAccess:      false,
	}
}

func (pdb *RDSPostgresDB) Init(ctx context.Context) error {
	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	var ok bool
	pdb.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	if pdb.region == "" {
		pdb.region, ok = os.LookupEnv(aws.Region)
		if !ok {
			return errkit.New("env var is not set", "name", aws.Region)
		}
	}

	// If sessionToken is set, accessID and secretKey not required
	pdb.sessionToken, ok = os.LookupEnv(aws.SessionToken)
	if ok {
		return nil
	}

	pdb.accessID, ok = os.LookupEnv(aws.AccessKeyID)
	if !ok {
		return errkit.New("env var is not set", "name", aws.AccessKeyID)
	}
	pdb.secretKey, ok = os.LookupEnv(aws.SecretAccessKey)
	if !ok {
		return errkit.New("env var is not set", "name", aws.SecretAccessKey)
	}
	return nil
}

func (pdb *RDSPostgresDB) Install(ctx context.Context, ns string) error {
	var err error
	pdb.namespace = ns

	// Create AWS config
	awsConfig, region, err := pdb.getAWSConfig(ctx)
	if err != nil {
		return errkit.Wrap(err, "Error getting aws config", "app", pdb.name)
	}

	ec2Cli, err := ec2.NewClient(ctx, awsConfig, region)
	if err != nil {
		return err
	}

	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return err
	}

	pdb.bastionDebugWorkloadName = fmt.Sprintf("%s-workload", pdb.name)

	deploymentSpec := bastionDebugWorkloadSpec(ctx, pdb.bastionDebugWorkloadName, "postgres", pdb.namespace)
	_, err = pdb.cli.AppsV1().Deployments(pdb.namespace).Create(ctx, deploymentSpec, metav1.CreateOptions{})
	if err != nil {
		return errkit.Wrap(err, "Failed to create deployment", "deployment", pdb.bastionDebugWorkloadName, "app", pdb.name)
	}

	if err := kube.WaitOnDeploymentReady(ctx, pdb.cli, pdb.namespace, pdb.bastionDebugWorkloadName); err != nil {
		return errkit.Wrap(err, "Failed while waiting for deployment to be ready", "deployment", pdb.bastionDebugWorkloadName, "app", pdb.name)
	}

	pdb.vpcID, err = vpcIDForRDSInstance(ctx, ec2Cli)
	if err != nil {
		return err
	}

	dbSubnetGroup, err := dbSubnetGroup(ctx, ec2Cli, rdsCli, pdb.vpcID, pdb.name, subnetGroupDescription)
	if err != nil {
		return err
	}
	pdb.dbSubnetGroup = dbSubnetGroup

	// Create security group
	log.Info().Print("Creating security group.", field.M{"app": pdb.name, "name": pdb.securityGroupName})
	sg, err := ec2Cli.CreateSecurityGroup(ctx, pdb.securityGroupName, "kanister-test-security-group", pdb.vpcID)
	if err != nil {
		return err
	}
	pdb.securityGroupID = *sg.GroupId

	// Add ingress rule
	log.Info().Print("Adding ingress rule to security group.", field.M{"app": pdb.name})
	_, err = ec2Cli.AuthorizeSecurityGroupIngress(ctx, pdb.securityGroupID, "0.0.0.0/0", "tcp", 5432)
	if err != nil {
		return err
	}

	// Create RDS instance
	log.Info().Print("Creating RDS instance.", field.M{"app": pdb.name, "id": pdb.id})
	_, err = rdsCli.CreateDBInstance(ctx, awssdk.Int64(20), dbInstanceType, pdb.id, "postgres", pdb.username, pdb.password, []string{pdb.securityGroupID}, awssdk.Bool(pdb.publicAccess), nil, pdb.dbSubnetGroup)
	if err != nil {
		return err
	}

	// Wait for DB to be ready
	log.Info().Print("Waiting for rds to be ready.", field.M{"app": pdb.name})
	err = rdsCli.WaitUntilDBInstanceAvailable(ctx, pdb.id)
	if err != nil {
		return err
	}

	// Find host of the instance
	dbInstance, err := rdsCli.DescribeDBInstances(ctx, pdb.id)
	if err != nil {
		return err
	}
	pdb.host = *dbInstance.DBInstances[0].Endpoint.Address

	// Create configmap
	dbconfig := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pdb.configMapName,
		},
		Data: map[string]string{
			"postgres.instanceid": pdb.id,
			"postgres.host":       pdb.host,
			"postgres.databases":  makeYamlList(pdb.databases),
			"postgres.user":       pdb.username,
			"postgres.secret":     pdb.secretName,
		},
	}
	_, err = pdb.cli.CoreV1().ConfigMaps(ns).Create(ctx, dbconfig, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create secret
	dbsecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pdb.secretName,
		},
		StringData: map[string]string{
			"password":          pdb.password,
			"username":          pdb.username,
			"access_key_id":     pdb.accessID,
			"secret_access_key": pdb.secretKey,
			"aws_region":        pdb.region,
		},
	}
	_, err = pdb.cli.CoreV1().Secrets(ns).Create(ctx, dbsecret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (pdb *RDSPostgresDB) IsReady(ctx context.Context) (bool, error) {
	return true, nil
}

func (pdb *RDSPostgresDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		APIVersion: "v1",
		Name:       pdb.configMapName,
		Namespace:  pdb.namespace,
		Resource:   "configmaps",
	}
}

// Ping makes and tests DB connection
func (pdb *RDSPostgresDB) Ping(ctx context.Context) error {
	log.Print("Pinging rds postgres database", field.M{"app": pdb.name})
	// Get connection info from configmap
	dbconfig, err := pdb.cli.CoreV1().ConfigMaps(pdb.namespace).Get(ctx, pdb.configMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Get secret creds
	dbsecret, err := pdb.cli.CoreV1().Secrets(pdb.namespace).Get(ctx, pdb.secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Parse databases from config data
	var databases []string
	if err := yaml.Unmarshal([]byte(dbconfig.Data["postgres.databases"]), &databases); err != nil {
		return err
	}
	if databases == nil {
		return errkit.New("Databases are missing from configmap")
	}

	isReadyQuery := fmt.Sprintf(postgresConnectionString+"'SELECT version();'", dbsecret.Data["password"], dbconfig.Data["postgres.host"], dbconfig.Data["postgres.user"], databases[0])

	pingCommand := []string{"sh", "-c", isReadyQuery}

	_, stderr, err := pdb.execCommand(ctx, pingCommand)
	if err != nil {
		return errkit.Wrap(err, "Error while Pinging the database", "stderr", stderr, "app", pdb.name)
	}
	log.Print("Ping to the application was successful.", field.M{"app": pdb.name})
	return nil
}

func (pdb RDSPostgresDB) Insert(ctx context.Context) error {
	log.Print("Adding entry to database", field.M{"app": pdb.name})
	now := time.Now().Format(time.RFC3339Nano)
	insertQuery := fmt.Sprintf(postgresConnectionString+
		"\"INSERT INTO inventory (name) VALUES ('%s');\"", pdb.password, pdb.host, pdb.username, pdb.databases[0], now)

	insertCommand := []string{"sh", "-c", insertQuery}
	_, stderr, err := pdb.execCommand(ctx, insertCommand)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting data into table", "stderr", stderr, "app", pdb.name)
	}
	log.Info().Print("Inserted a row in test db.", field.M{"app": pdb.name})
	return nil
}

func (pdb RDSPostgresDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting entries from database", field.M{"app": pdb.name})
	countQuery := fmt.Sprintf(postgresConnectionString+
		"\"SELECT COUNT(*) FROM inventory;\"", pdb.password, pdb.host, pdb.username, pdb.databases[0])

	countCommand := []string{"sh", "-c", countQuery}
	stdout, stderr, err := pdb.execCommand(ctx, countCommand)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while counting data of table", "stderr", stderr, "app", pdb.name)
	}

	rowsReturned, err := strconv.Atoi(stdout)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while converting response of count query", "stderr", stderr, "app", pdb.name)
	}

	log.Info().Print("Counting rows in test db.", field.M{"app": pdb.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (pdb RDSPostgresDB) Reset(ctx context.Context) error {
	log.Print("Resetting database", field.M{"app": pdb.name})
	deleteQuery := fmt.Sprintf(postgresConnectionString+"\"DROP TABLE IF EXISTS inventory;\"", pdb.password, pdb.host, pdb.username, pdb.databases[0])
	deleteCommand := []string{"sh", "-c", deleteQuery}
	_, stderr, err := pdb.execCommand(ctx, deleteCommand)
	if err != nil {
		return errkit.Wrap(err, "Error while deleting data from table", "stderr", stderr, "app", pdb.name)
	}
	log.Info().Print("Database reset successful!", field.M{"app": pdb.name})
	return nil
}

// Initialize is used to initialize the database or create schema
func (pdb RDSPostgresDB) Initialize(ctx context.Context) error {
	// Create table.
	log.Print("Initializing database", field.M{"app": pdb.name})
	createQuery := fmt.Sprintf(postgresConnectionString+"\"CREATE TABLE inventory (id serial PRIMARY KEY, name VARCHAR(50));\"", pdb.password, pdb.host, pdb.username, pdb.databases[0])
	createCommand := []string{"sh", "-c", createQuery}
	_, stderr, err := pdb.execCommand(ctx, createCommand)
	if err != nil {
		return errkit.Wrap(err, "Error while initializing the database", "stderr", stderr, "app", pdb.name)
	}
	return nil
}

func (pdb RDSPostgresDB) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"dbconfig": {
			Kind:      "configmap",
			Name:      pdb.configMapName,
			Namespace: pdb.namespace,
		},
	}
}

func (pdb RDSPostgresDB) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"dbsecret": {
			Kind:      "secret",
			Name:      pdb.secretName,
			Namespace: pdb.namespace,
		},
	}
}

func (pdb RDSPostgresDB) Uninstall(ctx context.Context) error {
	// Create AWS config
	awsConfig, region, err := pdb.getAWSConfig(ctx)
	if err != nil {
		return errkit.Wrap(err, "Error getting aws config", "app", pdb.name)
	}
	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return errkit.Wrap(err, "Failed to create rds client. You may need to delete RDS resources manually. app=rds-postgresql")
	}

	// Delete rds instance
	log.Info().Print("Deleting rds instance", field.M{"app": pdb.name})
	_, err = rdsCli.DeleteDBInstance(ctx, pdb.id)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case awsrds.ErrCodeDBInstanceNotFoundFault:
				log.Info().Print("RDS instance already deleted: ErrCodeDBInstanceNotFoundFault.", field.M{"app": pdb.name, "id": pdb.id})
			default:
				return errkit.Wrap(err, "Failed to delete rds instance. You may need to delete it manually.", "app", "rds-postgresql", "id", pdb.id)
			}
		}
	}

	// Waiting for rds to be deleted
	if err == nil {
		log.Info().Print("Waiting for rds to be deleted", field.M{"app": pdb.name})
		err = rdsCli.WaitUntilDBInstanceDeleted(ctx, pdb.id)
		if err != nil {
			return errkit.Wrap(err, "Failed to wait for rds instance till delete succeeds.", "app", "rds-postgresql", "id", pdb.id)
		}
	}

	// Create ec2 client
	ec2Cli, err := ec2.NewClient(ctx, awsConfig, region)
	if err != nil {
		return errkit.Wrap(err, "Failed to ec2 client. You may need to delete EC2 resources manually. app=rds-postgresql")
	}

	log.Info().Print("Deleting db subnet group.", field.M{"app": pdb.name})
	_, err = rdsCli.DeleteDBSubnetGroup(ctx, pdb.dbSubnetGroup)
	if err != nil {
		// If the subnet group does not exist, ignore the error and return
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case awsrds.ErrCodeDBSubnetGroupNotFoundFault:
				log.Info().Print("Subnet Group Does not exist: ErrCodeDBSubnetGroupNotFoundFault.", field.M{"app": pdb.name, "id": pdb.id})
			default:
				return errkit.Wrap(err, "Failed to delete db subnet group. You may need to delete it manually.", "app", "rds-postgresql", "name", pdb.dbSubnetGroup)
			}
		}
	}

	// Delete security group
	log.Info().Print("Deleting security group.", field.M{"app": pdb.name})
	_, err = ec2Cli.DeleteSecurityGroup(ctx, pdb.securityGroupID)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case "InvalidGroup.NotFound":
				log.Error().Print("Security group already deleted: InvalidGroup.NotFound.", field.M{"app": pdb.name, "name": pdb.securityGroupName})
			default:
				return errkit.Wrap(err, "Failed to delete security group. You may need to delete it manually.", "app", "rds-postgresql", "name", pdb.securityGroupName)
			}
		}
	}
	// Remove workload object created for executing commands
	err = pdb.cli.AppsV1().Deployments(pdb.namespace).Delete(ctx, pdb.bastionDebugWorkloadName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Error deleting Workload", "name", pdb.bastionDebugWorkloadName, "app", pdb.name)
	}

	return nil
}

func (pdb RDSPostgresDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (pdb RDSPostgresDB) getAWSConfig(ctx context.Context) (*awssdk.Config, string, error) {
	config := make(map[string]string)
	config[aws.ConfigRegion] = pdb.region
	config[aws.AccessKeyID] = pdb.accessID
	config[aws.SecretAccessKey] = pdb.secretKey
	config[aws.SessionToken] = pdb.sessionToken
	return aws.GetConfig(ctx, config)
}

func makeYamlList(dbs []string) string {
	dbsYaml := ""
	for _, db := range dbs {
		dbsYaml += fmt.Sprintf("- %s\n", db)
	}
	return dbsYaml
}

func (pdb RDSPostgresDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromDeployment(ctx, pdb.cli, pdb.namespace, pdb.bastionDebugWorkloadName)
	if err != nil || podName == "" {
		return "", "", err
	}
	return kube.Exec(ctx, pdb.cli, pdb.namespace, podName, containerName, command, nil)
}
