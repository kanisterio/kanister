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
	"fmt"
	"os"
	"strconv"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsrds "github.com/aws/aws-sdk-go/service/rds"
	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
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
	name                     string
	cli                      kubernetes.Interface
	namespace                string
	id                       string
	host                     string
	dbName                   string
	dbSubnetGroup            string
	username                 string
	password                 string
	accessID                 string
	secretKey                string
	region                   string
	sessionToken             string
	securityGroupID          string
	securityGroupName        string
	bastionDebugWorkloadName string
	publicAccess             bool
	vpcID                    string
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
		publicAccess:      false,
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
			return errkit.New("Env var is not set", "name", aws.Region)
		}
	}

	// If sessionToken is set, accessID and secretKey not required
	a.sessionToken, ok = os.LookupEnv(aws.SessionToken)
	if ok {
		return nil
	}

	a.accessID, ok = os.LookupEnv(aws.AccessKeyID)
	if !ok {
		return errkit.New("Env var is not set", "name", aws.AccessKeyID)
	}
	a.secretKey, ok = os.LookupEnv(aws.SecretAccessKey)
	if !ok {
		return errkit.New("Env var is not set", "name", aws.SecretAccessKey)
	}

	return nil
}

func (a *RDSAuroraMySQLDB) Install(ctx context.Context, namespace string) error {
	a.namespace = namespace

	// Get aws config
	awsConfig, region, err := a.getAWSConfig(ctx)
	if err != nil {
		return errkit.Wrap(err, "Error getting aws config", "app", a.name)
	}

	// Create ec2 client
	ec2Cli, err := ec2.NewClient(ctx, awsConfig, region)
	if err != nil {
		return err
	}

	a.bastionDebugWorkloadName = fmt.Sprintf("%s-workload", a.name)

	deploymentSpec := bastionDebugWorkloadSpec(ctx, a.bastionDebugWorkloadName, "mysql", a.namespace)
	_, err = a.cli.AppsV1().Deployments(a.namespace).Create(ctx, deploymentSpec, metav1.CreateOptions{})
	if err != nil {
		return errkit.Wrap(err, "Failed to create deployment", "deployment", a.bastionDebugWorkloadName, "app", a.name)
	}

	if err := kube.WaitOnDeploymentReady(ctx, a.cli, a.namespace, a.bastionDebugWorkloadName); err != nil {
		return errkit.Wrap(err, "Failed while waiting for deployment to be ready", "deployment", a.bastionDebugWorkloadName, "app", a.name)
	}

	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return err
	}

	a.vpcID, err = vpcIDForRDSInstance(ctx, ec2Cli)
	if err != nil {
		return err
	}

	dbSubnetGroup, err := dbSubnetGroup(ctx, ec2Cli, rdsCli, a.vpcID, a.name, subnetGroupDescription)
	if err != nil {
		return err
	}
	a.dbSubnetGroup = dbSubnetGroup

	// Create security group
	log.Info().Print("Creating security group.", field.M{"app": a.name, "name": a.securityGroupName})
	sg, err := ec2Cli.CreateSecurityGroup(ctx, a.securityGroupName, "To allow ingress to Aurora DB cluster", a.vpcID)
	if err != nil {
		return errkit.Wrap(err, "Error creating security group")
	}
	a.securityGroupID = *sg.GroupId

	// Add ingress rule
	_, err = ec2Cli.AuthorizeSecurityGroupIngress(ctx, a.securityGroupID, "0.0.0.0/0", "tcp", 3306)
	if err != nil {
		return errkit.Wrap(err, "Error authorizing security group")
	}

	// Create RDS instance
	log.Info().Print("Creating RDS Aurora DB cluster.", field.M{"app": a.name, "id": a.id})
	_, err = rdsCli.CreateDBCluster(ctx, AuroraDBStorage, AuroraDBInstanceClass, a.id, a.dbSubnetGroup, string(function.DBEngineAuroraMySQL), a.dbName, a.username, a.password, []string{a.securityGroupID})
	if err != nil {
		return errkit.Wrap(err, "Error creating DB cluster")
	}

	err = rdsCli.WaitUntilDBClusterAvailable(ctx, a.id)
	if err != nil {
		return errkit.Wrap(err, "Error waiting for DB cluster to be available")
	}

	_, err = rdsCli.CreateDBInstance(ctx, nil, AuroraDBInstanceClass, fmt.Sprintf("%s-instance-1", a.id), string(function.DBEngineAuroraMySQL), "", "", nil, awssdk.Bool(a.publicAccess), awssdk.String(a.id), a.dbSubnetGroup)
	if err != nil {
		return errkit.Wrap(err, "Error creating an instance in Aurora DB cluster")
	}

	err = rdsCli.WaitUntilDBInstanceAvailable(ctx, fmt.Sprintf("%s-instance-1", a.id))
	if err != nil {
		return errkit.Wrap(err, "Error waiting for DB instance to be available")
	}

	dbCluster, err := rdsCli.DescribeDBClusters(ctx, a.id)
	if err != nil {
		return err
	}
	if len(dbCluster.DBClusters) == 0 {
		return errkit.New("Error installing application %s, DBCluster not available", "name", a.name)
	}
	a.host = *dbCluster.DBClusters[0].Endpoint

	// Configmap that is going to store the details for blueprint
	cm := &corev1.ConfigMap{
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
	pingQuery := fmt.Sprintf(mysqlConnectionString+"'SELECT 1;'", a.host, a.username, a.password, a.dbName)

	pingCommand := []string{"sh", "-c", pingQuery}

	_, stderr, err := a.execCommand(ctx, pingCommand)
	if err != nil {
		return errkit.Wrap(err, "Error while Pinging the database", "stderr", stderr, "app", a.name)
	}

	log.Print("Ping to the application was success.", field.M{"app": a.name})
	return nil
}

func (a *RDSAuroraMySQLDB) Insert(ctx context.Context) error {
	log.Print("Adding entry to database", field.M{"app": a.name})
	insertQuery := fmt.Sprintf(mysqlConnectionString+
		"\"INSERT INTO pets VALUES ('Puffball', 'Diane', 'hamster', 'f', '1999-03-30', 'NULL');\"", a.host, a.username, a.password, a.dbName)

	insertCommand := []string{"sh", "-c", insertQuery}
	_, stderr, err := a.execCommand(ctx, insertCommand)
	if err != nil {
		return errkit.Wrap(err, "Error while inserting data into table", "stderr", stderr, "app", a.name)
	}
	log.Info().Print("Inserted a row in test db.", field.M{"app": a.name})
	return nil
}

func (a *RDSAuroraMySQLDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting entries from database", field.M{"app": a.name})
	countQuery := fmt.Sprintf(mysqlConnectionString+
		"\"SELECT COUNT(*) FROM pets;\"", a.host, a.username, a.password, a.dbName)

	countCommand := []string{"sh", "-c", countQuery}
	stdout, stderr, err := a.execCommand(ctx, countCommand)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while counting data of table", "stderr", stderr, "app", a.name)
	}

	rowsReturned, err := strconv.Atoi(stdout)
	if err != nil {
		return 0, errkit.Wrap(err, "Error while converting response of count query to int", "stderr", stderr, "app", a.name)
	}

	log.Info().Print("Number of rows in test DB.", field.M{"app": a.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (a *RDSAuroraMySQLDB) Reset(ctx context.Context) error {
	log.Print("Resetting the mysql instance.", field.M{"app": a.name})

	deleteQuery := fmt.Sprintf(mysqlConnectionString+"\"DROP TABLE IF EXISTS pets;\"", a.host, a.username, a.password, a.dbName)
	deleteCommand := []string{"sh", "-c", deleteQuery}
	_, stderr, err := a.execCommand(ctx, deleteCommand)
	if err != nil {
		return errkit.Wrap(err, "Error while deleting data from table", "stderr", stderr, "app", a.name)
	}

	log.Info().Print("Database reset was successful!", field.M{"app": a.name})
	return nil
}

func (a *RDSAuroraMySQLDB) Initialize(ctx context.Context) error {
	log.Print("Initializing database", field.M{"app": a.name})
	createQuery := fmt.Sprintf(mysqlConnectionString+"\"CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);\"", a.host, a.username, a.password, a.dbName)
	createCommand := []string{"sh", "-c", createQuery}
	_, stderr, err := a.execCommand(ctx, createCommand)
	if err != nil {
		return errkit.Wrap(err, "Error while creating the database", "stderr", stderr, "app", a.name)
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
		return errkit.Wrap(err, "Error getting aws config", "app", a.name)
	}
	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return errkit.Wrap(err, "Failed to create rds client. You may need to delete RDS resources manually. app=rds-postgresql")
	}

	descOp, err := rdsCli.DescribeDBClusters(ctx, a.id)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != awsrds.ErrCodeDBClusterNotFoundFault {
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
		return errkit.Wrap(err, "Failed to create ec2 client.")
	}

	log.Info().Print("Deleting db subnet group.", field.M{"app": a.name})
	_, err = rdsCli.DeleteDBSubnetGroup(ctx, a.dbSubnetGroup)
	if err != nil {
		// If the subnet group does not exist, ignore the error and return
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case awsrds.ErrCodeDBSubnetGroupNotFoundFault:
				log.Info().Print("Subnet Group Does not exist: ErrCodeDBSubnetGroupNotFoundFault.", field.M{"app": a.name, "name": a.dbSubnetGroup})
			default:
				return errkit.Wrap(err, "Failed to delete subnet group. You may need to delete it manually.", "app", a.name, "name", a.dbSubnetGroup)
			}
		}
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
				return errkit.Wrap(err, "Failed to delete security group. You may need to delete it manually.", "app", a.name, "name", a.securityGroupName)
			}
		}
	}

	// Remove workload object created for executing commands
	err = a.cli.AppsV1().Deployments(a.namespace).Delete(ctx, a.bastionDebugWorkloadName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Error deleting Workload", "deployment", a.bastionDebugWorkloadName, "app", a.name)
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

func (a RDSAuroraMySQLDB) execCommand(ctx context.Context, command []string) (string, string, error) {
	podName, containerName, err := kube.GetPodContainerFromDeployment(ctx, a.cli, a.namespace, a.bastionDebugWorkloadName)
	if err != nil || podName == "" {
		return "", "", err
	}
	return kube.Exec(ctx, a.cli, a.namespace, podName, containerName, command, nil)
}
