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
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
)

// Label keys used by KubeTaskWithBackupPVC and KubeTaskWithRestorePVC to
// coordinate with Kasten's snapshot phase and to discover the restored
// staging PVC.
const (
	// LabelKeyIncludeInBackup signals Kasten's post-prehook discovery pass that
	// this PVC should be added to the current ActionSet's snapshot set.
	LabelKeyIncludeInBackup = "kasten.io/include-in-backup"

	// LabelKeyOwnerAction scopes the rescan to a specific ActionSet (carries
	// the ActionSet UID so a stale label from a previous run is ignored).
	LabelKeyOwnerAction = "kanister.io/owner-action"

	// LabelKeyStagingPVC marks the PVC as a Kanister-owned staging volume.
	// KubeTaskWithRestorePVC uses this to find the PVC after Kasten restores it.
	LabelKeyStagingPVC = "kanister.io/staging-pvc"

	// LabelKeyWorkloadName / LabelKeyWorkloadNamespace narrow the restore
	// discovery to a single workload so namespaces hosting multiple staged
	// workloads do not get cross-matched.
	LabelKeyWorkloadName      = "kanister.io/workload-name"
	LabelKeyWorkloadNamespace = "kanister.io/workload-namespace"

	// Output keys returned by KubeTaskWithBackupPVC so a downstream cleanup
	// phase can reference the staging PVC.
	OutputKeyStagingPVCName      = "pvcName"
	OutputKeyStagingPVCNamespace = "namespace"

	// Additional output keys returned by KubeTaskWithBackupPVC when
	// `takeSnapshot=true`. Schema mirrors CreateCSISnapshot's output so an
	// `actions.backup` blueprint can render a snapshotInfo artifact identical
	// in shape to what CreateCSISnapshot produces today.
	OutputKeySnapshotName            = "volumeSnapshotName"
	OutputKeySnapshotNamespace       = "volumeSnapshotNamespace"
	OutputKeySnapshotRestoreSize     = "restoreSize"
	OutputKeySnapshotContent         = "snapshotContent"
)

// actionSetTagFromContext extracts the ActionSet identifier the controller
// already injects into the phase context via the standard `ActionsetNameKey`
// (Kanister core plumbs this; we don't need a new UID-keyed const). Returns
// the empty string if absent (typically only in unit tests where the
// controller has not framed the context).
//
// We use the ActionSet *name* as the owner-action label value. Names CAN
// theoretically be reused after an ActionSet is GC'd, but in practice
// `generateName` produces unique suffixes and the staging PVCs created with
// this label are short-lived (deleted at function exit). Using the name
// instead of the UID keeps all our changes confined to `pkg/function/`
// without touching Kanister core's `pkg/consts`, `pkg/controller`, or
// `pkg/kube` packages.
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

// workloadFromTemplateParams returns the name and namespace of the workload
// the action is targeting. Falls back to the first non-nil workload params
// in the order StatefulSet → Deployment → DeploymentConfig → PVC → Namespace.
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

// waitForPVCBound polls until the PVC reaches phase Bound, the PVC is deleted,
// or ctx is cancelled (including timeout).
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

// stringifyMapKeys recursively converts map[interface{}]interface{} (the shape
// produced by YAML decoders that allow non-string keys) into
// map[string]interface{} so the result is safe to hand to encoding/json.
// This is the standard YAML→JSON bridge.
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
// arg-parser (mapstructure.WeakDecode) does not follow JSON's `,inline` tag and
// silently drops the embedded LocalObjectReference inside
// `valueFrom.secretKeyRef` / `valueFrom.configMapKeyRef`, which means a blueprint
// like `env: [{valueFrom: {secretKeyRef: {name: pg, key: postgres-password}}}]`
// reaches the Kubernetes API with an empty secretKeyRef.name and is rejected.
// Going through json.Marshal/Unmarshal correctly traverses inline embedded
// structs; we just need to first stringify any non-string YAML map keys.
func parseEnvVars(args map[string]interface{}, argName string) ([]corev1.EnvVar, error) {
	raw, ok := args[argName]
	if !ok || raw == nil {
		return nil, nil
	}
	normalized := stringifyMapKeys(raw)
	blob, err := json.Marshal(normalized)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to marshal env arg for re-decode", "arg", argName)
	}
	var env []corev1.EnvVar
	if err := json.Unmarshal(blob, &env); err != nil {
		return nil, errkit.Wrap(err, "Failed to decode env arg as []corev1.EnvVar", "arg", argName)
	}
	return env, nil
}

// pvcGracefulDelete deletes a PVC; ignores NotFound so callers can defer it
// without checking whether the PVC was already cleaned up.
func pvcGracefulDelete(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	// Use a fresh context for cleanup so we still attempt deletion when the
	// caller's context has been cancelled (e.g. on phase timeout).
	delCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	err := cli.CoreV1().PersistentVolumeClaims(namespace).Delete(delCtx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
