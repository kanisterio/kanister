// Copyright 2024 The Kanister Authors.
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
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&waitForEphemeralSnapshotFunc{})
}

var _ kanister.Func = (*waitForEphemeralSnapshotFunc)(nil)

const (
	WaitForEphemeralSnapshotFuncName        = "WaitForEphemeralSnapshot"
	WaitForEphemeralSnapshotNamespaceArg    = "namespace"
	WaitForEphemeralSnapshotAfterArg        = "after"
	WaitForEphemeralSnapshotPodNameArg      = "podName"
	WaitForEphemeralSnapshotNameOutput      = "volumeSnapshotName"
	WaitForEphemeralSnapshotNamespaceOutput = "volumeSnapshotNamespace"

	ephemeralSnapshotPollInterval = 5 * time.Second
	ephemeralSnapshotPollTimeout  = 5 * time.Minute
)

type waitForEphemeralSnapshotFunc struct {
	progressPercent string
}

func (*waitForEphemeralSnapshotFunc) Name() string {
	return WaitForEphemeralSnapshotFuncName
}

func (w *waitForEphemeralSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	w.progressPercent = progress.StartedPercent
	defer func() { w.progressPercent = progress.CompletedPercent }()

	var namespace string
	var afterStr, podName string

	if err := Arg(args, WaitForEphemeralSnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	// Default: use the ActionSet start time so blueprint authors don't have to pass it.
	defaultAfter := tp.Time
	if err := OptArg(args, WaitForEphemeralSnapshotAfterArg, &afterStr, defaultAfter); err != nil {
		return nil, err
	}
	if err := OptArg(args, WaitForEphemeralSnapshotPodNameArg, &podName, ""); err != nil {
		return nil, err
	}

	after, err := time.Parse(time.RFC3339, afterStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'after' timestamp %q: %w", afterStr, err)
	}

	snapName, err := findEphemeralVolumeSnapshot(ctx, namespace, after, podName)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		WaitForEphemeralSnapshotNameOutput:      snapName,
		WaitForEphemeralSnapshotNamespaceOutput: namespace,
	}, nil
}

func (*waitForEphemeralSnapshotFunc) RequiredArgs() []string {
	return []string{
		WaitForEphemeralSnapshotNamespaceArg,
	}
}

func (*waitForEphemeralSnapshotFunc) Arguments() []string {
	return []string{
		WaitForEphemeralSnapshotNamespaceArg,
		WaitForEphemeralSnapshotAfterArg,
		WaitForEphemeralSnapshotPodNameArg,
	}
}

func (w *waitForEphemeralSnapshotFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(w.Arguments(), args); err != nil {
		return err
	}
	return utils.CheckRequiredArgs(w.RequiredArgs(), args)
}

func (w *waitForEphemeralSnapshotFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    w.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

// findEphemeralVolumeSnapshot polls for the VolumeSnapshot auto-created by the CSI driver
// during NodeUnpublishVolume of an ephemeral inline CSI volume.
//
// Polling is required because NodeUnpublishVolume runs asynchronously on the node after
// the KubeTask pod exits — the snapshot may not exist yet when this function first runs.
func findEphemeralVolumeSnapshot(ctx context.Context, namespace string, after time.Time, podName string) (string, error) {
	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return "", err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, ephemeralSnapshotPollTimeout)
	defer cancel()

	for {
		name, err := listEphemeralVolumeSnapshot(timeoutCtx, dynCli, namespace, after, podName)
		if err == nil {
			return name, nil
		}

		select {
		case <-timeoutCtx.Done():
			return "", fmt.Errorf("timed out waiting for ephemeral VolumeSnapshot in namespace %q after %s: %w", namespace, after.Format(time.RFC3339), err)
		case <-time.After(ephemeralSnapshotPollInterval):
		}
	}
}

// listEphemeralVolumeSnapshot does a single list of VolumeSnapshots and returns the name
// of the most recent one that was pre-provisioned (not PVC-sourced) and created after `after`.
// The CSI driver creates pre-provisioned snapshots (spec.source.volumeSnapshotContentName set)
// and encodes the pod name in the snapshot name as "snapshot-{podName}-{timestamp}".
func listEphemeralVolumeSnapshot(ctx context.Context, dynCli dynamic.Interface, namespace string, after time.Time, podName string) (string, error) {
	snapList, err := dynCli.Resource(snapshot.VolSnapGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list VolumeSnapshots in namespace %q: %w", namespace, err)
	}

	var bestName string
	var bestTime time.Time

	for _, item := range snapList.Items {
		createdAt := item.GetCreationTimestamp().Time
		if !createdAt.After(after) {
			continue
		}

		// Ephemeral snapshots have spec.source.volumeSnapshotContentName set (pre-provisioned),
		// not spec.source.persistentVolumeClaimName (PVC-sourced).
		spec, ok := item.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		source, ok := spec["source"].(map[string]interface{})
		if !ok {
			continue
		}
		contentName, _ := source["volumeSnapshotContentName"].(string)
		if contentName == "" {
			continue
		}

		// The CSI driver encodes the pod name in the snapshot name as "snapshot-{podName}-{ts}".
		if podName != "" && !strings.HasPrefix(item.GetName(), "snapshot-"+podName+"-") {
			continue
		}

		if bestName == "" || createdAt.After(bestTime) {
			bestName = item.GetName()
			bestTime = createdAt
		}
	}

	if bestName == "" {
		return "", fmt.Errorf("no ephemeral VolumeSnapshot found in namespace %q created after %s", namespace, after.Format(time.RFC3339))
	}
	return bestName, nil
}
