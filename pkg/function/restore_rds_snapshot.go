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
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	rdserr "github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
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
	RestoreRDSSnapshotDBEngine = "dbEngine"
	// RestoreRDSSnapshotNamespace for namespace arg
	RestoreRDSSnapshotNamespace = "namespace"
	// RestoreRDSSnapshotInstanceID is ID of the target instance
	RestoreRDSSnapshotInstanceID = "instanceID"
	// RestoreRDSSnapshotBackupArtifactPrefix stores the prefix of backup in object storage
	RestoreRDSSnapshotBackupArtifactPrefix = "backupArtifactPrefix"
	// RestoreRDSSnapshotBackupID stores the ID of backup in object storage
	RestoreRDSSnapshotBackupID = "backupID"
	// RestoreRDSSnapshotSnapshotID stores the snapshot ID
	RestoreRDSSnapshotSnapshotID = "snapshotID"
	// RestoreRDSSnapshotEndpoint to set endpoint of restored rds instance
	RestoreRDSSnapshotEndpoint = "endpoint"

	// RestoreRDSSnapshotUsername stores username of the database
	RestoreRDSSnapshotUsername = "username"
	// RestoreRDSSnapshotPassword stores the password of the database
	RestoreRDSSnapshotPassword = "password"

	// PostgreSQLEngine stores the postgres appname
	PostgreSQLEngine RDSDBEngine = "PostgreSQL"
)

type restoreRDSSnapshotFunc struct{}

func (*restoreRDSSnapshotFunc) Name() string {
	return RestoreRDSSnapshotFuncName
}

func (*restoreRDSSnapshotFunc) RequiredArgs() []string {
	return []string{RestoreRDSSnapshotNamespace, RestoreRDSSnapshotInstanceID, RestoreRDSSnapshotDBEngine}
}

func (*restoreRDSSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, instanceID, snapshotID, backupArtifactPrefix, backupID, username, password string
	var dbEngine RDSDBEngine

	if err := Arg(args, RestoreRDSSnapshotNamespace, &namespace); err != nil {
		return nil, err
	}
	if err := Arg(args, RestoreRDSSnapshotInstanceID, &instanceID); err != nil {
		return nil, err
	}

	if err := OptArg(args, RestoreRDSSnapshotSnapshotID, &snapshotID, ""); err != nil {
		return nil, err
	}

	if snapshotID == "" {
		// Snapshot ID is not provided get backupPrefix and backupID
		if err := Arg(args, RestoreRDSSnapshotBackupArtifactPrefix, &backupArtifactPrefix); err != nil {
			return nil, err
		}

		if err := Arg(args, RestoreRDSSnapshotBackupID, &backupID); err != nil {
			return nil, err
		}

		if err := Arg(args, RestoreRDSSnapshotUsername, &username); err != nil {
			return nil, err
		}
		if err := Arg(args, RestoreRDSSnapshotPassword, &password); err != nil {
			return nil, err
		}
		if err := Arg(args, RestoreRDSSnapshotDBEngine, &dbEngine); err != nil {
			return nil, err
		}
	}

	return restoreRDSSnapshot(ctx, namespace, instanceID, snapshotID, backupArtifactPrefix, backupID, username, password, dbEngine, tp.Profile)
}

func restoreRDSSnapshot(ctx context.Context, namespace, instanceID, snapshotID, backupArtifactPrefix, backupID, username, password string, dbEngine RDSDBEngine, profile *param.Profile) (map[string]interface{}, error) {
	// Validate profile
	if err := ValidateProfile(profile); err != nil {
		return nil, errors.Wrap(err, "Error validating profile")
	}

	awsConfig, region, err := getAWSConfigFromProfile(ctx, profile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AWS creds from profile")
	}

	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create RDS client")
	}

	// Restore from snapshot
	if snapshotID != "" {
		// Find security group ids
		sgIDs, err := findSecurityGroups(ctx, rdsCli, instanceID)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch security group ids. InstanceID=%s", instanceID)
		}
		return nil, restoreFromSnapshot(ctx, rdsCli, instanceID, snapshotID, sgIDs)
	}

	// Restore from dump
	descOp, err := rdsCli.DescribeDBInstances(ctx, instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to describe DB instance. InstanceID=%s", instanceID)
	}

	dbEndpoint := *descOp.DBInstances[0].Endpoint.Address
	if _, err = execDumpCommand(ctx, dbEngine, RestoreAction, namespace, instanceID, dbEndpoint, username, password, backupArtifactPrefix, backupID, profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to restore RDS from dump. InstanceID=%s", instanceID)
	}

	return map[string]interface{}{
		RestoreRDSSnapshotEndpoint: dbEndpoint,
	}, nil
}

func postgresRestoreCommand(pgHost, username, password string, dbList []string, backupArtifactPrefix, backupID string, profile []byte) ([]string, error) {

	if len(dbList) == 0 {
		return nil, errors.New("No database found. Atleast one db needed to connect")
	}

	return []string{
		"bash",
		"-o",
		"errexit",
		"-o",
		"pipefail",
		"-c",
		fmt.Sprintf(`
		export PGHOST=%s
		kando location pull --profile '%s' --path "%s" - | gunzip -c -f | psql -q -U "${PGUSER}" %s
		`, pgHost, profile, fmt.Sprintf("%s/%s", backupArtifactPrefix, backupID), dbList[0]),
	}, nil
}

func restoreFromSnapshot(ctx context.Context, rdsCli *rds.RDS, instanceID, snapshotID string, securityGrpIDs []*string) error {
	log.Print("Deleting existing instance.", field.M{"instanceID": instanceID})
	if _, err := rdsCli.DeleteDBInstance(ctx, instanceID); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBInstanceNotFoundFault {
				return err
			}
			log.Print("RDS instance is not present ErrCodeDBInstanceNotFoundFault", field.M{"instanceID": instanceID})
		}
	} else {
		// Wait for the instance to be deleted
		if err := rdsCli.WaitUntilDBInstanceDeleted(ctx, instanceID); err != nil {
			return errors.Wrapf(err, "Error waiting for the dbinstance to be available")
		}
	}

	log.Print("Restoring database from snapshot.", field.M{"instanceID": instanceID})
	// Restore from snapshot
	if _, err := rdsCli.RestoreDBInstanceFromDBSnapshot(ctx, instanceID, snapshotID, securityGrpIDs); err != nil {
		return errors.Wrapf(err, "Error restoring database instance from snapshot")
	}

	// Wait for instance to be ready
	log.Print("Waiting for database to be ready.", field.M{"instanceID": instanceID})
	err := rdsCli.WaitUntilDBInstanceAvailable(ctx, instanceID)
	return errors.Wrap(err, "Error while waiting for new rds instance to be ready.")
}
