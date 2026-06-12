// Copyright 2026 The Kanister Authors.
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
	"time"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	LabelKeyIncludeInBackup   = "kasten.io/include-in-backup"
	LabelKeyOwnerAction       = "kanister.io/owner-action"
	LabelKeyStagingPVC        = "kanister.io/staging-pvc"
	LabelKeyWorkloadName      = "kanister.io/workload-name"
	LabelKeyWorkloadNamespace = "kanister.io/workload-namespace"
	LabelKeySnapshotCloned    = "kanister.io/snapshot-cloned"

	OutputKeyStagingPVCName      = "pvcName"
	OutputKeyStagingPVCNamespace = "namespace"
	OutputKeySnapshotName        = "volumeSnapshotName"
	OutputKeySnapshotNamespace   = "volumeSnapshotNamespace"
	OutputKeySnapshotRestoreSize = "restoreSize"
	OutputKeySnapshotContent     = "snapshotContent"
	OutputKeySnapshotHandle      = "snapshotHandle"

	backupCSIDriverName = "backup.csi.kastenhq.io"
	kanisterClonePrefix = "kanister-clone-"

	cleanupSnapshotTimeout = 60 * time.Second

	// SnapshotArtifactKey, ArtifactKeyBackupIdentifier, ArtifactKeySize mirror
	// K10's kanconsts so the restore function can read tp.ArtifactsIn without
	// the blueprint re-templating each keyValue field as a function arg.
	SnapshotArtifactKey         = "snapshot"
	ArtifactKeyBackupIdentifier = "backupIdentifier"
	ArtifactKeySize             = "size"
)

func actionSetTagFromContext(ctx context.Context) string {
	fields := field.FromContext(ctx)
	if fields == nil {
		return ""
	}
	for _, f := range fields.Fields() {
		if f.Key() == consts.ActionsetNameKey {
			if v, ok := f.Value().(string); ok {
				return v
			}
		}
	}
	return ""
}

func workloadFromTemplateParams(tp param.TemplateParams) (name, namespace string) {
	switch {
	case tp.StatefulSet != nil:
		return tp.StatefulSet.Name, tp.StatefulSet.Namespace
	case tp.Deployment != nil:
		return tp.Deployment.Name, tp.Deployment.Namespace
	case tp.DeploymentConfig != nil:
		return tp.DeploymentConfig.Name, tp.DeploymentConfig.Namespace
	case tp.PVC != nil:
		return tp.PVC.Name, tp.PVC.Namespace
	case tp.Namespace != nil:
		return "", tp.Namespace.Namespace
	}
	return "", ""
}

func waitForPVCBound(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, errkit.Wrap(err, "Staging PVC disappeared while waiting for Bound", "namespace", namespace, "pvcName", name)
			}
			return false, err
		}
		switch pvc.Status.Phase {
		case corev1.ClaimBound:
			return true, nil
		case corev1.ClaimLost:
			return false, errkit.New("Staging PVC entered Lost phase", "namespace", namespace, "pvcName", name)
		}
		return false, nil
	})
}

// stringifyMapKeys recursively converts map[interface{}]interface{} (YAML
// decoder output) into map[string]interface{} so the result is safe to hand
// to encoding/json. Standard YAML→JSON bridge.
func stringifyMapKeys(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(x))
		for k, vv := range x {
			m[fmt.Sprint(k)] = stringifyMapKeys(vv)
		}
		return m
	case map[string]interface{}:
		for k, vv := range x {
			x[k] = stringifyMapKeys(vv)
		}
		return x
	case []interface{}:
		for i, vv := range x {
			x[i] = stringifyMapKeys(vv)
		}
		return x
	default:
		return v
	}
}

// parseEnvVars decodes a Kanister arg holding a []corev1.EnvVar. The shared
// arg-parser (mapstructure.WeakDecode) drops embedded LocalObjectReference
// inside `valueFrom.secretKeyRef`/`valueFrom.configMapKeyRef`, so we route
// through json.Marshal/Unmarshal which correctly traverses inline embedded
// structs.
func parseEnvVars(args map[string]interface{}, argName string) ([]corev1.EnvVar, error) {
	raw, ok := args[argName]
	if !ok || raw == nil {
		return nil, nil
	}
	blob, err := json.Marshal(stringifyMapKeys(raw))
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to marshal env arg for re-decode", "arg", argName)
	}
	var env []corev1.EnvVar
	if err := json.Unmarshal(blob, &env); err != nil {
		return nil, errkit.Wrap(err, "Failed to decode env arg as []corev1.EnvVar", "arg", argName)
	}
	return env, nil
}

// pvcGracefulDelete deletes a PVC, ignoring NotFound. Uses a fresh context so
// cleanup still runs when the caller's ctx is cancelled.
func pvcGracefulDelete(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	delCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	err := cli.CoreV1().PersistentVolumeClaims(namespace).Delete(delCtx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// stagingPodRunner waits for the pod to be ready, streams + parses its logs,
// then waits for completion. failMsg is the wrapper for the completion error.
func stagingPodRunner(failMsg string) func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errkit.Wrap(err, "Failed while waiting for pod to be ready", "pod", pc.PodName())
		}
		ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
		r, err := pc.StreamPodLogs(ctx)
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to fetch pod logs", "pod", pc.PodName())
		}
		out, err := output.LogAndParse(ctx, r)
		if err != nil {
			return nil, err
		}
		if err := pc.WaitForPodCompletion(ctx); err != nil {
			return nil, errkit.Wrap(err, failMsg, "pod", pc.PodName())
		}
		return out, nil
	}
}
