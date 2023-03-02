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
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsrds "github.com/aws/aws-sdk-go/service/rds"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	aws "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/aws/ec2"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"

	// Initialize pq driver
	_ "github.com/lib/pq"
)

type RDSPostgresDB struct {
	name              string
	cli               kubernetes.Interface
	namespace         string
	id                string
	host              string
	databases         []string
	username          string
	password          string
	accessID          string
	secretKey         string
	region            string
	sessionToken      string
	securityGroupID   string
	securityGroupName string
	sqlDB             *sql.DB
	configMapName     string
	secretName        string
	vpcID             string
	subnetGroup       string
	publicAccess      bool
}

const (
	dbInstanceType   = "db.t3.micro"
	connectionString = "PGPASSWORD=%s psql -h %s -p 5432 -U %s -d %s -c"
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
			return fmt.Errorf("env var %s is not set", aws.Region)
		}
	}

	// If sessionToken is set, accessID and secretKey not required
	pdb.sessionToken, ok = os.LookupEnv(aws.SessionToken)
	if ok {
		return nil
	}

	pdb.accessID, ok = os.LookupEnv(aws.AccessKeyID)
	if !ok {
		return fmt.Errorf("env var %s is not set", aws.AccessKeyID)
	}
	pdb.secretKey, ok = os.LookupEnv(aws.SecretAccessKey)
	if !ok {
		return fmt.Errorf("env var %s is not set", aws.SecretAccessKey)
	}
	return nil
}

func (pdb *RDSPostgresDB) SetVpcID(vpcId string) {
	pdb.vpcID = vpcId
}

func (pdb *RDSPostgresDB) Install(ctx context.Context, ns string) error {
	var err error
	pdb.namespace = ns

	// Create AWS config
	awsConfig, region, err := pdb.getAWSConfig(ctx)
	if err != nil {
		return errors.Wrapf(err, "app=%s", pdb.name)
	}
	// Create ec2 client
	// ec2Cli, err := ec2.NewClient(ctx, awsConfig, region)
	// if err != nil {
	// 	return err
	// }

	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return err
	}

	// pdb.vpcID = os.Getenv("VPC_ID")
	// log.Info().Print("VPC_ID from kanister", field.M{"VPC ID": pdb.vpcID})

	// // VPCId is not provided, use Default VPC and subnet group
	// if pdb.vpcID == "" {
	// 	pdb.publicAccess = true
	// 	defaultVpc, err := ec2Cli.DescribeDefaultVpc(ctx)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if len(defaultVpc.Vpcs) == 0 {
	// 		return fmt.Errorf("No default VPC found")
	// 	}
	// 	pdb.vpcID = *defaultVpc.Vpcs[0].VpcId
	// 	fmt.Println(pdb.vpcID)
	// 	pdb.subnetGroup = "default"
	// } else {
	// 	// create a subnetgroup in the VPCID
	// 	resp, err := ec2Cli.DescribeSubnets(ctx, pdb.vpcID)
	// 	if err != nil {
	// 		fmt.Println("Failed to describe subnets", err)
	// 		return err
	// 	}

	// 	// Extract subnet IDs from the response
	// 	var subnetIDs []string
	// 	for _, subnet := range resp.Subnets {
	// 		log.Info().Print("subnet")
	// 		log.Info().Print(*subnet.SubnetId)
	// 		subnetIDs = append(subnetIDs, *subnet.SubnetId)
	// 	}
	// 	subnetGroup, err := rdsCli.CreateDBSubnetGroup(ctx, fmt.Sprintf("%s-subnetgroup", pdb.name), "kanister-test-subnet-group", subnetIDs)
	// 	if err != nil {
	// 		fmt.Println("Failed to create subnet group", err)
	// 		return err
	// 	}
	// 	pdb.subnetGroup = *subnetGroup.DBSubnetGroup.DBSubnetGroupName
	// }

	pdb.subnetGroup = "rds-postgres-snap-subnetgroup"
	// Create security group
	// log.Info().Print("Creating security group.", field.M{"app": pdb.name, "name": pdb.securityGroupName, "vpcID": pdb.vpcID})
	// sg, err := ec2Cli.CreateSecurityGroup(ctx, pdb.securityGroupName, "kanister-test-security-group", pdb.vpcID)
	// if err != nil {
	// 	return err
	// }
	// pdb.securityGroupID = *sg.GroupId

	pdb.securityGroupID = "rds-postgres-snap-sg"
	// Add ingress rule
	// log.Info().Print("Adding ingress rule to security group.", field.M{"app": pdb.name})
	// log.Info().Print("Security Group ID", field.M{"groupID": pdb.securityGroupID}, field.M{"groupName": pdb.securityGroupName})
	// _, err = ec2Cli.AuthorizeSecurityGroupIngress(ctx, pdb.securityGroupID, "0.0.0.0/0", "tcp", 5432)
	// if err != nil {
	// 	return err
	// }

	// Create RDS instance
	// log.Info().Print("Creating RDS instance.", field.M{"app": pdb.name, "id": pdb.id})
	// _, err = rdsCli.CreateDBInstance(ctx, 20, dbInstanceType, pdb.id, "postgres", pdb.subnetGroup, pdb.username, pdb.password, []string{pdb.securityGroupID}, pdb.publicAccess)
	// if err != nil {
	// 	return err
	// }

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
	dbconfig := &v1.ConfigMap{
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
	dbsecret := &v1.Secret{
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

	testPodyaml := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "postgres",
					Image:   "postgres",
					Command: []string{"sleep", "infinity"},
				},
			},
		},
	}
	pod, err := pdb.cli.CoreV1().Pods(pdb.namespace).Create(context.Background(), testPodyaml, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "Failed while creating for Pod to be created")
	}

	if err := kube.WaitForPodReady(ctx, pdb.cli, pod.Namespace, pod.Name); err != nil {
		return errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
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
	// Get connection info from configmap

	log.Info().Print("Pinging")

	dbconfig, err := pdb.cli.CoreV1().ConfigMaps(pdb.namespace).Get(ctx, pdb.configMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Info().Print("dbconfig")
	// Get secret creds
	dbsecret, err := pdb.cli.CoreV1().Secrets(pdb.namespace).Get(ctx, pdb.secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Info().Print("get secret")
	// Parse databases from config data
	var databases []string
	if err := yaml.Unmarshal([]byte(dbconfig.Data["postgres.databases"]), &databases); err != nil {
		return err
	}
	if databases == nil {
		return errors.New("Databases are missing from configmap")
	}

	log.Print("Pinging rds postgres database", field.M{"app": pdb.name})
	isReadyCommand := fmt.Sprintf(connectionString+"'SELECT version();'", dbsecret.Data["password"], dbconfig.Data["postgres.host"], dbconfig.Data["postgres.user"], databases[0])

	pingCommand := []string{"sh", "-c", isReadyCommand}

	log.Print("pinging command ", field.M{"isReadyCommad": isReadyCommand}, field.M{"pingCommand": pingCommand})
	_, stderr, err := pdb.execCommand(ctx, "test-pod", pingCommand)
	if err != nil {
		return errors.Wrapf(err, "Error while Pinging the database: %s", stderr)
	}
	log.Print("Ping to the application was success.", field.M{"app": pdb.name})
	return nil
}

func (pdb RDSPostgresDB) Insert(ctx context.Context) error {
	log.Print("Adding entry to database", field.M{"app": pdb.name})
	log.Info().Print("Insert")
	now := time.Now().Format(time.RFC3339Nano)
	insert := fmt.Sprintf(connectionString+
		"\"INSERT INTO inventory (name) VALUES (\"%s\");\"", pdb.password, pdb.host, pdb.username, pdb.databases[0], now)

	log.Info().Print(insert)
	insertQuery := []string{"sh", "-c", insert}
	_, stderr, err := pdb.execCommand(ctx, "test-pod", insertQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while inserting data into table: %s", stderr)
	}
	return nil
}

func (pdb RDSPostgresDB) Count(ctx context.Context) (int, error) {
	log.Print("Counting entries from database", field.M{"app": pdb.name})
	count := fmt.Sprintf(connectionString+
		"\"SELECT COUNT(*) FROM Inventory\" -h -1", pdb.password, pdb.host, pdb.username, pdb.databases[0])

	countQuery := []string{"sh", "-c", count}
	stdout, stderr, err := pdb.execCommand(ctx, "test-pod", countQuery)
	if err != nil {
		return 0, errors.Wrapf(err, "Error while counting data into table: %s", stderr)
	}
	log.Info().Print("count result")
	log.Info().Print(stdout)
	rowsReturned, err := strconv.Atoi(strings.TrimSpace(strings.Split(stdout, "\n")[1]))
	if err != nil {
		return 0, errors.Wrapf(err, "Error while converting response of count query: %s", stderr)
	}
	log.Info().Print("Counting rows in test db.", field.M{"app": pdb.name, "count": rowsReturned})
	return rowsReturned, nil
}

func (pdb RDSPostgresDB) Reset(ctx context.Context) error {
	log.Print("Reseting database", field.M{"app": pdb.name})
	delete := fmt.Sprintf(connectionString+"\"DROP TABLE IF EXISTS inventory;\"", pdb.password, pdb.host, pdb.username, pdb.databases[0])
	deleteQuery := []string{"sh", "-c", delete}
	_, stderr, err := pdb.execCommand(ctx, "test-pod", deleteQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while deleting data into table: %s", stderr)
	}
	log.Info().Print("Database reset successful!", field.M{"app": pdb.name})
	return nil
}

// Initialize is used to initialize the database or create schema
func (pdb RDSPostgresDB) Initialize(ctx context.Context) error {
	// Create table.
	log.Print("Initializing database", field.M{"app": pdb.name})
	createTable := fmt.Sprintf(connectionString+"\"CREATE TABLE inventory (id serial PRIMARY KEY, name VARCHAR(50));\"", pdb.password, pdb.host, pdb.username, pdb.databases[0])
	log.Info().Print("create Table command")
	log.Info().Print(createTable)
	execQuery := []string{"sh", "-c", createTable}
	_, stderr, err := pdb.execCommand(ctx, "test-pod", execQuery)
	if err != nil {
		return errors.Wrapf(err, "Error while creating the database: %s", stderr)
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
		return errors.Wrapf(err, "app=%s", pdb.name)
	}
	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return errors.Wrap(err, "Failed to create rds client. You may need to delete RDS resources manually. app=rds-postgresql")
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
				return errors.Wrapf(err, "Failed to delete rds instance. You may need to delete it manually. app=rds-postgresql id=%s", pdb.id)
			}
		}
	}

	// Waiting for rds to be deleted
	if err == nil {
		log.Info().Print("Waiting for rds to be deleted", field.M{"app": pdb.name})
		err = rdsCli.WaitUntilDBInstanceDeleted(ctx, pdb.id)
		if err != nil {
			return errors.Wrapf(err, "Failed to wait for rds instance till delete succeeds. app=rds-postgresql id=%s", pdb.id)
		}
	}

	// Create ec2 client
	ec2Cli, err := ec2.NewClient(ctx, awsConfig, region)
	if err != nil {
		return errors.Wrap(err, "Failed to ec2 client. You may need to delete EC2 resources manually. app=rds-postgresql")
	}

	// Delete security group
	log.Info().Print("Deleting security group.", field.M{"app": pdb.name})
	_, err = ec2Cli.DeleteSecurityGroup(ctx, pdb.securityGroupName)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case "InvalidGroup.NotFound":
				log.Error().Print("Security group already deleted: InvalidGroup.NotFound.", field.M{"app": pdb.name, "name": pdb.securityGroupName})
			default:
				return errors.Wrapf(err, "Failed to delete security group. You may need to delete it manually. app=rds-postgresql name=%s", pdb.securityGroupName)
			}
		}
	}

	// Delete subnetGroup
	log.Info().Print("Deleting db subnet group.", field.M{"app": pdb.name})
	if pdb.subnetGroup != "default" {
		log.Info().Print("subnet group is not default deleting it")
		_, err = rdsCli.DeleteDBSubnetGroup(ctx, pdb.subnetGroup)
		if err != nil {
			// If the subnet group does not exist, ignore the error and return
			if err, ok := err.(awserr.Error); ok {
				switch err.Code() {
				case awsrds.ErrCodeDBSubnetGroupNotFoundFault:
					log.Info().Print("Subnet Group Does not exist: ErrCodeDBSubnetGroupNotFoundFault.", field.M{"app": pdb.name, "id": pdb.id})
				default:
					return errors.Wrapf(err, "Failed to delete subnet group. You may need to delete it manually. app=rds-postgresql id=%s", pdb.id)
				}
			}
		}
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

func (pdb RDSPostgresDB) execCommand(ctx context.Context, podName string, command []string) (string, string, error) {
	pod, err := pdb.cli.CoreV1().Pods(pdb.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", "", errors.Wrapf(err, "Error getting pod and container name for app %s.", pdb.name)
	}
	return kube.Exec(pdb.cli, pdb.namespace, podName, pod.Spec.Containers[0].Name, command, nil)
}
