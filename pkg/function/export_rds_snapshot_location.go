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
	"k8s.io/apimachinery/pkg/util/rand"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
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
	ExportRDSSnapshotToLocNamespaceArg       = "namespace"
	ExportRDSSnapshotToLocInstanceIDArg      = "instanceID"
	ExportRDSSnapshotToLocSnapshotIDArg      = "snapshotID"
	ExportRDSSnapshotToLocDBUsernameArg      = "username"
	ExportRDSSnapshotToLocDBPasswordArg      = "password"
	ExportRDSSnapshotToLocBackupArtPrefixArg = "backupArtifactPrefix"
	ExportRDSSnapshotToLocDBEngineArg        = "dbEngine"
	ExportRDSSnapshotToLocBackupID           = "backupID"

	PostgrSQLEngine RDSDBEngine = "PostgreSQL"

	BackupAction  RDSAction = "backup"
	RestoreAction RDSAction = "restore"

	postgresToolsImage = "kanisterio/postgres-kanister-tools:0.23.0"
)

type exportRDSSnapshotToLocationFunc struct{}

// RDSDBEngine for RDS Engine types
type RDSDBEngine string

// RDSAction for RDS backup or restore actions
type RDSAction string

func (*exportRDSSnapshotToLocationFunc) Name() string {
	return ExportRDSSnapshotToLocFuncName
}

func exportRDSSnapshotToLoc(ctx context.Context, namespace, instanceID, snapshotID, username, password, backupPrefix string, dbEngine RDSDBEngine, profile *param.Profile) (map[string]interface{}, error) {
	// Validate profilextractDumpFromDBe
	if err := ValidateProfile(profile); err != nil {
		return nil, errors.Wrap(err, "Profile Validation failed")
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

	// Create tmp instance from the snapshot
	tmpInstanceID := fmt.Sprintf("%s-%s", instanceID, rand.String(10))

	log.Print("Restore RDS instance from snapshot.", field.M{"SnapshotID": snapshotID, "InstanceID": tmpInstanceID})

	sgIDs, err := findSecurityGroups(ctx, rdsCli, instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch security group ids. InstanceID=%s", instanceID)
	}

	// Create tmp instance from snapshot
	if err := restoreFromSnapshot(ctx, rdsCli, tmpInstanceID, snapshotID, sgIDs); err != nil {
		return nil, errors.Wrapf(err, "Failed to restore snapshot. SnapshotID=%s", snapshotID)
	}
	defer func() {
		if err := cleanupRDSDB(ctx, rdsCli, tmpInstanceID); err != nil {
			log.Error().WithError(err).Print("Failed to delete rds instance")
		}
	}()

	// Find host of the instance
	dbEndpoint, err := findRDSEndpoint(ctx, rdsCli, tmpInstanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't find endpoint for instance %s", tmpInstanceID)
	}

	// Create unique backupID
	backupID := fmt.Sprintf("backup-%s.tar.gz", rand.String(10))

	// Extract dump from DB
	output, err := execDumpCommand(ctx, dbEngine, BackupAction, namespace, tmpInstanceID, dbEndpoint, username, password, backupPrefix, backupID, profile)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to extract and push db dump to location")
	}

	// Add output artifacts
	output[ExportRDSSnapshotToLocSnapshotIDArg] = snapshotID
	output[ExportRDSSnapshotToLocInstanceIDArg] = instanceID

	return output, nil
}

func (crs *exportRDSSnapshotToLocationFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, instanceID, snapshotID, username, password, backupArtifact string
	var dbEngine RDSDBEngine

	if err := Arg(args, ExportRDSSnapshotToLocNamespaceArg, &namespace); err != nil {
		return nil, err
	}
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

	return exportRDSSnapshotToLoc(ctx, namespace, instanceID, snapshotID, username, password, backupArtifact, dbEngine, tp.Profile)
}

func (*exportRDSSnapshotToLocationFunc) RequiredArgs() []string {
	return []string{ExportRDSSnapshotToLocNamespaceArg, ExportRDSSnapshotToLocInstanceIDArg, ExportRDSSnapshotToLocSnapshotIDArg, ExportRDSSnapshotToLocDBEngineArg}
}

func execDumpCommand(ctx context.Context, dbEngine RDSDBEngine, action RDSAction, namespace, instanceID, dbEndpoint, username, password, backupPrefix, backupID string, profile *param.Profile) (map[string]interface{}, error) {
	// Prepare and execute command with kubetask
	command, image, err := prepareCommand(dbEngine, BackupAction, instanceID, dbEndpoint, username, password, backupPrefix, backupID, profile)
	if err != nil {
		return nil, err
	}

	// Create Kubernetes client
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	// Create cred secret
	secretName := "postgres-secret"
	err = createPostgresSecret(cli, secretName, namespace, username, password)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create postgres secret")
	}

	defer func() {
		if err := deletePostgresSecret(cli, secretName, namespace); err != nil {
			log.Error().WithError(err).Print("Failed to cleanup postgres-secret")
		}
	}()

	injectSecret := crv1alpha1.JSONMap{
		"containers": []map[string]interface{}{
			{
				"name": "container",
				"env": []map[string]interface{}{
					{
						"name": "PGUSER",
						"valueFrom": map[string]interface{}{
							"secretKeyRef": map[string]interface{}{
								"name": secretName,
								"key":  "username",
							},
						},
					},
					{
						"name": "PGPASSWORD",
						"valueFrom": map[string]interface{}{
							"secretKeyRef": map[string]interface{}{
								"name": secretName,
								"key":  "password",
							},
						},
					},
				},
			},
		},
	}

	return kubeTask(ctx, cli, namespace, image, command, injectSecret)
}

func prepareCommand(dbEngine RDSDBEngine, action RDSAction, instanceID, dbEndpoint, username, password, backupPrefix, backupID string, profile *param.Profile) ([]string, string, error) {
	// Convert profile object into json
	profileJson, err := json.Marshal(profile)
	if err != nil {
		return nil, "", err
	}

	switch dbEngine {
	case PostgrSQLEngine:
		switch action {
		case BackupAction:
			command, err := postgresBackupCommand(dbEndpoint, username, password, backupPrefix, backupID, profileJson)
			return command, postgresToolsImage, err
		case RestoreAction:
			command, err := postgresRestoreCommand(dbEndpoint, username, password, backupPrefix, backupID, profileJson)
			return command, postgresToolsImage, err
		}
	}
	return nil, "", errors.New("Invalid RDSDBEngine or RDSAction")
}

func postgresBackupCommand(dbEndpoint, username, password, backupPrefix, backupID string, profile []byte) ([]string, error) {

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
			dbEndpoint, backupPrefix, backupID, profile, ExportRDSSnapshotToLocBackupID),
	}
	return command, nil
}

func cleanupRDSDB(ctx context.Context, rdsCli *rds.RDS, instanceID string) error {
	// Deleting tmp instance
	log.Print("Delete temporary RDS instance.", field.M{"InstanceID": instanceID})
	if _, err := rdsCli.DeleteDBInstance(ctx, instanceID); err != nil {
		return err
	}

	// Wait until instance is deleted
	log.Print("Waiting for RDS DB instance to be deleted", field.M{"InstanceID": instanceID})
	return rdsCli.WaitUntilDBInstanceDeleted(ctx, instanceID)
}
