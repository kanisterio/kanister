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
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/config/aws"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/testutil"

	// Initialize pq driver
	_ "github.com/lib/pq"
)

type PostgresDB struct {
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

type PostgresBP struct {
	name         string
	appNamespace string
}

func NewPostgresDB() App {
	return &PostgresDB{
		id:       "test-postgresql-instance",
		dbname:   "postgres",
		username: "master",
		password: "secret99",
	}
}

func (pdb *PostgresDB) Init(ctx context.Context) error {
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

func (pdb *PostgresDB) Install(ctx context.Context, ns string) error {
	var err error
	pdb.namespace = ns

	// Create ec2 client
	ec2, err := testutil.NewEC2Client(ctx, pdb.accessID, pdb.secretKey, pdb.region, pdb.sessionToken, "")
	if err != nil {
		return err
	}

	// Create security group
	log.Info("PostgresDB: creating security group")
	sg, err := ec2.CreateSecurityGroup(ctx, "pgtest-sg", "pgtest-security-group")
	if err != nil {
		return err
	}
	pdb.securityGroupID = *sg.GroupId

	// Add ingress rule
	log.Info("PostgresDB: adding ingress rule to security group")
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
	log.Info("PostgresDB: creating rds instance")
	_, err = rds.CreateDBInstance(ctx, 20, "db.t2.micro", pdb.id, "postgres", pdb.username, pdb.password, pdb.securityGroupID)
	if err != nil {
		return err
	}

	// Wait for DB to be ready
	log.Info("PostgresDB: Waiting for rds to be ready")
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

func (pdb *PostgresDB) IsReady(ctx context.Context) (bool, error) {
	return true, nil
}

func (pdb *PostgresDB) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "namespace",
		Name:      pdb.namespace,
		Namespace: pdb.namespace,
	}
}

// Ping makes and tests DB connection
func (pdb *PostgresDB) Ping(ctx context.Context) error {
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
	log.Info("Successfully created connection to database")
	return nil
}

func (pdb PostgresDB) Insert(ctx context.Context, n int) error {
	for i := 0; i < n; i++ {
		now := time.Now().Format(time.RFC3339Nano)
		stmt := "INSERT INTO inventory (name) VALUES ($1);"
		_, err := pdb.sqlDB.Exec(stmt, now)
		if err != nil {
			return err
		}
		log.Info("Inserted a row")
	}
	return nil
}

func (pdb PostgresDB) Count(ctx context.Context) (int, error) {
	stmt := "SELECT COUNT(*) FROM inventory;"
	row := pdb.sqlDB.QueryRow(stmt)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	log.Infof("Found %d rows\n", count)
	return count, nil
}

func (pdb PostgresDB) Reset(ctx context.Context) error {
	_, err := pdb.sqlDB.Exec("DROP TABLE IF EXISTS inventory;")
	if err != nil {
		return err
	}
	log.Info("Finished dropping table (if existed)")

	// Create table.
	_, err = pdb.sqlDB.Exec("CREATE TABLE inventory (id serial PRIMARY KEY, name VARCHAR(50));")
	if err != nil {
		return err
	}
	log.Info("Finished creating table")
	return nil
}

func (pdb PostgresDB) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"dbconfig": crv1alpha1.ObjectReference{
			Kind:      "configmap",
			Name:      "dbconfig",
			Namespace: pdb.namespace,
		},
	}
}

func (pdb PostgresDB) Secrets() map[string]crv1alpha1.ObjectReference {
	return map[string]crv1alpha1.ObjectReference{
		"dbsecret": crv1alpha1.ObjectReference{
			Kind:      "secret",
			Name:      "dbsecret",
			Namespace: pdb.namespace,
		},
	}
}

func (pdb PostgresDB) Uninstall(ctx context.Context) error {
	// Create rds client
	rds, err := testutil.NewRDSClient(ctx, pdb.accessID, pdb.secretKey, pdb.region, pdb.sessionToken, "")
	if err != nil {
		log.Errorf("Failed to create rds client: %s. You may need to delete RDS resources manually", err.Error())
		return err
	}

	// Delete rds instance
	log.Info("PostgresDB: deleting rds instance")
	_, err = rds.DeleteDBInstance(ctx, pdb.id)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case awsrds.ErrCodeDBInstanceNotFoundFault:
				log.Infof("Rds instance %s already deleted: ErrCodeDBInstanceNotFoundFault", pdb.id)
			default:
				log.Errorf("Failed to delete rds instance %s: %s. You may need to delete it manually", pdb.id, err.Error())
				return err
			}
		}
	}

	// Waiting for rds to be deleted
	if err == nil {
		log.Info("PostgresDB: Waiting for rds to be deleted")
		err = rds.WaitUntilDBInstanceDeleted(ctx, pdb.id)
		if err != nil {
			log.Errorf("Failed to wait for rds instance %s till delete succeeds: %s", pdb.id, err.Error())
			return err
		}
	}

	// Create ec2 client
	ec2, err := testutil.NewEC2Client(ctx, pdb.accessID, pdb.secretKey, pdb.region, pdb.sessionToken, "")
	if err != nil {
		log.Errorf("Failed to ec2 rds client: %s. You may need to delete EC2 resources manually", err.Error())
		return err
	}

	// Delete security group
	log.Info("PostgresDB: deleting security group")
	_, err = ec2.DeleteSecurityGroup(ctx, "pgtest-sg")
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			switch err.Code() {
			case "InvalidGroup.NotFound":
				log.Errorf("Security group pgtest-sg already deleted: InvalidGroup.NotFound")
			default:
				log.Errorf("Failed to delete security group pgtest-sg: %s. You may need to delete it manually", err.Error())
				return err
			}
		}
	}
	return nil
}

func NewPostgresBP() Blueprinter {
	return PostgresBP{}
}

func (pbp PostgresBP) Blueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			Name: "blueprint",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": &crv1alpha1.BlueprintAction{
				Kind: "Namespace",
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"snapshot": crv1alpha1.Artifact{
						KeyValue: map[string]string{
							"id":   "{{ .Namespace.Name }}-{{ toDate \"2006-01-02T15:04:05.999999999Z07:00\" .Time  | date \"2006-01-02T15-04-05\" }}",
							"sgid": "{{ .Phases.backupSnapshot.Output.securityGroupID }}",
						},
					},
				},
				ConfigMapNames: []string{"dbconfig"},
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Func: "KubeTask",
						Name: "backupSnapshot",
						Args: map[string]interface{}{
							"namespace": "rds-postgres-test",
							"image":     "kanisterio/postgres-kanister-tools:0.21.0",
							"command": []string{
								"bash",
								"-o",
								"errexit",
								"-o",
								"pipefail",
								"-o",
								"nounset",
								"-o",
								"xtrace",
								"-c",
								"set +o xtrace\n" +
									"export AWS_SECRET_ACCESS_KEY=\"{{ .Profile.Credential.KeyPair.Secret }}\"\n" +
									"export AWS_ACCESS_KEY_ID=\"{{ .Profile.Credential.KeyPair.ID }}\"\n" +
									"set -o xtrace\n" +
									"aws rds create-db-snapshot --db-instance-identifier=\"{{ index .ConfigMaps.dbconfig.Data \"postgres.instanceid\" }}\" --db-snapshot-identifier=\"{{ .Namespace.Name }}-{{ toDate \"2006-01-02T15:04:05.999999999Z07:00\" .Time  | date \"2006-01-02T15-04-05\" }}\" --region \"{{ .Profile.Location.Region }}\"\n" +
									"aws rds wait db-snapshot-completed --region \"{{ .Profile.Location.Region }}\" --db-snapshot-identifier=\"{{ .Namespace.Name }}-{{ toDate \"2006-01-02T15:04:05.999999999Z07:00\" .Time  | date \"2006-01-02T15-04-05\" }}\" \n" +
									"\n" +
									"vpcsgid=$(aws rds describe-db-instances --db-instance-identifier=\"{{ index .ConfigMaps.dbconfig.Data \"postgres.instanceid\" }}\" --region \"{{ .Profile.Location.Region }}\" --query 'DBInstances[].VpcSecurityGroups[].VpcSecurityGroupId' --output text)\n" +
									"kando output securityGroupID $vpcsgid\n",
							},
						},
					},
				},
			},

			"restore": &crv1alpha1.BlueprintAction{
				Kind:               "Namespace",
				InputArtifactNames: []string{"snapshot"},
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Func: "KubeTask",
						Name: "restoreSnapshot",
						Args: map[string]interface{}{
							"namespace": "rds-postgres-test",
							"image":     "kanisterio/postgres-kanister-tools:0.21.0",
							"command": []string{
								"bash",
								"-o",
								"errexit",
								"-o",
								"nounset",
								"-o",
								"xtrace",
								"-c",
								"set +o xtrace\n" +
									"export AWS_SECRET_ACCESS_KEY=\"{{ .Profile.Credential.KeyPair.Secret }}\"\n" +
									"export AWS_ACCESS_KEY_ID=\"{{ .Profile.Credential.KeyPair.ID }}\"\n" +
									"set -o xtrace\n" +
									"\n" +
									"# Delete old db instance\n" +
									"aws rds delete-db-instance --db-instance-identifier=\"{{ index .ConfigMaps.dbconfig.Data \"postgres.instanceid\" }}\" --skip-final-snapshot --region \"{{ .Profile.Location.Region }}\"\n" +
									"\n" +
									"aws rds wait db-instance-deleted --region \"{{ .Profile.Location.Region }}\" --db-instance-identifier=\"{{ index .ConfigMaps.dbconfig.Data \"postgres.instanceid\" }}\"\n" +
									"\n" +
									"# Restore instance from snapshot\n" +
									"aws rds restore-db-instance-from-db-snapshot --db-instance-identifier=\"{{ index .ConfigMaps.dbconfig.Data \"postgres.instanceid\" }}\" --db-snapshot-identifier=\"{{ .ArtifactsIn.snapshot.KeyValue.id }}\" --vpc-security-group-ids \"{{ .ArtifactsIn.snapshot.KeyValue.sgid }}\" --region \"{{ .Profile.Location.Region }}\"\n" +
									"aws rds wait db-instance-available --region \"{{ .Profile.Location.Region }}\" --db-instance-identifier=\"{{ index .ConfigMaps.dbconfig.Data \"postgres.instanceid\" }}\"\n",
							},
						},
					},
				},
			},

			"delete": &crv1alpha1.BlueprintAction{
				Kind:               "Namespace",
				InputArtifactNames: []string{"snapshot"},
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Func: "KubeTask",
						Name: "deleteSnapshot",
						Args: map[string]interface{}{
							"namespace": "rds-postgres-test",
							"image":     "kanisterio/postgres-kanister-tools:0.21.0",
							"command": []string{
								"bash",
								"-o",
								"errexit",
								"-o",
								"nounset",
								"-o",
								"xtrace",
								"-c",
								"set +o xtrace\n" +
									"export AWS_SECRET_ACCESS_KEY=\"{{ .Profile.Credential.KeyPair.Secret }}\"\n" +
									"export AWS_ACCESS_KEY_ID=\"{{ .Profile.Credential.KeyPair.ID }}\"\n" +
									"set -o xtrace\n" +
									"aws rds delete-db-snapshot --db-snapshot-identifier=\"{{ .ArtifactsIn.snapshot.KeyValue.id }}\" --region \"{{ .Profile.Location.Region }}\"\n",
							},
						},
					},
				},
			},
		},
	}
}
