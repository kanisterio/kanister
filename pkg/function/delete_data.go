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
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// DeleteDataNamespaceArg provides the namespace
	DeleteDataNamespaceArg = "namespace"
	// DeleteDataBackupArtifactPrefixArg provides the path to restore backed up data
	DeleteDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// DeleteDataBackupIdentifierArg provides a unique ID added to the backed up artifacts
	DeleteDataBackupIdentifierArg = "backupID"
	// DeleteDataBackupTagArg provides a unique tag added to the backed up artifacts
	DeleteDataBackupTagArg = "backupTag"
	// DeleteDataEncryptionKeyArg provides the encryption key to be used for deletes
	DeleteDataEncryptionKeyArg = "encryptionKey"
	// DeleteDataReclaimSpace provides a way to specify if space should be reclaimed
	DeleteDataReclaimSpace = "reclaimSpace"
	// DeleteDataPodOverrideArg contains pod specs to override default pod specs
	DeleteDataPodOverrideArg = "podOverride"
	deleteDataJobPrefix      = "delete-data-"
)

func init() {
	kanister.Register(&deleteDataFunc{})
}

var _ kanister.Func = (*deleteDataFunc)(nil)

type deleteDataFunc struct{}

func (*deleteDataFunc) Name() string {
	return "DeleteData"
}

func deleteData(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, reclaimSpace bool, namespace, encryptionKey string, targetPaths, deleteTags, deleteIdentifiers []string, jobPrefix string, podOverride map[string]interface{}) (map[string]interface{}, error) {
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        kanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := deleteDataPodFunc(cli, tp, reclaimSpace, namespace, encryptionKey, targetPaths, deleteTags, deleteIdentifiers)
	return pr.Run(ctx, podFunc)
}

func deleteDataPodFunc(cli kubernetes.Interface, tp param.TemplateParams, reclaimSpace bool, namespace, encryptionKey string, targetPaths, deleteTags, deleteIdentifiers []string) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
	return func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
		// Wait for pod to reach running state
		if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}
		if (len(deleteIdentifiers) == 0) == (len(deleteTags) == 0) {
			return nil, errors.Errorf("Require one argument: %s or %s", DeleteDataBackupIdentifierArg, DeleteDataBackupTagArg)
		}
		pw, err := getPodWriter(cli, ctx, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, tp.Profile)
		if err != nil {
			return nil, err
		}
		defer cleanUpCredsFile(ctx, pw, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name)
		for i, deleteTag := range deleteTags {
			cmd, err := restic.SnapshotsCommandByTag(tp.Profile, targetPaths[i], deleteTag, encryptionKey)
			if err != nil {
				return nil, err
			}
			stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
			format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
			format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to forget data, could not get snapshotID from tag, Tag: %s", deleteTag)
			}
			deleteIdentifier, err := restic.SnapshotIDFromSnapshotLog(stdout)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to forget data, could not get snapshotID from tag, Tag: %s", deleteTag)
			}
			deleteIdentifiers = append(deleteIdentifiers, deleteIdentifier)
		}
		for i, deleteIdentifier := range deleteIdentifiers {
			cmd, err := restic.ForgetCommandByID(tp.Profile, targetPaths[i], deleteIdentifier, encryptionKey)
			if err != nil {
				return nil, err
			}
			stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
			format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
			format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to forget data")
			}
			if reclaimSpace {
				err := pruneData(cli, tp, pod, namespace, encryptionKey, targetPaths[i])
				if err != nil {
					return nil, errors.Wrapf(err, "Error executing prune command")
				}
			}
		}

		return nil, nil
	}
}

func pruneData(cli kubernetes.Interface, tp param.TemplateParams, pod *v1.Pod, namespace, encryptionKey, targetPath string) error {
	cmd, err := restic.PruneCommand(tp.Profile, targetPath, encryptionKey)
	if err != nil {
		return err
	}
	stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
	format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
	format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
	return errors.Wrapf(err, "Failed to prune data after forget")
}

func (*deleteDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, deleteArtifactPrefix, deleteIdentifier, deleteTag, encryptionKey string
	var reclaimSpace bool
	var podOverride map[string]interface{}
	var err error
	if err = Arg(args, DeleteDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteDataBackupArtifactPrefixArg, &deleteArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataBackupIdentifierArg, &deleteIdentifier, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataBackupTagArg, &deleteTag, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataReclaimSpace, &reclaimSpace, false); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataPodOverrideArg, &podOverride, podOverride); err != nil {
		return nil, err
	}

	// Check if PodOverride specs are passed through actionset
	// If yes, override podOverride specs
	if tp.PodOverride != nil {
		podOverride, err = kube.CreateAndMergeJsonPatch(podOverride, tp.PodOverride)
		if err != nil {
			return nil, err
		}
	}

	// Validate profile
	if err = validateProfile(tp.Profile); err != nil {
		return nil, err
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return deleteData(ctx, cli, tp, reclaimSpace, namespace, encryptionKey, strings.Fields(deleteArtifactPrefix), strings.Fields(deleteTag), strings.Fields(deleteIdentifier), deleteDataJobPrefix, podOverride)
}

func (*deleteDataFunc) RequiredArgs() []string {
	return []string{DeleteDataNamespaceArg, DeleteDataBackupArtifactPrefixArg}
}
