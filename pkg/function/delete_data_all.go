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

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// DeleteDataAllNamespaceArg provides the namespace
	DeleteDataAllNamespaceArg = "namespace"
	// DeleteDataAllBackupArtifactPrefixArg provides the path to restore backed up data
	DeleteDataAllBackupArtifactPrefixArg = "backupArtifactPrefix"
	// DeleteDataAllEncryptionKeyArg provides the encryption key to be used for deletes
	DeleteDataAllEncryptionKeyArg = "encryptionKey"
	// DeleteDataAllReclaimSpace provides a way to specify if space should be reclaimed
	DeleteDataAllReclaimSpace = "reclaimSpace"
	// DeleteDataAllBackupInfo provides backup info required for delete
	DeleteDataAllBackupInfo = "backupInfo"
	// DeleteDataAllPodOverrideArg contains pod specs to override default pod specs
	DeleteDataAllPodOverrideArg = "podOverride"
	deleteDataAllJobPrefix      = "delete-data-all-"
)

func init() {
	kanister.Register(&deleteDataAllFunc{})
}

var _ kanister.Func = (*deleteDataAllFunc)(nil)

type deleteDataAllFunc struct{}

func (*deleteDataAllFunc) Name() string {
	return "DeleteDataAll"
}

func (*deleteDataAllFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, deleteArtifactPrefix, backupInfo, encryptionKey string
	var reclaimSpace bool
	var err error
	if err = Arg(args, DeleteDataAllNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteDataAllBackupArtifactPrefixArg, &deleteArtifactPrefix); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteDataAllBackupInfo, &backupInfo); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataAllEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataAllReclaimSpace, &reclaimSpace, false); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, DeleteDataAllPodOverrideArg)
	if err != nil {
		return nil, err
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, err
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	input := make(map[string]BackupInfo)
	err = json.Unmarshal([]byte(backupInfo), &input)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not decode JSON data")
	}
	var targetPaths []string
	var deleteIdentifiers []string
	for pod, info := range input {
		targetPaths = append(targetPaths, fmt.Sprintf("%s/%s", deleteArtifactPrefix, pod))
		deleteIdentifiers = append(deleteIdentifiers, info.BackupID)
	}

	return deleteData(ctx, cli, tp, reclaimSpace, namespace, encryptionKey, targetPaths, nil, deleteIdentifiers, deleteDataAllJobPrefix, podOverride)
}

func (*deleteDataAllFunc) RequiredArgs() []string {
	return []string{DeleteDataAllNamespaceArg, DeleteDataAllBackupArtifactPrefixArg, DeleteDataAllBackupInfo}
}
