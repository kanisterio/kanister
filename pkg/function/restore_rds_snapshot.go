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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	rdserr "github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/postgres"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
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
	// RestoreRDSSnapshotDBSubnetGroup is the dbSubnetGroup of the restored RDS instance
	RestoreRDSSnapshotDBSubnetGroup = "dbSubnetGroup"
	// RestoreRDSSnapshotUsername stores username of the database
	RestoreRDSSnapshotUsername = "username"
	// RestoreRDSSnapshotPassword stores the password of the database
	RestoreRDSSnapshotPassword = "password"
	// RestoreRDSSnapshotImage provides the image of the container with required tools
	RestoreRDSSnapshotImage = "image"

	// PostgreSQLEngine stores the postgres appname
	PostgreSQLEngine RDSDBEngine = "PostgreSQL"

	restoredAuroraInstanceSuffix       = "instance-1"
	defaultAuroraInstanceClass         = "db.r5.large"
	RDSPostgresDBInstanceEngineVersion = "13.0"
)

type restoreRDSSnapshotFunc struct {
	progressPercent string
}

func (*restoreRDSSnapshotFunc) Name() string {
	return RestoreRDSSnapshotFuncName
}

func (*restoreRDSSnapshotFunc) RequiredArgs() []string {
	return []string{RestoreRDSSnapshotInstanceID}
}

func (*restoreRDSSnapshotFunc) Arguments() []string {
	return []string{
		RestoreRDSSnapshotInstanceID,
		RestoreRDSSnapshotSnapshotID,
		RestoreRDSSnapshotDBEngine,
		RestoreRDSSnapshotBackupArtifactPrefix,
		RestoreRDSSnapshotBackupID,
		RestoreRDSSnapshotUsername,
		RestoreRDSSnapshotPassword,
		RestoreRDSSnapshotNamespace,
		RestoreRDSSnapshotSecGrpID,
		RestoreRDSSnapshotDBSubnetGroup,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (r *restoreRDSSnapshotFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(r.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(r.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(r.RequiredArgs(), args)
}

func (r *restoreRDSSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	r.progressPercent = progress.StartedPercent
	defer func() { r.progressPercent = progress.CompletedPercent }()

	var namespace, instanceID, subnetGroup, snapshotID, backupArtifactPrefix, backupID, username, password, postgresToolsImage string
	var dbEngine RDSDBEngine
	var bpAnnotations, bpLabels map[string]string

	if err := Arg(args, RestoreRDSSnapshotInstanceID, &instanceID); err != nil {
		return nil, err
	}

	if err := OptArg(args, RestoreRDSSnapshotSnapshotID, &snapshotID, ""); err != nil {
		return nil, err
	}

	if err := OptArg(args, RestoreRDSSnapshotDBEngine, &dbEngine, ""); err != nil {
		return nil, err
	}
	if err := OptArg(args, RestoreRDSSnapshotDBSubnetGroup, &subnetGroup, "default"); err != nil {
		return nil, err
	}
	if err := OptArg(args, RestoreRDSSnapshotImage, &postgresToolsImage, defaultPostgresToolsImage); err != nil {
		return nil, err
	}
	if err := OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err := OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}

	annotations := bpAnnotations
	labels := bpLabels
	if tp.PodAnnotations != nil {
		// merge the actionset annotations with blueprint annotations
		var actionSetAnn ActionSetAnnotations = tp.PodAnnotations
		annotations = actionSetAnn.MergeBPAnnotations(bpAnnotations)
	}

	if tp.PodLabels != nil {
		// merge the actionset labels with blueprint labels
		var actionSetLabels ActionSetLabels = tp.PodLabels
		labels = actionSetLabels.MergeBPLabels(bpLabels)
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

	return restoreRDSSnapshot(
		ctx,
		namespace,
		instanceID,
		subnetGroup,
		snapshotID,
		backupArtifactPrefix,
		backupID,
		username,
		password,
		dbEngine,
		sgIDs,
		tp.Profile,
		postgresToolsImage,
		annotations,
		labels,
	)
}

func (r *restoreRDSSnapshotFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    r.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func restoreRDSSnapshot(
	ctx context.Context,
	namespace,
	instanceID,
	subnetGroup,
	snapshotID,
	backupArtifactPrefix,
	backupID,
	username,
	password string,
	dbEngine RDSDBEngine,
	sgIDs []string,
	profile *param.Profile,
	postgresToolsImage string,
	annotations,
	labels map[string]string,
) (map[string]interface{}, error) {
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
			sgIDs, err = findSecurityGroupIDs(ctx, rdsCli, instanceID, string(dbEngine))
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to fetch security group ids. InstanceID=%s", instanceID)
			}
		}
		if !isAuroraCluster(string(dbEngine)) {
			return nil, restoreFromSnapshot(ctx, rdsCli, instanceID, subnetGroup, snapshotID, sgIDs)
		}
		return nil, restoreAuroraFromSnapshot(ctx, rdsCli, instanceID, subnetGroup, snapshotID, string(dbEngine), sgIDs)
	}

	// Restore from dump
	descOp, err := rdsCli.DescribeDBInstances(ctx, instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to describe DB instance. InstanceID=%s", instanceID)
	}

	dbEndpoint := *descOp.DBInstances[0].Endpoint.Address

	// get the engine version
	dbEngineVersion, err := rdsDBEngineVersion(ctx, rdsCli, instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't find DBInstance Version")
	}

	if _, err = execDumpCommand(
		ctx,
		dbEngine,
		RestoreAction,
		namespace,
		dbEndpoint,
		username,
		password,
		nil,
		backupArtifactPrefix,
		backupID,
		profile,
		dbEngineVersion,
		postgresToolsImage,
		annotations,
		labels,
	); err != nil {
		return nil, errors.Wrapf(err, "Failed to restore RDS from dump. InstanceID=%s", instanceID)
	}

	return map[string]interface{}{
		RestoreRDSSnapshotEndpoint: dbEndpoint,
	}, nil
}
func findSecurityGroupIDs(ctx context.Context, rdsCli *rds.RDS, instanceID, dbEngine string) ([]string, error) {
	if !isAuroraCluster(dbEngine) {
		return findSecurityGroups(ctx, rdsCli, instanceID)
	}
	return findAuroraSecurityGroups(ctx, rdsCli, instanceID)
}

//nolint:unparam
func postgresRestoreCommand(pgHost, username, password string, backupArtifactPrefix, backupID string, profile []byte, dbEngineVersion string) ([]string, error) {
	replaceCommand := ""

	// check if PostgresDB version < 13
	v1, err := version.NewVersion(dbEngineVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't find DBInstance Version")
	}
	// Add Constraints
	constraints, err := version.NewConstraint("< " + RDSPostgresDBInstanceEngineVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't add constraint to DBInstance Version")
	}
	// Verify Constraints
	if constraints.Check(v1) {
		replaceCommand = ` sed 's/"LOCALE"/"LC_COLLATE"/' |`
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
		kando location pull --profile '%s' --path "%s" - | gunzip -c -f |%s psql -q -U "${PGUSER}" %s
		`, pgHost, profile, fmt.Sprintf("%s/%s", backupArtifactPrefix, backupID), replaceCommand, postgres.DefaultConnectDatabase),
	}, nil
}

func restoreFromSnapshot(ctx context.Context, rdsCli *rds.RDS, instanceID, subnetGroup, snapshotID string, securityGrpIDs []string) error {
	log.WithContext(ctx).Print("Deleting existing RDS DB instance.", field.M{"instanceID": instanceID})
	if _, err := rdsCli.DeleteDBInstance(ctx, instanceID); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBInstanceNotFoundFault {
				return err
			}
			log.WithContext(ctx).Print("RDS instance is not present ErrCodeDBInstanceNotFoundFault", field.M{"instanceID": instanceID})
		}
	} else {
		log.WithContext(ctx).Print("Waiting for RDS DB instance to be deleted.", field.M{"instanceID": instanceID})
		// Wait for the instance to be deleted
		if err := rdsCli.WaitUntilDBInstanceDeleted(ctx, instanceID); err != nil {
			return errors.Wrapf(err, "Error while waiting RDS DB instance to be deleted")
		}
	}

	log.WithContext(ctx).Print("Restoring RDS DB instance from snapshot.", field.M{"instanceID": instanceID, "snapshotID": snapshotID})
	// Restore from snapshot
	if _, err := rdsCli.RestoreDBInstanceFromDBSnapshot(ctx, instanceID, subnetGroup, snapshotID, securityGrpIDs); err != nil {
		return errors.Wrapf(err, "Error restoring RDS DB instance from snapshot")
	}

	// Wait for instance to be ready
	log.WithContext(ctx).Print("Waiting for RDS DB instance database to be ready.", field.M{"instanceID": instanceID})
	err := rdsCli.WaitUntilDBInstanceAvailable(ctx, instanceID)
	return errors.Wrap(err, "Error while waiting for new rds instance to be ready.")
}

func restoreAuroraFromSnapshot(ctx context.Context, rdsCli *rds.RDS, instanceID, subnetGroup, snapshotID, dbEngine string, securityGroupIDs []string) error {
	// To delete an Aurora RDS instance we will have to delete all the instance that are running through it
	// Once all those instances are deleted, Aurora cluster will be deleted automatically
	descOp, err := rdsCli.DescribeDBClusters(ctx, instanceID)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBClusterNotFoundFault {
				return err
			}
			log.WithContext(ctx).Print("Aurora DB cluster is not found")
		}
	} else {
		// DB Cluster is present, delete and wait for it to be deleted
		if err := DeleteAuroraDBCluster(ctx, rdsCli, descOp, instanceID); err != nil {
			return nil
		}
	}

	version, err := engineVersion(ctx, rdsCli, snapshotID)
	if err != nil {
		return errors.Wrap(err, "Error getting the engine version before restore")
	}

	log.WithContext(ctx).Print("Restoring RDS Aurora DB Cluster from snapshot.", field.M{"instanceID": instanceID, "snapshotID": snapshotID})
	op, err := rdsCli.RestoreDBClusterFromDBSnapshot(ctx, instanceID, subnetGroup, snapshotID, dbEngine, version, securityGroupIDs)
	if err != nil {
		return errors.Wrap(err, "Error restorig aurora db cluster from snapshot")
	}

	// From docs: Above action only restores the DB cluster, not the DB instances for that DB cluster
	// wait for db cluster to be available
	log.WithContext(ctx).Print("Waiting for db cluster to be available")
	if err := rdsCli.WaitUntilDBClusterAvailable(ctx, *op.DBCluster.DBClusterIdentifier); err != nil {
		return errors.Wrap(err, "Error waiting for DBCluster to be available")
	}

	log.WithContext(ctx).Print("Creating DB instance in the cluster")
	// After Aurora cluster is created, we will have to explicitly create the DB instance
	dbInsOp, err := rdsCli.CreateDBInstance(
		ctx,
		nil,
		defaultAuroraInstanceClass,
		fmt.Sprintf("%s-%s", *op.DBCluster.DBClusterIdentifier, restoredAuroraInstanceSuffix),
		dbEngine,
		"",
		"",
		nil,
		nil,
		aws.String(*op.DBCluster.DBClusterIdentifier),
		subnetGroup,
	)
	if err != nil {
		return errors.Wrap(err, "Error while creating Aurora DB instance in the cluster.")
	}
	// wait for instance to be up and running
	log.WithContext(ctx).Print("Waiting for RDS Aurora instance to be ready.", field.M{"instanceID": instanceID})
	if err = rdsCli.WaitUntilDBInstanceAvailable(ctx, *dbInsOp.DBInstance.DBInstanceIdentifier); err != nil {
		return errors.Wrap(err, "Error while waiting for new RDS Aurora instance to be ready.")
	}
	return nil
}

func DeleteAuroraDBCluster(ctx context.Context, rdsCli *rds.RDS, descOp *rdserr.DescribeDBClustersOutput, instanceID string) error {
	for k, member := range descOp.DBClusters[0].DBClusterMembers {
		if _, err := rdsCli.DeleteDBInstance(ctx, *member.DBInstanceIdentifier); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() != rdserr.ErrCodeDBInstanceNotFoundFault {
					return err
				}
			}
		} else {
			log.WithContext(ctx).Print("Waiting for RDS Aurora cluster instance to be deleted", field.M{"instance": k})
			if err := rdsCli.WaitUntilDBInstanceDeleted(ctx, *member.DBInstanceIdentifier); err != nil {
				return errors.Wrapf(err, "Error while waiting for RDS Aurora DB instance to be deleted")
			}
		}
	}

	log.WithContext(ctx).Print("Deleting existing RDS Aurora DB Cluster.", field.M{"instanceID": instanceID})
	if _, err := rdsCli.DeleteDBCluster(ctx, instanceID); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rdserr.ErrCodeDBClusterNotFoundFault {
				return err
			}
		}
	} else {
		log.WithContext(ctx).Print("Waiting for RDS Aurora cluster to be deleted.", field.M{"instanceID": instanceID})
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
