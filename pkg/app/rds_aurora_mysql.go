// Copyright 2021 The Kanister Authors.
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
	"database/sql"
	"fmt"
	"os"
	"strconv"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	rdserr "github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	aws "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/aws/ec2"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	AuroraDBInstanceClass = "db.r5.large"
	AuroraDBStorage       = 20
	DetailsCMName         = "dbconfig"
	mysqlConnectionString = "mysql -h %s -u %s -p%s %s -N -e"
)

type RDSAuroraMySQLDB struct {
	name              string
	cli               kubernetes.Interface
	namespace         string
	id                string
	host              string
	dbName            string
	username          string
	password          string
	accessID          string
	secretKey         string
	region            string
	sessionToken      string
	securityGroupID   string
	securityGroupName string
	testWorkloadName  string
}

func NewRDSAuroraMySQLDB(name, region string) App {
	return &RDSAuroraMySQLDB{
		name:              name,
		id:                fmt.Sprintf("test-%s", name),
		securityGroupName: fmt.Sprintf("%s-sg", name),
		region:            region,
		username:          "admin",
		password:          "secret99",
		dbName:            "testdb",
	}
}

func (a *RDSAuroraMySQLDB) Init(context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	var ok bool
	a.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	if a.region == "" {
		a.region, ok = os.LookupEnv(aws.Region)
		if !ok {
			return errors.New(fmt.Sprintf("Env var %s is not set", aws.Region))
		}
	}

	// If sessionToken is set, accessID and secretKey not required
	a.sessionToken, ok = os.LookupEnv(aws.SessionToken)
	if ok {
		return nil
	}

	a.accessID, ok = os.LookupEnv(aws.AccessKeyID)
	if !ok {
		return errors.New(fmt.Sprintf("Env var %s is not set", aws.AccessKeyID))
	}
	a.secretKey, ok = os.LookupEnv(aws.SecretAccessKey)
	if !ok {
		return errors.New(fmt.Sprintf("Env var %s is not set", aws.SecretAccessKey))
	}

	return nil
}

func (a *RDSAuroraMySQLDB) Install(ctx context.Context, namespace string) error {
	a.namespace = namespace

	// Get aws config
	awsConfig, region, err := a.getAWSConfig(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error getting aws config app=%s", a.name)
	}

	// Create ec2 client
	ec2Cli, err := ec2.NewClient(ctx, awsConfig, region)
	if err != nil {
		return err
	}

	a.testWorkloadName = fmt.Sprintf("%s-workload", a.name)

	testDeployment := bastionWorkload(ctx, a.testWorkloadName, "mysql", a.namespace)
	_, err = a.cli.AppsV1().Deployments(a.namespace).Create(context.Background(), testDeployment, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "Failed to create test deployment %s", a.testWorkloadName)
	}

	if err := kube.WaitOnDeploymentReady(ctx, a.cli, a.namespace, a.testWorkloadName); err != nil {
		return errors.Wrapf(err, "Failed while waiting for deployment %s to be ready", a.testWorkloadName)
	}
	// Create security group
	log.Info().Print("Creating security group.", field.M{"app": a.name, "name": a.securityGroupName})
	sg, err := ec2Cli.CreateSecurityGroup(ctx, a.securityGroupName, "To allow ingress to Aurora DB cluster")
	if err != nil {
		return errors.Wrap(err, "Error creating security group")
	}
	a.securityGroupID = *sg.GroupId

	// Add ingress rule
	_, err = ec2Cli.AuthorizeSecurityGroupIngress(ctx, a.securityGroupName, "0.0.0.0/0", "tcp", 3306)
	if err != nil {
		return errors.Wrap(err, "Error authorizing security group")
	}

	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return err
	}

	// Create RDS instance
	log.Info().Print("Creating RDS Aurora DB cluster.", field.M{"app": a.name, "id": a.id})
	_, err = rdsCli.CreateDBCluster(ctx, AuroraDBStorage, AuroraDBInstanceClass, a.id, string(function.DBEngineAuroraMySQL), a.dbName, a.username, a.password, []string{a.securityGroupID})
	if err != nil {
		return errors.Wrap(err, "Error creating DB cluster")
	}

	err = rdsCli.WaitUntilDBClusterAvailable(ctx, a.id)
	if err != nil {
		return errors.Wrap(err, "Error waiting for DB cluster to be available")
	}

	// create db instance in the cluster
	_, err = rdsCli.CreateDBInstanceInCluster(ctx, a.id, fmt.Sprintf("%s-instance-1", a.id), AuroraDBInstanceClass, string(function.DBEngineAuroraMySQL))
	if err != nil {
		return errors.Wrap(err, "Error creating an instance in Aurora DB cluster")
	}

	err = rdsCli.WaitUntilDBInstanceAvailable(ctx, fmt.Sprintf("%s-instance-1", a.id))
	if err != nil {
		return errors.Wrap(err, "Error waiting for DB instance to be available")
	}

	dbCluster, err := rdsCli.DescribeDBClusters(ctx, a.id)
	if err != nil {
		return err
	}
	if len(dbCluster.DBClusters) == 0 {
		return errors.New(fmt.Sprintf("Error installing application %s, DBCluster not available", a.name))
	}
	a.host = *dbCluster.DBClusters[0].Endpoint

	// Configmap that is going to store the details for blueprint
	cm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: DetailsCMName,
		},
		Data: map[string]string{
			"aurora.clusterID": a.id,
		},
	}

	_, err = a.cli.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	return err
}

func (a *RDSAuroraMySQLDB) IsReady(context.Context) (bool, error) {
	// we are already waiting for dbcluster using WaitUntilDBClusterAvailable while installing it
	return true, nil
}

func (a *RDSAuroraMySQLDB) Ping(ctx context.Context) error {
	log.Print("Pinging rds aurora database", field.M{"app": a.name})
	isReadyCommand := fmt.Sprintf(mysqlConnectionString+"'SELECT 1;'", a.host, a.username, a.password, a.dbName)

	pingCommand := []string{"sh", "-c", isReadyCommand}

	_, stderr, err := a.execCommand(ctx, pingCommand)
	if err != nil {
		return errors.Wrapf(err, "Error while Pinging the database: %s", stderr)
	}

	log.Print("Ping to the application was success.", field.M{"app": a.name})
	return nil
}

func (a *RDSAuroraMySQLDB) Insert(ctx context.Context) error {
	log.Print("Adding entry to database", field.M{"app": a.name})
	insert := fmt.Sprintf(mysqlConnectionString+
		"\"INSERT INTO pets VALUES ('Puffball', 'Diane', 'hamster', 'f', '1999-03-30', 'NULL');\"", a.host, a.username, a.password, a.dbName)

	insertQuery := []string{"sh", "-c", insert}
	_, stderr, err := a.execCommand(ctx, insertQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while inserting data into table: %s", stderr)
	}
	log.Info().Print("Inserted a row in test db.", field.M{"app": a.name})
	return nil
}

func (a *RDSAuroraMySQLDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting entries from database", field.M{"app": a.name})
	count := fmt.Sprintf(mysqlConnectionString+
		"\"SELECT COUNT(*) FROM pets;\"", a.host, a.username, a.password, a.dbName)

	countQuery := []string{"sh", "-c", count}
	stdout, stderr, err := a.execCommand(ctx, countQuery)
	if err != nil {
		return 0, errors.Wrapf(err, "Error while counting data into table: %s", stderr)
	}

	rowsReturned, err := strconv.Atoi(stdout)
	if err != nil {
		return 0, errors.Wrapf(err, "Error while converting response of count query: %s", stderr)
	}

	log.Info().Print("Counting rows in test db.", field.M{"app": a.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (a *RDSAuroraMySQLDB) Reset(ctx context.Context) error {
	log.Print("Resetting the mysql instance.", field.M{"app": a.name})

	delete := fmt.Sprintf(mysqlConnectionString+"\"DROP TABLE IF EXISTS pets;\"", a.host, a.username, a.password, a.dbName)
	deleteQuery := []string{"sh", "-c", delete}
	_, stderr, err := a.execCommand(ctx, deleteQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while deleting data into table: %s", stderr)
	}

	log.Info().Print("Database reset successful!", field.M{"app": a.name})
	return nil
}

func (a *RDSAuroraMySQLDB) Initialize(ctx context.Context) error {
	// Create table.
	log.Print("Initializing database", field.M{"app": a.name})
	createTable := fmt.Sprintf(mysqlConnectionString+"\"CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);\"", a.host, a.username, a.password, a.dbName)
	createQuery := []string{"sh", "-c", createTable}
	_, stderr, err := a.execCommand(ctx, createQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while creating the database: %s", stderr)
	}
	return nil
}

func (a *RDSAuroraMySQLDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		APIVersion: "v1",
		Name:       DetailsCMName,
		Namespace:  a.namespace,
		Resource:   "configmaps",
	}
}

func (a *RDSAuroraMySQLDB) Uninstall(ctx context.Context) error {
	awsConfig, region, err := a.getAWSConfig(ctx)
	if err != nil {
		return errors.Wrapf(err, "app=%s", a.name)
	}
	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return errors.Wrap(err, "Failed to create rds client. You may need to delete RDS resources manually. app=rds-postgresql")
	}

	descOp, err := rdsCli.DescribeDBClusters(ctx, a.id)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBClusterNotFoundFault {
				return err
			}
			log.Print("Aurora DB cluster is not found")
		}
	} else {
		// DB Cluster is present, delete and wait for it to be deleted
		if err := function.DeleteAuroraDBCluster(ctx, rdsCli, descOp, a.id); err != nil {
			return nil
		}
	}

	// Create ec2 client
	ec2Cli, err := ec2.NewClient(ctx, awsConfig, region)
	if err != nil {
		return errors.Wrap(err, "Failed to create ec2 client.")
	}

	// delete security group
	log.Info().Print("Deleting security group.", field.M{"app": a.name})
	_, err = ec2Cli.DeleteSecurityGroup(ctx, a.securityGroupID)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case "InvalidGroup.NotFound":
				log.Error().Print("Security group already deleted: InvalidGroup.NotFound.", field.M{"app": a.name, "name": a.securityGroupName})
			default:
				return errors.Wrapf(err, "Failed to delete security group. You may need to delete it manually. app=rds-postgresql name=%s", a.securityGroupName)
			}
		}
	}

	// Remove workload object created for executing commands
	err = a.cli.AppsV1().Deployments(a.namespace).Delete(ctx, a.testWorkloadName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "Error deleting Workload %s", a.testWorkloadName)
	}
	return nil
}

func (a *RDSAuroraMySQLDB) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (a *RDSAuroraMySQLDB) getAWSConfig(ctx context.Context) (*awssdk.Config, string, error) {
	config := make(map[string]string)
	config[aws.ConfigRegion] = a.region
	config[aws.AccessKeyID] = a.accessID
	config[aws.SecretAccessKey] = a.secretKey
	config[aws.SessionToken] = a.sessionToken
	return aws.GetConfig(ctx, config)
}

func (a *RDSAuroraMySQLDB) openDBConnection() (*sql.DB, error) {
	return sql.Open("mysql", fmt.Sprintf("%s:%s@(%s)/%s", a.username, a.password, a.host, a.dbName))
}

func (a RDSAuroraMySQLDB) closeDBConnection(db *sql.DB) error {
	return db.Close()
}

func (a RDSAuroraMySQLDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromDeployment(ctx, a.cli, a.namespace, a.testWorkloadName)
	if err != nil || podName == "" {
		return "", "", err
	}
	return kube.Exec(a.cli, a.namespace, podName, containerName, command, nil)
}
