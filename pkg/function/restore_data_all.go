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

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	restoreDataAllJobPrefix = "restore-data-all-"
	// RestoreDataAllNamespaceArg provides the namespace
	RestoreDataAllNamespaceArg = "namespace"
	// RestoreDataAllImageArg provides the image of the container with required tools
	RestoreDataAllImageArg = "image"
	// RestoreDataAllBackupArtifactPrefixArg provides the path of the backed up artifact
	RestoreDataAllBackupArtifactPrefixArg = "backupArtifactPrefix"
	// RestoreDataAllRestorePathArg provides the path to restore backed up data
	RestoreDataAllRestorePathArg = "restorePath"
	// RestoreDataAllPodsArg provides the pod connected to the data volume
	RestoreDataAllPodsArg = "pods"
	// RestoreDataAllEncryptionKeyArg provides the encryption key used during backup
	RestoreDataAllEncryptionKeyArg = "encryptionKey"
	// RestoreDataAllBackupInfo provides backup info required for restore
	RestoreDataAllBackupInfo = "backupInfo"
)

func init() {
	kanister.Register(&restoreDataAllFunc{})
}

var _ kanister.Func = (*restoreDataAllFunc)(nil)

type restoreDataAllFunc struct{}

func (*restoreDataAllFunc) Name() string {
	return "RestoreDataAll"
}

func validateAndGetRestoreAllOptArgs(args map[string]interface{}, tp param.TemplateParams) (string, string, []string, error) {
	var restorePath, encryptionKey, pods string
	var ps []string
	var err error

	if err = OptArg(args, RestoreDataAllRestorePathArg, &restorePath, "/"); err != nil {
		return restorePath, encryptionKey, ps, err
	}
	if err = OptArg(args, RestoreDataAllEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return restorePath, encryptionKey, ps, err
	}
	if err = OptArg(args, RestoreDataAllPodsArg, &pods, ""); err != nil {
		return restorePath, encryptionKey, ps, err
	}

	if pods != "" {
		ps = strings.Fields(pods)
	} else {
		switch {
		case tp.Deployment != nil:
			ps = tp.Deployment.Pods
		case tp.StatefulSet != nil:
			ps = tp.StatefulSet.Pods
		default:
			return restorePath, encryptionKey, ps, errors.New("Unsupported workload type")
		}
	}

	return restorePath, encryptionKey, ps, nil
}

func (*restoreDataAllFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, image, backupArtifactPrefix, backupInfo string
	var err error
	if err = Arg(args, RestoreDataAllNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataAllImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataAllBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataAllBackupInfo, &backupInfo); err != nil {
		return nil, err
	}
	// Validate and get optional arguments
	restorePath, encryptionKey, pods, err := validateAndGetRestoreAllOptArgs(args, tp)
	if err != nil {
		return nil, err
	}
	// Validate profile
	if err = validateProfile(tp.Profile); err != nil {
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
	var chanLen = len(pods)
	errChan := make(chan error, chanLen)
	outputChan := make(chan map[string]interface{}, chanLen)
	output := make(map[string]interface{})
	for _, pod := range pods {
		go func(pod string) {
			vols, err := fetchPodVolumes(pod, tp)
			var out map[string]interface{}
			if err != nil {
				errChan <- errors.Wrapf(err, "Failed to get volumes of pod %s", pod)
				outputChan <- out
				return
			}
			out, err = restoreData(ctx, cli, tp, namespace, encryptionKey, fmt.Sprintf("%s/%s", backupArtifactPrefix, pod), restorePath, "", input[pod].BackupID, restoreDataAllJobPrefix, vols)
			errChan <- errors.Wrapf(err, "Failed to restore data for pod %s", pod)
			outputChan <- out
		}(pod)
	}
	errs := make([]string, 0, chanLen)
	for i := 0; i < chanLen; i++ {
		err := <-errChan
		out := <-outputChan
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			for k, v := range out {
				output[k] = v
			}
		}
	}
	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	return output, nil
}

func (*restoreDataAllFunc) RequiredArgs() []string {
	return []string{RestoreDataAllNamespaceArg, RestoreDataAllImageArg,
		RestoreDataAllBackupArtifactPrefixArg, RestoreDataAllBackupInfo}
}
