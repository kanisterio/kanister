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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	restoreDataJobPrefix = "restore-data-"
	// RestoreDataFuncName gives the function name
	RestoreDataFuncName = "RestoreData"
	// RestoreDataNamespaceArg provides the namespace
	RestoreDataNamespaceArg = "namespace"
	// RestoreDataImageArg provides the image of the container with required tools
	RestoreDataImageArg = "image"
	// RestoreDataBackupArtifactPrefixArg provides the path of the backed up artifact
	RestoreDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// RestoreDataRestorePathArg provides the path to restore backed up data
	RestoreDataRestorePathArg = "restorePath"
	// RestoreDataBackupIdentifierArg provides a unique ID added to the backed up artifacts
	RestoreDataBackupIdentifierArg = "backupIdentifier"
	// RestoreDataPodArg provides the pod connected to the data volume
	RestoreDataPodArg = "pod"
	// RestoreDataVolsArg provides a map of PVC->mountPaths to be attached
	RestoreDataVolsArg = "volumes"
	// RestoreDataEncryptionKeyArg provides the encryption key used during backup
	RestoreDataEncryptionKeyArg = "encryptionKey"
	// RestoreDataBackupTagArg provides a unique tag added to the backup artifacts
	RestoreDataBackupTagArg = "backupTag"
	// RestoreDataPodOverrideArg contains pod specs which overrides default pod specs
	RestoreDataPodOverrideArg = "podOverride"
)

func init() {
	kanister.Register(&restoreDataFunc{})
}

var _ kanister.Func = (*restoreDataFunc)(nil)

type restoreDataFunc struct{}

func (*restoreDataFunc) Name() string {
	return RestoreDataFuncName
}

func validateAndGetOptArgs(args map[string]interface{}, tp param.TemplateParams) (string, string, string, map[string]string, string, string, crv1alpha1.JSONMap, error) {
	var restorePath, encryptionKey, pod, tag, id string
	var vols map[string]string
	var podOverride crv1alpha1.JSONMap
	var err error

	if err = OptArg(args, RestoreDataRestorePathArg, &restorePath, "/"); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride, err
	}
	if err = OptArg(args, RestoreDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride, err
	}
	if err = OptArg(args, RestoreDataPodArg, &pod, ""); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride, err
	}
	if err = OptArg(args, RestoreDataVolsArg, &vols, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride, err
	}
	if (pod != "") == (len(vols) > 0) {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride,
			errors.Errorf("Require one argument: %s or %s", RestoreDataPodArg, RestoreDataVolsArg)
	}
	if err = OptArg(args, RestoreDataBackupTagArg, &tag, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride, err
	}
	if err = OptArg(args, RestoreDataBackupIdentifierArg, &id, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride, err
	}
	if (tag != "") == (id != "") {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride,
			errors.Errorf("Require one argument: %s or %s", RestoreDataBackupTagArg, RestoreDataBackupIdentifierArg)
	}
	podOverride, err = GetPodSpecOverride(tp, args, RestoreDataPodOverrideArg)
	if err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, podOverride, err
	}

	return restorePath, encryptionKey, pod, vols, tag, id, podOverride, nil
}

func restoreData(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID, jobPrefix, image string,
	vols map[string]string, podOverride crv1alpha1.JSONMap) (map[string]interface{}, error) {
	// Validate volumes
	for pvc := range vols {
		if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(pvc, metav1.GetOptions{}); err != nil {
			return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvc)
		}
	}
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      vols,
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := restoreDataPodFunc(cli, tp, namespace, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID)
	return pr.Run(ctx, podFunc)
}

func restoreDataPodFunc(cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID string) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
	return func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
		// Wait for pod to reach running state
		if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}
		pw, err := GetPodWriter(cli, ctx, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, tp.Profile)
		if err != nil {
			return nil, err
		}
		defer CleanUpCredsFile(ctx, pw, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name)
		var cmd []string
		// Generate restore command based on the identifier passed
		if backupTag != "" {
			cmd, err = restic.RestoreCommandByTag(tp.Profile, backupArtifactPrefix, backupTag, restorePath, encryptionKey)
		} else if backupID != "" {
			cmd, err = restic.RestoreCommandByID(tp.Profile, backupArtifactPrefix, backupID, restorePath, encryptionKey)
		}
		if err != nil {
			return nil, err
		}
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to restore backup")
		}
		out, err := parseLogAndCreateOutput(stdout)
		return out, errors.Wrap(err, "Failed to parse phase output")
	}
}

func (*restoreDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, image, backupArtifactPrefix, backupTag, backupID string
	var podOverride crv1alpha1.JSONMap
	var err error
	if err = Arg(args, RestoreDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}

	// Validate and get optional arguments
	restorePath, encryptionKey, pod, vols, backupTag, backupID, podOverride, err := validateAndGetOptArgs(args, tp)
	if err != nil {
		return nil, err
	}
	if podOverride == nil {
		podOverride = tp.PodOverride
	}

	// Check if PodOverride specs are passed through actionset
	// If yes, override podOverride specs
	if tp.PodOverride != nil {
		podOverride, err = kube.CreateAndMergeJsonPatch(podOverride, tp.PodOverride)
		if err != nil {
			return nil, err
		}
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, err
	}
	if len(vols) == 0 {
		// Fetch Volumes
		vols, err = FetchPodVolumes(pod, tp)
		if err != nil {
			return nil, err
		}
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return restoreData(ctx, cli, tp, namespace, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID, restoreDataJobPrefix, image, vols, podOverride)
}

func (*restoreDataFunc) RequiredArgs() []string {
	return []string{RestoreDataNamespaceArg, RestoreDataImageArg,
		RestoreDataBackupArtifactPrefixArg}
}
