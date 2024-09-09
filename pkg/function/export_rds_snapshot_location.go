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
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/yaml"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws/rds"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/postgres"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
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
	ExportRDSSnapshotToLocDatabasesArg       = "databases"
	ExportRDSSnapshotToLocSecGrpIDArg        = "securityGroupID"
	ExportRDSSnapshotToLocBackupID           = "backupID"
	ExportRDSSnapshotToLocDBSubnetGroupArg   = "dbSubnetGroup"
	ExportRDSSnapshotToLocImageArg           = "image"

	PostgrSQLEngine RDSDBEngine = "PostgreSQL"

	BackupAction  RDSAction = "backup"
	RestoreAction RDSAction = "restore"

	defaultPostgresToolsImage = "ghcr.io/kanisterio/postgres-kanister-tools:0.110.0"
)

type exportRDSSnapshotToLocationFunc struct {
	progressPercent string
}

// RDSDBEngine for RDS Engine types
type RDSDBEngine string

// RDSAction for RDS backup or restore actions
type RDSAction string

func (*exportRDSSnapshotToLocationFunc) Name() string {
	return ExportRDSSnapshotToLocFuncName
}

func exportRDSSnapshotToLoc(
	ctx context.Context,
	namespace,
	instanceID,
	snapshotID,
	username,
	password string,
	databases []string,
	dbSubnetGroup,
	backupPrefix string,
	dbEngine RDSDBEngine,
	sgIDs []string,
	profile *param.Profile,
	postgresToolsImage string,
	annotations,
	labels map[string]string,
) (map[string]interface{}, error) {
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

	// If securityGroupID arg is nil, we will try to find the sgIDs by describing the existing instance
	if sgIDs == nil {
		sgIDs, err = findSecurityGroups(ctx, rdsCli, instanceID)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch security group ids. InstanceID=%s", instanceID)
		}
	}

	log.WithContext(ctx).Print("Spin up temporary RDS instance from the snapshot.", field.M{"SnapshotID": snapshotID, "InstanceID": tmpInstanceID})
	// Create tmp instance from snapshot
	if err := restoreFromSnapshot(ctx, rdsCli, tmpInstanceID, dbSubnetGroup, snapshotID, sgIDs); err != nil {
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

	// get the engine version
	dbEngineVersion, err := rdsDBEngineVersion(ctx, rdsCli, tmpInstanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't find DBInstance Version")
	}

	// Extract dump from DB
	output, err := execDumpCommand(
		ctx,
		dbEngine,
		BackupAction,
		namespace,
		dbEndpoint,
		username,
		password,
		databases,
		backupPrefix,
		backupID,
		profile,
		dbEngineVersion,
		postgresToolsImage,
		annotations,
		labels,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to extract and push db dump to location")
	}

	// Convert to yaml format
	sgIDYaml, err := yaml.Marshal(sgIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create securityGroupID artifact. InstanceID=%s", tmpInstanceID)
	}

	// Add output artifacts
	output[ExportRDSSnapshotToLocSnapshotIDArg] = snapshotID
	output[ExportRDSSnapshotToLocInstanceIDArg] = instanceID
	output[ExportRDSSnapshotToLocSecGrpIDArg] = string(sgIDYaml)

	return output, nil
}

func (e *exportRDSSnapshotToLocationFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	e.progressPercent = progress.StartedPercent
	defer func() { e.progressPercent = progress.CompletedPercent }()

	var namespace, instanceID, snapshotID, username, password, dbSubnetGroup, backupArtifact, postgresToolsImage string
	var dbEngine RDSDBEngine
	var bpAnnotations, bpLabels map[string]string

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
	if err := OptArg(args, ExportRDSSnapshotToLocDBSubnetGroupArg, &dbSubnetGroup, "default"); err != nil {
		return nil, err
	}
	if err := OptArg(args, ExportRDSSnapshotToLocImageArg, &postgresToolsImage, defaultPostgresToolsImage); err != nil {
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

	// Find databases
	databases, err := GetYamlList(args, ExportRDSSnapshotToLocDatabasesArg)
	if err != nil {
		return nil, err
	}

	// Find security groups
	sgIDs, err := GetYamlList(args, ExportRDSSnapshotToLocSecGrpIDArg)
	if err != nil {
		return nil, err
	}

	return exportRDSSnapshotToLoc(
		ctx,
		namespace,
		instanceID,
		snapshotID,
		username,
		password,
		databases,
		dbSubnetGroup,
		backupArtifact,
		dbEngine,
		sgIDs,
		tp.Profile,
		postgresToolsImage,
		annotations,
		labels,
	)
}

func (*exportRDSSnapshotToLocationFunc) RequiredArgs() []string {
	return []string{
		ExportRDSSnapshotToLocNamespaceArg,
		ExportRDSSnapshotToLocInstanceIDArg,
		ExportRDSSnapshotToLocSnapshotIDArg,
		ExportRDSSnapshotToLocDBEngineArg,
	}
}

func (*exportRDSSnapshotToLocationFunc) Arguments() []string {
	return []string{
		ExportRDSSnapshotToLocNamespaceArg,
		ExportRDSSnapshotToLocInstanceIDArg,
		ExportRDSSnapshotToLocSnapshotIDArg,
		ExportRDSSnapshotToLocDBEngineArg,
		ExportRDSSnapshotToLocDBUsernameArg,
		ExportRDSSnapshotToLocDBPasswordArg,
		ExportRDSSnapshotToLocBackupArtPrefixArg,
		ExportRDSSnapshotToLocDatabasesArg,
		ExportRDSSnapshotToLocSecGrpIDArg,
		ExportRDSSnapshotToLocDBSubnetGroupArg,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (e *exportRDSSnapshotToLocationFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(e.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(e.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(e.RequiredArgs(), args)
}

func (e *exportRDSSnapshotToLocationFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    e.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func execDumpCommand(
	ctx context.Context,
	dbEngine RDSDBEngine,
	action RDSAction,
	namespace,
	dbEndpoint,
	username,
	password string,
	databases []string,
	backupPrefix,
	backupID string,
	profile *param.Profile,
	dbEngineVersion string,
	postgresToolsImage string,
	annotations,
	labels map[string]string,
) (map[string]interface{}, error) {
	// Trim "\n" from creds
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	// Prepare and execute command with kubetask
	command, err := prepareCommand(ctx, dbEngine, action, dbEndpoint, username, password, databases, backupPrefix, backupID, profile, dbEngineVersion)
	if err != nil {
		return nil, err
	}

	// Create Kubernetes client
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	// Create cred secret
	secretName := fmt.Sprintf("%s-%s", "postgres-secret", rand.String(10))
	err = createPostgresSecret(cli, secretName, namespace, username, password)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create postgres secret")
	}

	defer func() {
		if err := deletePostgresSecret(cli, secretName, namespace); err != nil {
			log.Error().WithError(err).Print("Failed to cleanup postgres-secret")
		}
	}()

	return kubeTask(ctx, cli, namespace, postgresToolsImage, command, injectPostgresSecrets(secretName), annotations, labels)
}

func prepareCommand(
	ctx context.Context,
	dbEngine RDSDBEngine,
	action RDSAction,
	dbEndpoint,
	username,
	password string,
	dbList []string,
	backupPrefix,
	backupID string,
	profile *param.Profile,
	dbEngineVersion string,
) ([]string, error) {
	// Convert profile object into json
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}

	if dbEngine == PostgrSQLEngine {
		switch action {
		case BackupAction:
			// For backup operation, if database arg is not set, we take backup of all databases
			if dbList == nil {
				dbList, err = findDBList(ctx, dbEndpoint, username, password)
				if err != nil {
					return nil, err
				}
			}
			command, err := postgresBackupCommand(dbEndpoint, username, password, dbList, backupPrefix, backupID, profileJSON)
			return command, err
		case RestoreAction:
			command, err := postgresRestoreCommand(dbEndpoint, username, password, backupPrefix, backupID, profileJSON, dbEngineVersion)
			return command, err
		}
	}
	return nil, errors.New("Invalid RDSDBEngine or RDSAction")
}

func findDBList(ctx context.Context, dbEndpoint, username, password string) ([]string, error) {
	pg, err := postgres.NewClient(dbEndpoint, username, password, postgres.DefaultConnectDatabase, "disable")
	if err != nil {
		return nil, errors.Wrap(err, "Error in creating postgres client")
	}

	// Test DB connection
	if err := pg.PingDB(ctx); err != nil {
		return nil, errors.Wrap(err, "Failed to ping postgres database")
	}

	dbList, err := pg.ListDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error while listing databases")
	}
	return filterRestrictedDB(dbList), nil
}

//nolint:unparam
func postgresBackupCommand(dbEndpoint, username, password string, dbList []string, backupPrefix, backupID string, profile []byte) ([]string, error) {
	if len(dbList) == 0 {
		return nil, errors.New("No database found to backup")
	}

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
			dblist=( %s )
			for db in "${dblist[@]}";
			  do echo "backing up $db db" && pg_dump $db -C --inserts > /backup/$db.sql;
			done
			tar -zc backup | kando location push --profile '%s' --path "${BACKUP_PREFIX}/${BACKUP_ID}" -
			kando output %s ${BACKUP_ID}`,
			dbEndpoint, backupPrefix, backupID, strings.Join(dbList, " "), profile, ExportRDSSnapshotToLocBackupID),
	}
	return command, nil
}

func cleanupRDSDB(ctx context.Context, rdsCli *rds.RDS, instanceID string) error {
	// Deleting tmp instance
	log.WithContext(ctx).Print("Delete temporary RDS instance.", field.M{"InstanceID": instanceID})
	if _, err := rdsCli.DeleteDBInstance(ctx, instanceID); err != nil {
		return err
	}

	// Wait until instance is deleted
	log.WithContext(ctx).Print("Waiting for RDS DB instance to be deleted", field.M{"InstanceID": instanceID})
	return rdsCli.WaitUntilDBInstanceDeleted(ctx, instanceID)
}

func filterRestrictedDB(dbList []string) []string {
	// Map of restricted DBs
	restricted := map[string]bool{
		"template0": true,
		"rdsadmin":  true,
	}

	var validDBs []string
	for _, db := range dbList {
		if !restricted[db] {
			validDBs = append(validDBs, db)
		}
	}
	return validDBs
}

func injectPostgresSecrets(secretName string) crv1alpha1.JSONMap {
	return crv1alpha1.JSONMap{
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
}
