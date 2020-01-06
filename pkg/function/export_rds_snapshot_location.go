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

	"github.com/pkg/errors"
	"github.com/teris-io/shortid"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	_ = kanister.Register(&exportRDSSnapshotToLocationFunc{})
}

var (
	_ kanister.Func = (*exportRDSSnapshotToLocationFunc)(nil)
)

const (
	ExportRDSSnapshotToLocFuncName           = "ExportRDSSnapshotToLocation"
	ExportRDSSnapshotToLocInstanceIDArg      = "instanceID"
	ExportRDSSnapshotToLocSnapshotIDArg      = "snapshotID"
	ExportRDSSnapshotToLocDBUsernameArg      = "username"
	ExportRDSSnapshotToLocDBPasswordArg      = "password"
	ExportRDSSnapshotToLocBackupArtPrefixArg = "backupArtifactPrefix"
	ExportRDSSnapshotToLocDBEngineArg        = "dbengine"
	ExportRDSSnapshotToLocBackupID           = "backupID"

	PostgrSQLEngine RDSDBEngine = "PostgreSQL"

	BackupAction  RDSAction = "backup"
	RestoreAction RDSAction = "restore"

	postgresToolsImage = "kanisterio/postgres-kanister-tools:0.23.0"
)

type exportRDSSnapshotToLocationFunc struct{}

type RDSDBEngine string
type RDSAction string

func (*exportRDSSnapshotToLocationFunc) Name() string {
	return ExportRDSSnapshotToLocFuncName
}

func exportRDSSnapshotToLoc(ctx context.Context, instanceID, snapshotID, username, password, backupPrefix string, dbEngine RDSDBEngine, profile *param.Profile) (map[string]interface{}, error) {
	// Validate profilextractDumpFromDBe
	if err := ValidateProfile(profile); err != nil {
		return nil, errors.Wrapf(err, "Profile Validation failed")
	}

	awsConfig, region, err := getAWSConfigFromProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	// Create rds client
	rdsCli, err := rds.NewClient(ctx, awsConfig, region)
	if err != nil {
		return nil, err
	}

	// Create tmp instance from the snapshot
	randomID, err := shortid.Generate()
	if err != nil {
		return nil, err
	}

	tmpInstanceID := fmt.Sprintf("%s-%s", instanceID, randomID)
	log.Print("Restore RDS instance from snapshot.", field.M{"SnapshotID": snapshotID, "InstanceID": tmpInstanceID})
	// TODO: Use RDSRestoreSnapshot function instead
	sgIDs, err := findSecurityGroups(ctx, rdsCli, instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch security group ids. InstanceID=%s", instanceID)
	}
	_, err = rdsCli.RestoreDBInstanceFromDBSnapshot(ctx, tmpInstanceID, snapshotID, sgIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to restore snapshot. SnapshotID=%s", snapshotID)
	}
	// Wait until snapshot becomes available
	log.Print("Waiting for RDS DB instance to be available", field.M{"InstanceID": tmpInstanceID})
	if err := rdsCli.WaitUntilDBInstanceAvailable(ctx, tmpInstanceID); err != nil {
		return nil, err
	}

	// Find host of the instance
	dbInstance, err := rdsCli.DescribeDBInstances(ctx, tmpInstanceID)
	if err != nil {
		return nil, err
	}
	dbEndpoint := *dbInstance.DBInstances[0].Endpoint.Address

	// Extract dump from DB
	output, err := extractAndPushDump(ctx, dbEngine, tmpInstanceID, dbEndpoint, username, password, backupPrefix, profile)
	if err != nil {
		return nil, err
	}

	// Deleting tmp instance
	log.Print("Delete temporary RDS instance.", field.M{"SnapshotID": snapshotID, "InstanceID": tmpInstanceID})
	_, err = rdsCli.DeleteDBInstance(ctx, tmpInstanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to delete rds instance")
	}
	// Wait until instance is deleted
	log.Print("Waiting for RDS DB instance to be deleted", field.M{"InstanceID": tmpInstanceID})
	if err := rdsCli.WaitUntilDBInstanceDeleted(ctx, tmpInstanceID); err != nil {
		return nil, err
	}

	// Add output artifacts
	output[ExportRDSSnapshotToLocSnapshotIDArg] = snapshotID
	output[ExportRDSSnapshotToLocInstanceIDArg] = instanceID

	return output, nil
}

func (crs *exportRDSSnapshotToLocationFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var instanceID, snapshotID, username, password, backupArtifact string
	var dbEngine RDSDBEngine
	if err := Arg(args, ExportRDSSnapshotToLocInstanceIDArg, &instanceID); err != nil {
		return nil, err
	}
	if err := Arg(args, ExportRDSSnapshotToLocSnapshotIDArg, &snapshotID); err != nil {
		return nil, err
	}
	if err := Arg(args, ExportRDSSnapshotToLocDBEngineArg, &dbEngine); err != nil {
		return nil, err
	}
	if err := OptArg(args, ExportRDSSnapshotToLocDBUsernameArg, &username, ""); err != nil {
		return nil, err
	}
	if err := OptArg(args, ExportRDSSnapshotToLocDBPasswordArg, &password, ""); err != nil {
		return nil, err
	}
	if err := OptArg(args, ExportRDSSnapshotToLocBackupArtPrefixArg, &backupArtifact, instanceID); err != nil {
		return nil, err
	}
	return exportRDSSnapshotToLoc(ctx, instanceID, snapshotID, username, password, backupArtifact, dbEngine, tp.Profile)
}

func (*exportRDSSnapshotToLocationFunc) RequiredArgs() []string {
	return []string{ExportRDSSnapshotToLocInstanceIDArg, ExportRDSSnapshotToLocSnapshotIDArg, ExportRDSSnapshotToLocDBEngineArg}
}

func extractAndPushDump(ctx context.Context, dbEngine RDSDBEngine, instanceID, dbEndpoint, username, password, backupPrefix string, profile *param.Profile) (map[string]interface{}, error) {
	command, image, err := prepareCommand(dbEngine, BackupAction, instanceID, dbEndpoint, username, password, backupPrefix, profile)
	if err != nil {
		return nil, err
	}

	// Execute kubetask
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}

	namespace, err := kube.GetControllerNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get controller namespace")
	}

	return kubeTask(ctx, cli, namespace, image, command, nil)
}

func prepareCommand(dbEngine RDSDBEngine, action RDSAction, instanceID, dbEndpoint, username, password, backupPrefix string, profile *param.Profile) ([]string, string, error) {
	switch dbEngine {
	case PostgrSQLEngine:
		switch action {
		case BackupAction:
			command, err := postgresBackupCommand(instanceID, dbEndpoint, username, password, backupPrefix, profile)
			return command, postgresToolsImage, err
		case RestoreAction:
			fallthrough
		default:
		}
	default:
	}
	return nil, "", errors.New("Invalid RDSDBEngine or RDSAction")
}

func postgresBackupCommand(instanceID, dbEndpoint, username, password, backupPrefix string, profile *param.Profile) ([]string, error) {
	profileJson, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}
	randomID, err := shortid.Generate()
	if err != nil {
		return nil, err
	}
	backupID := fmt.Sprintf("backup-%s.tar.gz", randomID)
	// TODO:
	// 1. Pass and read creds from K8s Secrets
	// 2. Find list of dbs using correct postgres go sdks
	command := []string{
		"bash",
		"-o",
		"errexit",
		"-o",
		"pipefail",
		"-c",
		fmt.Sprintf(`
			export PGHOST=%s
			export PGUSER=%s
			export PGPASSWORD=%s
			BACKUP_PREFIX=%s
			BACKUP_ID=%s

			mkdir /backup
			restricted=("template0", "rdsadmin")
			psql -lqt | awk -F "|" '{print $1}' | tr -d " " | sed '/^$/d' |
			while read db;
			  do [[ ! ${restricted[@]} =~ ${db} ]] && echo "backing up $db db" && pg_dump $db -C > /backup/$db.sql;
			done
			tar -zc backup | kando location push --profile '%s' --path "${BACKUP_PREFIX}/${BACKUP_ID}" -
			kando output %s ${BACKUP_ID}`,
			dbEndpoint, username, password, backupPrefix, backupID, profileJson, ExportRDSSnapshotToLocBackupID),
	}
	return command, nil
}
