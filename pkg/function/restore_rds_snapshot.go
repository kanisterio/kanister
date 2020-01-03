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

package function

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	rdserr "github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	_ = kanister.Register(&restoreRDSSnapshotFunc{})
}

var (
	_ kanister.Func = (*restoreRDSSnapshotFunc)(nil)
)

const (
	// RestoreRDSSnapshotFuncName will store the name of the function
	RestoreRDSSnapshotFuncName = "RestoreRDSSnapshot"
	// RestoreRDSSnapshotDBEngine is type that will store which db we are dealing with
	RestoreRDSSnapshotDBEngine RDSDBEngine = "dbEngine"
	// RestoreRDSSnapshotInstanceID is ID of the target instance
	RestoreRDSSnapshotInstanceID = "instanceID"
	// RestoreRDSSnapshotBackupArtifactPrefix stores the prefix of backup in object storage
	RestoreRDSSnapshotBackupArtifactPrefix = "backupArtifactPrefix"
	// RestoreRDSSnapshotBackupID stores the ID of backup in object storage
	RestoreRDSSnapshotBackupID = "backupID"
	// RestoreRDSSnapshotSecurityGroupID stores the securitygroup
	RestoreRDSSnapshotSecurityGroupID = "securityGroupID"
	// RestoreRDSSnapshotSnapshotID stores the snapshot ID
	RestoreRDSSnapshotSnapshotID = "snapshotID"

	// RestoreRDSSnapshotUsername stores username of the database
	RestoreRDSSnapshotUsername = "username"
	// RestoreRDSSnapshotPassword stores the password of the database
	RestoreRDSSnapshotPassword = "password"

	// PostgreSQLEngine stores the postgres appname
	PostgreSQLEngine RDSDBEngine = "PostgreSQL"

	// PostgresToolsImage is the image that has tools to take backup and restore of rds postgres instance
	PostgresToolsImage = "kanisterio/postgres-kanister-tools:0.22.1"
)

type restoreRDSSnapshotFunc struct{}

func (*restoreRDSSnapshotFunc) Name() string {
	return RestoreRDSSnapshotFuncName
}

func (*restoreRDSSnapshotFunc) RequiredArgs() []string {
	return []string{RestoreRDSSnapshotInstanceID, RestoreRDSSnapshotSecurityGroupID, string(RestoreRDSSnapshotDBEngine),
		RestoreRDSSnapshotUsername, RestoreRDSSnapshotPassword}
}

func (*restoreRDSSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var instanceID, snapshotID, securityGroupID, backupArtifactPrefix, backupID, username, password, dbEngine string

	if err := Arg(args, RestoreRDSSnapshotInstanceID, &instanceID); err != nil {
		return nil, err
	}

	if err := OptArg(args, RestoreRDSSnapshotSnapshotID, &snapshotID, ""); err != nil {
		return nil, err
	}

	if snapshotID == "" {
		// snapshot ID is not provided get backupPrefix and backupID
		if err := Arg(args, RestoreRDSSnapshotBackupArtifactPrefix, &backupArtifactPrefix); err != nil {
			return nil, err
		}

		if err := Arg(args, RestoreRDSSnapshotBackupID, &backupID); err != nil {
			return nil, err
		}

	}

	if err := Arg(args, RestoreRDSSnapshotSecurityGroupID, &securityGroupID); err != nil {
		return nil, err
	}

	if err := Arg(args, string(RestoreRDSSnapshotDBEngine), &dbEngine); err != nil {
		return nil, err
	}

	if err := Arg(args, RestoreRDSSnapshotUsername, &username); err != nil {
		return nil, err
	}
	if err := Arg(args, RestoreRDSSnapshotPassword, &password); err != nil {
		return nil, err
	}

	return restoreRDSSnapshot(ctx, instanceID, snapshotID, securityGroupID, backupArtifactPrefix, backupID, username, password, dbEngine, tp.Profile)
}

func restoreRDSSnapshot(ctx context.Context, instanceID, snapshotID, securityGroupID, backupArtifactPrefix, backupID, username, password, dbEngine string, profile *param.Profile) (map[string]interface{}, error) {
	// Validate profile
	if err := ValidateProfile(profile); err != nil {
		return nil, errors.Wrapf(err, "error validating profile")
	}

	awsConfig, region, err := getAWSConfigFromProfile(ctx, profile)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting awsconfig from profile")
	}

	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting rds client from awsconfig")
	}

	if snapshotID != "" {
		// TODO: if the instance already exists
		// delete the db instance
		_, err = rdsCli.DeleteDBInstance(ctx, instanceID)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() != rdserr.ErrCodeDBInstanceNotFoundFault {
					return nil, err
				}
				log.Print("RDS instance is not present ErrCodeDBInstanceNotFoundFault", field.M{"instanceID": instanceID})
			}
		} else {
			// wait for the instance to be deleted
			err = rdsCli.WaitUntilDBInstanceDeleted(ctx, instanceID)
			if err != nil {
				return nil, errors.Wrapf(err, "error waiting for the dbinstance to be available")
			}
		}

		log.Print("restoring database from snapshot", field.M{"instanceID": instanceID})
		// restore from snapshot
		_, err := rdsCli.RestoreDBInstanceFromDBSnapshot(ctx, instanceID, snapshotID, securityGroupID)
		if err != nil {
			return nil, errors.Wrapf(err, "error restoring database instance from snapshot")
		}

		log.Print("waiting for database to be ready", field.M{"instanceID": instanceID})
		// wait for instance to be ready
		err = rdsCli.WaitUntilDBInstanceAvailable(ctx, instanceID)
		if err != nil {
			return nil, errors.Wrap(err, "error while waiting for new rds instance to be ready.")
		}

		return nil, nil
	}

	// restore from backup
	var command []string
	// convert the profile object to string
	// profilejson, err := json.Marshal(profile)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "error converting profile object to string")
	// }

	descOp, err := rdsCli.DescribeDBInstances(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	dbEndpoint := *descOp.DBInstances[0].Endpoint.Address

	command, image, err := prepareCommand(PostgreSQLEngine, RestoreAction, instanceID, dbEndpoint, username, password, backupArtifactPrefix, backupID, profile)
	// switch dbEngine {
	// case string(PostgreSQLEngine):
	// 	command = getPostgreSQLRestoreCommand(dbEndpoint, password, backupArtifactPrefix, backupID, username, string(profilejson), "postgres")

	// default:
	// 	return nil, errors.New("provided value of dbEngine is incorrect")
	// }

	return restorePostgreSQLFrom(ctx, image, command)
}

func getPostgreSQLRestoreCommand(pgHost, password, backupArtifactPrefix, backupID, username string, profile *param.Profile) ([]string, error) {
	// convert the profile object to string
	profilejson, err := json.Marshal(profile)
	if err != nil {
		return nil, errors.Wrapf(err, "error converting profile object to string")
	}
	// TODO: use rds dbEngine lib to communicate to the datbase instead of using BASH
	// TODO: use secrets to read the secrets details don't set as ENV var
	return []string{
		"bash",
		"-o",
		"errexit",
		"-o",
		"pipefail",
		"-c",
		fmt.Sprintf(`
		export PGHOST=%s
		export PGPASSWORD=%s
		export PGUSER=%s

		if psql -l | grep -Fwq  "postgres"
		then 
		DATABASE=postgres
		elif psql -l | grep -Fwq  "template1"
		then 
		DATABASE=template1
		else
		echo "either postgres or template1 database should already be there in the database."
		EXIT 1
		fi

		kando location pull --profile '%s' --path "%s" - | gunzip -c -f | psql -q -U "${PGUSER}" "${DATABASE}"
		`, pgHost, password, username, profilejson, fmt.Sprintf("%s/%s", backupArtifactPrefix, backupID)),
	}, nil
}

func restorePostgreSQLFrom(ctx context.Context, image string, command []string) (map[string]interface{}, error) {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return nil, err
	}

	kubeclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting kubeclient from kubeconfig")
	}

	ns, err := kube.GetControllerNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting controller namespace")
	}

	return kubeTask(ctx, kubeclient, ns, image, command, nil)
}
