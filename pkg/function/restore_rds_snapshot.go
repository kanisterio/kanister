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
	// RestoreRDSSnapshotSecGrpID stores securityGroupID in the args
	RestoreRDSSnapshotSecGrpID = "securityGroupID"
	// RestoreRDSSnapshotEndpoint to set endpoint of restored rds instance
	RestoreRDSSnapshotEndpoint = "endpoint"

	// RestoreRDSSnapshotUsername stores username of the database
	RestoreRDSSnapshotUsername = "username"
	// RestoreRDSSnapshotPassword stores the password of the database
	RestoreRDSSnapshotPassword = "password"

	// PostgreSQLEngine stores the postgres appname
	PostgreSQLEngine RDSDBEngine = "PostgreSQL"

	restoredAuroraInstanceSuffix = "instance-1"
	defaultAuroraInstanceClass   = "db.r5.large"
)

type restoreRDSSnapshotFunc struct{}

func (*restoreRDSSnapshotFunc) Name() string {
	return RestoreRDSSnapshotFuncName
}

func (*restoreRDSSnapshotFunc) RequiredArgs() []string {
	return []string{RestoreRDSSnapshotInstanceID}
}

func (*restoreRDSSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, instanceID, snapshotID, backupArtifactPrefix, backupID, username, password string
	var dbEngine RDSDBEngine

	if err := Arg(args, RestoreRDSSnapshotInstanceID, &instanceID); err != nil {
		return nil, err
	}

	if err := OptArg(args, RestoreRDSSnapshotSnapshotID, &snapshotID, ""); err != nil {
		return nil, err
	}

	if err := OptArg(args, RestoreRDSSnapshotDBEngine, &dbEngine, ""); err != nil {
		return nil, err
	}
	// Find security groups
	sgIDs, err := GetYamlList(args, RestoreRDSSnapshotSecGrpID)
	if err != nil {
		return nil, err
	}

	// if snapshotID is nil, we'll try to restore from dumps
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
		if err := Arg(args, RestoreRDSSnapshotNamespace, &namespace); err != nil {
			return nil, err
		}
	}

	return restoreRDSSnapshot(ctx, namespace, instanceID, snapshotID, backupArtifactPrefix, backupID, username, password, dbEngine, sgIDs, tp.Profile)
}

func restoreRDSSnapshot(ctx context.Context, namespace, instanceID, snapshotID, backupArtifactPrefix, backupID, username, password string, dbEngine RDSDBEngine, sgIDs []string, profile *param.Profile) (map[string]interface{}, error) {
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
		// If securityGroupID arg is nil, we will try to find the sgIDs by describing the existing instance
		// Find security group ids
		if sgIDs == nil {
			if !isAuroraCluster(string(dbEngine)) {
				sgIDs, err = findSecurityGroups(ctx, rdsCli, instanceID)
			} else {
				sgIDs, err = findAuroraSecurityGroups(ctx, rdsCli, instanceID)
			}
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to fetch security group ids. InstanceID=%s", instanceID)
			}
		}
		if !isAuroraCluster(string(dbEngine)) {
			return nil, restoreFromSnapshot(ctx, rdsCli, instanceID, snapshotID, sgIDs)
		}
		return nil, restoreAuroraFromSnapshot(ctx, rdsCli, instanceID, snapshotID, string(dbEngine), sgIDs)
	}

	// Restore from dump
	descOp, err := rdsCli.DescribeDBInstances(ctx, instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to describe DB instance. InstanceID=%s", instanceID)
	}

	dbEndpoint := *descOp.DBInstances[0].Endpoint.Address
	if _, err = execDumpCommand(ctx, dbEngine, RestoreAction, namespace, dbEndpoint, username, password, nil, backupArtifactPrefix, backupID, profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to restore RDS from dump. InstanceID=%s", instanceID)
	}

	return map[string]interface{}{
		RestoreRDSSnapshotEndpoint: dbEndpoint,
	}, nil
}

// nolint:unparam
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

func restoreFromSnapshot(ctx context.Context, rdsCli *rds.RDS, instanceID, snapshotID string, securityGrpIDs []string) error {
	log.Print("Deleting existing RDS DB instance.", field.M{"instanceID": instanceID})
	if _, err := rdsCli.DeleteDBInstance(ctx, instanceID); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBInstanceNotFoundFault {
				return err
			}
			log.Print("RDS instance is not present ErrCodeDBInstanceNotFoundFault", field.M{"instanceID": instanceID})
		}
	} else {
		log.Print("Waiting for RDS DB instance to be deleted.", field.M{"instanceID": instanceID})
		// Wait for the instance to be deleted
		if err := rdsCli.WaitUntilDBInstanceDeleted(ctx, instanceID); err != nil {
			return errors.Wrapf(err, "Error while waiting RDS DB instance to be deleted")
		}
	}

	log.Print("Restoring RDS DB instance from snapshot.", field.M{"instanceID": instanceID, "snapshotID": snapshotID})
	// Restore from snapshot
	if _, err := rdsCli.RestoreDBInstanceFromDBSnapshot(ctx, instanceID, snapshotID, securityGrpIDs); err != nil {
		return errors.Wrapf(err, "Error restoring RDS DB instance from snapshot")
	}

	// Wait for instance to be ready
	log.Print("Waiting for RDS DB instance database to be ready.", field.M{"instanceID": instanceID})
	err := rdsCli.WaitUntilDBInstanceAvailable(ctx, instanceID)
	return errors.Wrap(err, "Error while waiting for new rds instance to be ready.")
}

func restoreAuroraFromSnapshot(ctx context.Context, rdsCli *rds.RDS, instanceID, snapshotID, dbEngine string, securityGroupIDs []string) error {
	// To delete an Aurora RDS instance we will have to delete all the instance that are running through it
	// Once all those instances are deleted, Aurora cluster will be deleted automatically
	descOp, err := rdsCli.DescribeDBClusters(ctx, instanceID)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBClusterNotFoundFault {
				return err
			}
			log.Print("Aurora DB cluster is not found")
		}
	} else {
		// DB Cluster is present, delete and wait for it to be deleted
		if err := deleteAuroraDBCluster(ctx, rdsCli, descOp, instanceID); err != nil {
			return nil
		}
	}

	version, err := engineVersion(ctx, rdsCli, snapshotID)
	if err != nil {
		return errors.Wrap(err, "Error getting the engine version before restore")
	}

	log.Print("Restoring RDS Aurora DB Cluster from snapshot.", field.M{"instanceID": instanceID, "snapshotID": snapshotID})
	op, err := rdsCli.RestoreDBClusterFromDBSnapshot(ctx, instanceID, snapshotID, dbEngine, version, securityGroupIDs)
	if err != nil {
		return errors.Wrap(err, "Error restorig aurora db cluster from snapshot")
	}

	// From docs: Above action only restores the DB cluster, not the DB instances for that DB cluster
	// wait for db cluster to be available
	log.Print("Waiting for db cluster to be available")
	if err := rdsCli.WaitUntilDBClusterAvailable(ctx, *op.DBCluster.DBClusterIdentifier); err != nil {
		return errors.Wrap(err, "Error waiting for DBCluster to be available")
	}

	log.Print("Creating DB instance in the cluster")
	// After Aurora cluster is created, we will have to explictly create the DB instance
	dbInsOp, err := rdsCli.CreateDBInstanceInCluster(ctx, *op.DBCluster.DBClusterIdentifier, fmt.Sprintf("%s-%s", *op.DBCluster.DBClusterIdentifier, restoredAuroraInstanceSuffix), defaultAuroraInstanceClass, dbEngine)
	if err != nil {
		return errors.Wrap(err, "Error while creating Aurora DB instance in the cluster.")
	}
	// wait for instance to be up and running
	log.Print("Waiting for RDS Aurora instance to be ready.", field.M{"instanceID": instanceID})
	if err = rdsCli.WaitUntilDBInstanceAvailable(ctx, *dbInsOp.DBInstance.DBInstanceIdentifier); err != nil {
		return errors.Wrap(err, "Error while waiting for new RDS Aurora instance to be ready.")
	}
	return nil
}

func deleteAuroraDBCluster(ctx context.Context, rdsCli *rds.RDS, descOp *rdserr.DescribeDBClustersOutput, instanceID string) error {
	for k, member := range descOp.DBClusters[0].DBClusterMembers {
		if _, err := rdsCli.DeleteDBInstance(ctx, *member.DBInstanceIdentifier); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() != rdserr.ErrCodeDBInstanceNotFoundFault {
					return err
				}
			}
		} else {
			log.Print("Waiting for RDS Aurora cluster instance to be deleted", field.M{"instance": k})
			if err := rdsCli.WaitUntilDBInstanceDeleted(ctx, *member.DBInstanceIdentifier); err != nil {
				return errors.Wrapf(err, "Error while waiting for RDS Aurora DB instance to be deleted")
			}
		}
	}

	log.Print("Deleting existing RDS Aurora DB Cluster.", field.M{"instanceID": instanceID})
	if _, err := rdsCli.DeleteDBCluster(ctx, instanceID); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBClusterNotFoundFault {
				return err
			}
		}
	} else {
		log.Print("Waiting for RDS Aurora cluster to be deleted.", field.M{"instanceID": instanceID})
		if err := rdsCli.WaitUntilDBClusterDeleted(ctx, instanceID); err != nil {
			return errors.Wrapf(err, "Error while waiting RDS Aurora DB cluster to be deleted")
		}
	}
	return nil
}

func engineVersion(ctx context.Context, rdsCli *rds.RDS, snapshotID string) (string, error) {
	snapshot, err := rdsCli.DescribeDBClustersSnapshot(ctx, snapshotID)
	if err != nil {
		return "", err
	}
	return *snapshot.DBClusterSnapshots[0].EngineVersion, nil
}
