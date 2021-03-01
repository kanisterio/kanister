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

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	rdserr "github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
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
	"github.com/kanisterio/kanister/pkg/poll"

	_ "github.com/go-sql-driver/mysql"
)

const (
	AuroraDBInstanceClass = "db.r5.large"
	AuroraDBStorage       = 20
	DetailsCMName         = "dbconfig"
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
			return fmt.Errorf("Env var %s is not set", aws.Region)
		}
	}

	// If sessionToken is set, accessID and secretKey not required
	a.sessionToken, ok = os.LookupEnv(aws.SessionToken)
	if ok {
		return nil
	}

	a.accessID, ok = os.LookupEnv(aws.AccessKeyID)
	if !ok {
		return fmt.Errorf("Env var %s is not set", aws.AccessKeyID)
	}
	a.secretKey, ok = os.LookupEnv(aws.SecretAccessKey)
	if !ok {
		return fmt.Errorf("Env var %s is not set", aws.SecretAccessKey)
	}

	return nil
}

func (a *RDSAuroraMySQLDB) Install(ctx context.Context, namespace string) error {
	a.namespace = namespace

	// Get aws config
	awsConfig, region, err := a.getAWSConfig(ctx)
	fmt.Printf("awsConfig %+v\n", *awsConfig.Credentials)
	if err != nil {
		return errors.Wrapf(err, "Error getting aws config app=%s", a.name)
	}

	// Create ec2 client
	ec2Cli, err := ec2.NewClient(ctx, awsConfig, region)
	if err != nil {
		return err
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
	if err != nil {
		return err
	}

	return nil
}

func (a *RDSAuroraMySQLDB) IsReady(context.Context) (bool, error) {
	// we are alrady waiting for dbcluster using WaitUntilDBClusterAvailable while installing it
	return true, nil
}

func (a *RDSAuroraMySQLDB) Ping(context.Context) error {
	db, err := a.openDBConnection()
	if err != nil {
		return errors.Wrap(err, "Error opening database connectio")
	}
	defer a.closeDBConnection(db)

	pingQuery := "select 1"
	_, err = db.Query(pingQuery)
	if err != nil {
		return err
	}

	return nil
}

func (a *RDSAuroraMySQLDB) Insert(ctx context.Context) error {
	db, err := a.openDBConnection()
	if err != nil {
		return err
	}
	defer a.closeDBConnection(db)

	query, err := db.Prepare("INSERT INTO pets VALUES (?,?,?,?,?,?);")
	if err != nil {
		return errors.Wrap(err, "Error preparing query")
	}

	// start a transaction
	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "Error begining transaction")
	}

	_, err = tx.Stmt(query).Exec("Puffball", "Diane", "hamster", "f", "1999-03-30", "NULL")
	if err != nil {
		return errors.Wrap(err, "Error inserting data into Aurora DB cluster")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "Error commiting data into Aurora DB database")
	}

	return nil
}

func (a *RDSAuroraMySQLDB) Count(context.Context) (int, error) {
	db, err := a.openDBConnection()
	if err != nil {
		return 0, err
	}
	defer a.closeDBConnection(db)

	rows, err := db.Query("select * from pets;")
	if err != nil {
		return 0, errors.Wrap(err, "Error preparing count query")
	}
	count := 0
	for rows.Next() {
		count++
	}

	return count, nil
}

func (a *RDSAuroraMySQLDB) Reset(ctx context.Context) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, mysqlWaitTimeout)
	defer waitCancel()
	err := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		err := a.Ping(ctx)
		return err == nil, nil
	})

	if err != nil {
		return errors.Wrapf(err, "Error waiting for application %s to be ready to reset it", a.name)
	}

	log.Print("Resetting the mysql instance.", field.M{"app": a.name})

	db, err := a.openDBConnection()
	if err != nil {
		return err
	}
	defer a.closeDBConnection(db)

	query, err := db.Prepare(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", a.dbName))
	if err != nil {
		return errors.Wrap(err, "Error preparing reset query")
	}

	// start a transaction
	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "Error begining transaction")
	}

	_, err = tx.Stmt(query).Exec()
	if err != nil {
		return errors.Wrap(err, "Error resetting Aurora database")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "Error commiting DB reset transaction")
	}

	return nil
}

func (a *RDSAuroraMySQLDB) Initialize(context.Context) error {
	db, err := a.openDBConnection()
	if err != nil {
		return err
	}
	// TODO @viveksinghggits handle this error
	defer a.closeDBConnection(db)

	query, err := db.Prepare("CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);")
	if err != nil {
		return errors.Wrap(err, "Error preparing query")
	}

	// start a transaction
	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "Error begining transaction")
	}

	_, err = tx.Stmt(query).Exec()
	if err != nil {
		return errors.Wrap(err, "Error creating table into Aurora database")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "Error commiting table creation")
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
		return errors.Wrap(err, "Failed to ec2 client. You may need to delete EC2 resources manually. app=rds-postgresql")
	}

	// delete security group
	log.Info().Print("Deleting security group.", field.M{"app": a.name})
	_, err = ec2Cli.DeleteSecurityGroup(ctx, a.securityGroupName)
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
