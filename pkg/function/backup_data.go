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

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// BackupDataFuncName gives the name of the function
	BackupDataFuncName = "BackupData"
	// BackupDataNamespaceArg provides the namespace
	BackupDataNamespaceArg = "namespace"
	// BackupDataPodArg provides the pod connected to the data volume
	BackupDataPodArg = "pod"
	// BackupDataContainerArg provides the container on which the backup is taken
	BackupDataContainerArg = "container"
	// BackupDataIncludePathArg provides the path of the volume or sub-path for required backup
	BackupDataIncludePathArg = "includePath"
	// BackupDataBackupArtifactPrefixArg provides the path to store artifacts on the object store
	BackupDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// BackupDataEncryptionKeyArg provides the encryption key to be used for backups
	BackupDataEncryptionKeyArg = "encryptionKey"
	// BackupDataOutputBackupID is the key used for returning backup ID output
	BackupDataOutputBackupID = "backupID"
	// BackupDataOutputBackupTag is the key used for returning backupTag output
	BackupDataOutputBackupTag = "backupTag"
	// BackupDataOutputBackupFileCount is the key used for returning backup file count
	BackupDataOutputBackupFileCount = "fileCount"
	// BackupDataOutputBackupSize is the key used for returning backup size
	BackupDataOutputBackupSize = "size"
)

func init() {
	kanister.Register(&backupDataFunc{})
}

var _ kanister.Func = (*backupDataFunc)(nil)

type backupDataFunc struct{}

func (*backupDataFunc) Name() string {
	return BackupDataFuncName
}

func (*backupDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, pod, container, includePath, backupArtifactPrefix, encryptionKey string
	var err error
	if err = Arg(args, BackupDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataPodArg, &pod); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataContainerArg, &container); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataIncludePathArg, &includePath); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}

	backupArtifactPrefix = ResolveArtifactPrefix(backupArtifactPrefix, tp.Profile)

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	ctx = field.Context(ctx, consts.PodNameKey, pod)
	ctx = field.Context(ctx, consts.ContainerNameKey, container)
	backupOutputs, err := backupData(ctx, cli, namespace, pod, container, backupArtifactPrefix, includePath, encryptionKey, tp)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to backup data")
	}
	output := map[string]interface{}{
		BackupDataOutputBackupID:        backupOutputs.backupID,
		BackupDataOutputBackupTag:       backupOutputs.backupTag,
		BackupDataOutputBackupFileCount: backupOutputs.fileCount,
		BackupDataOutputBackupSize:      backupOutputs.backupSize,
	}
	return output, nil
}

func (*backupDataFunc) RequiredArgs() []string {
	return []string{BackupDataNamespaceArg, BackupDataPodArg, BackupDataContainerArg,
		BackupDataIncludePathArg, BackupDataBackupArtifactPrefixArg}
}

type backupDataParsedOutput struct {
	backupID   string
	backupTag  string
	fileCount  string
	backupSize string
}

func backupData(ctx context.Context, cli kubernetes.Interface, namespace, pod, container, backupArtifactPrefix, includePath, encryptionKey string, tp param.TemplateParams) (backupDataParsedOutput, error) {
	pw, err := GetPodWriter(cli, ctx, namespace, pod, container, tp.Profile)
	if err != nil {
		return backupDataParsedOutput{}, err
	}
	defer CleanUpCredsFile(ctx, pw, namespace, pod, container)
	if err = restic.GetOrCreateRepository(cli, namespace, pod, container, backupArtifactPrefix, encryptionKey, tp.Profile); err != nil {
		return backupDataParsedOutput{}, err
	}

	// Create backup and dump it on the object store
	backupTag := rand.String(10)
	cmd, err := restic.BackupCommandByTag(tp.Profile, backupArtifactPrefix, backupTag, includePath, encryptionKey)
	if err != nil {
		return backupDataParsedOutput{}, err
	}
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err != nil {
		return backupDataParsedOutput{}, errors.Wrapf(err, "Failed to create and upload backup")
	}
	// Get the snapshot ID from log
	backupID := restic.SnapshotIDFromBackupLog(stdout)
	if backupID == "" {
		return backupDataParsedOutput{}, errors.New("Failed to parse the backup ID from logs")
	}
	// Get the file count and size of the backup from log
	fileCount, backupSize := restic.SnapshotStatsFromBackupLog(stdout)
	if fileCount == "" || backupSize == "" {
		log.Debug().Print("Could not parse backup stats from backup log")
	}
	return backupDataParsedOutput{
		backupID:   backupID,
		backupTag:  backupTag,
		fileCount:  fileCount,
		backupSize: backupSize}, nil
}
