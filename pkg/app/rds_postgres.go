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
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	awsrds "github.com/aws/aws-sdk-go/service/rds"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/config/aws"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/testutil"

	// Initialize pq driver
	_ "github.com/lib/pq"
)

type RDSPostgresDB struct {
	cli             kubernetes.Interface
	namespace       string
	id              string
	host            string
	dbname          string
	username        string
	password        string
	accessID        string
	secretKey       string
	region          string
	sessionToken    string
	securityGroupID string
	sqlDB           *sql.DB
}

type RDSPostgresBP struct {
	name         string
	appNamespace string
}

func NewRDSPostgresDB() App {
	return &RDSPostgresDB{
		id:       "test-postgresql-instance",
		dbname:   "postgres",
		username: "master",
		password: "secret99",
	}
}

func (pdb *RDSPostgresDB) Init(ctx context.Context) error {
	var ok bool
	pdb.region, ok = os.LookupEnv(awsconfig.Region)
	if !ok {
		return fmt.Errorf("Env var %s is not set", awsconfig.Region)
	}

	// If sessionToken is set, accessID and secretKey not required
	pdb.sessionToken, ok = os.LookupEnv(awsconfig.SessionToken)
	if ok {
		return nil
	}

	pdb.accessID, ok = os.LookupEnv(awsconfig.AccessKeyID)
	if !ok {
		return fmt.Errorf("Env var %s is not set", awsconfig.AccessKeyID)
	}
	pdb.secretKey, ok = os.LookupEnv(awsconfig.SecretAccessKey)
	if !ok {
		return fmt.Errorf("Env var %s is not set", awsconfig.SecretAccessKey)
	}

	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	if err != nil {
		return nil
	}
	pdb.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	return nil
}

func (pdb *RDSPostgresDB) Install(ctx context.Context, ns string) error {
	var err error
	pdb.namespace = ns

	// Create ec2 client
	ec2, err := testutil.NewEC2Client(ctx, pdb.accessID, pdb.secretKey, pdb.region, pdb.sessionToken, "")
	if err != nil {
		return err
	}

	// Create security group
	log.Info().Print("Creating security group.", field.M{"app": "rds-postgresql", "name": "pgtest-sg"})
	sg, err := ec2.CreateSecurityGroup(ctx, "pgtest-sg", "pgtest-security-group")
	if err != nil {
		return err
	}
	pdb.securityGroupID = *sg.GroupId

	// Add ingress rule
	log.Info().Print("Adding ingress rule to security group.", field.M{"app": "rds-postgresql"})
	_, err = ec2.AuthorizeSecurityGroupIngress(ctx, "pgtest-sg", "0.0.0.0/0", "tcp", 5432)
	if err != nil {
		return err
	}

	// Create rds client
	rds, err := testutil.NewRDSClient(ctx, pdb.accessID, pdb.secretKey, pdb.region, pdb.sessionToken, "")
	if err != nil {
		return err
	}

	// Create RDS instance
	log.Info().Print("Creating RDS instance.", field.M{"app": "rds-postgresql", "id": pdb.id})
	_, err = rds.CreateDBInstance(ctx, 20, "db.t2.micro", pdb.id, "postgres", pdb.username, pdb.password, pdb.securityGroupID)
	if err != nil {
		return err
	}

	// Wait for DB to be ready
	log.Info().Print("Waiting for rds to be ready.", field.M{"app": "rds-postgresql"})
	err = rds.WaitUntilDBInstanceAvailable(ctx, pdb.id)
	if err != nil {
		return err
	}

	// Find host of the instance
	dbInstance, err := rds.DescribeDBInstances(ctx, pdb.id)
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
			Name: "dbconfig",
		},
		Data: map[string]string{
			"postgres.instanceid": pdb.id,
			"postgres.host":       pdb.host,
			"postgres.database":   pdb.dbname,
			"postgres.user":       pdb.username,
		},
	}
	_, err = pdb.cli.CoreV1().ConfigMaps(ns).Create(dbconfig)
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
			Name: "dbsecret",
		},
		StringData: map[string]string{
			"password":          pdb.password,
			"access_key_id":     pdb.accessID,
			"secret_access_key": pdb.secretKey,
			"aws_region":        pdb.region,
		},
	}
	_, err = pdb.cli.CoreV1().Secrets(ns).Create(dbsecret)
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
		Kind:      "namespace",
		Name:      pdb.namespace,
		Namespace: pdb.namespace,
	}
}

// Ping makes and tests DB connection
func (pdb *RDSPostgresDB) Ping(ctx context.Context) error {
	// Get connection info from configmap
	dbconfig, err := pdb.cli.CoreV1().ConfigMaps(pdb.namespace).Get("dbconfig", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Get secret creds
	dbsecret, err := pdb.cli.CoreV1().Secrets(pdb.namespace).Get("dbsecret", metav1.GetOptions{})
	if err != nil {
		return err
	}

	var connectionString string = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", dbconfig.Data["postgres.host"], dbconfig.Data["postgres.user"], dbsecret.Data["password"], dbconfig.Data["postgres.database"])

	// Initialize connection object.
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		return err
	}

	pdb.sqlDB = db
	log.Info().Print("Connected to database.", field.M{"app": "rds-postgresql"})
	return nil
}

func (pdb RDSPostgresDB) Insert(ctx context.Context, n int) error {
	for i := 0; i < n; i++ {
		now := time.Now().Format(time.RFC3339Nano)
		stmt := "INSERT INTO inventory (name) VALUES ($1);"
		_, err := pdb.sqlDB.Exec(stmt, now)
		if err != nil {
			return err
		}
		log.Info().Print("Inserted a row in test db.", field.M{"app": "rds-postgresql"})
	}
	return nil
}

func (pdb RDSPostgresDB) Count(ctx context.Context) (int, error) {
	stmt := "SELECT COUNT(*) FROM inventory;"
	row := pdb.sqlDB.QueryRow(stmt)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	log.Info().Print("Counting rows in test db.", field.M{"app": "rds-postgresql", "count": count})
	return count, nil
}

func (pdb RDSPostgresDB) Reset(ctx context.Context) error {
	_, err := pdb.sqlDB.Exec("DROP TABLE IF EXISTS inventory;")
	if err != nil {
		return err
	}

	// Create table.
	_, err = pdb.sqlDB.Exec("CREATE TABLE inventory (id serial PRIMARY KEY, name VARCHAR(50));")
	if err != nil {
		return err
	}

	log.Info().Print("Database reset successful!", field.M{"app": "rds-postgresql"})
	return nil
}

func (pdb RDSPostgresDB) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"dbconfig": crv1alpha1.ObjectReference{
			Kind:      "configmap",
			Name:      "dbconfig",
			Namespace: pdb.namespace,
		},
	}
}

func (pdb RDSPostgresDB) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"dbsecret": crv1alpha1.ObjectReference{
			Kind:      "secret",
			Name:      "dbsecret",
			Namespace: pdb.namespace,
		},
	}
}

func (pdb RDSPostgresDB) Uninstall(ctx context.Context) error {
	// Create rds client
	rds, err := testutil.NewRDSClient(ctx, pdb.accessID, pdb.secretKey, pdb.region, pdb.sessionToken, "")
	if err != nil {
		log.Error().WithError(err).Print("Failed to create rds client. You may need to delete RDS resources manually.", field.M{"app": "rds-postgresql"})
		return err
	}

	// Delete rds instance
	log.Info().Print("Deleting rds instance", field.M{"app": "rds-postgresql"})
	_, err = rds.DeleteDBInstance(ctx, pdb.id)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case awsrds.ErrCodeDBInstanceNotFoundFault:
				log.Info().Print("Rds instance already deleted: ErrCodeDBInstanceNotFoundFault.", field.M{"app": "rds-postgresql", "id": pdb.id})
			default:
				log.Error().WithError(err).Print("Failed to delete rds instance. You may need to delete it manually.", field.M{"app": "rds-postgresql", "id": pdb.id})
				return err
			}
		}
	}

	// Waiting for rds to be deleted
	if err == nil {
		log.Info().Print("Waiting for rds to be deleted", field.M{"app": "rds-postgresql"})
		err = rds.WaitUntilDBInstanceDeleted(ctx, pdb.id)
		if err != nil {
			log.Error().Print("Failed to wait for rds instance till delete succeeds.", field.M{"app": "rds-postgresql", "id": pdb.id})
			return err
		}
	}

	// Create ec2 client
	ec2, err := testutil.NewEC2Client(ctx, pdb.accessID, pdb.secretKey, pdb.region, pdb.sessionToken, "")
	if err != nil {
		log.Error().WithError(err).Print("Failed to ec2 client. You may need to delete EC2 resources manually.", field.M{"app": "rds-postgresql"})
		return err
	}

	// Delete security group
	log.Info().Print("Deleting security group.", field.M{"app": "rds-postgresql"})
	_, err = ec2.DeleteSecurityGroup(ctx, "pgtest-sg")
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case "InvalidGroup.NotFound":
				log.Error().Print("Security group pgtest-sg already deleted: InvalidGroup.NotFound.", field.M{"app": "rds-postgresql"})
			default:
				log.Error().WithError(err).Print("Failed to delete security group. You may need to delete it manually.", field.M{"app": "rds-postgresql", "name": "pgtest-sg"})
				return err
			}
		}
	}
	return nil
}
