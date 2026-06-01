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
	"testing"

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/param"
)

// Hook into the existing pkg/function test entrypoint by registering this
// suite via check.Suite; check.TestingT is already wired up in function_test.go.
var _ = check.Suite(&KubeTaskWithBackupPVCSuite{})

type KubeTaskWithBackupPVCSuite struct{}

// Ensure we hook into go test correctly even if this file is built standalone.
func TestKubeTaskWithBackupPVCSuiteEntry(t *testing.T) { check.TestingT(t) }

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsDefaults(c *check.C) {
	ctx := field.Context(context.Background(), consts.ActionsetUIDKey, "uid-abc-123")
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"},
	}
	// snapshotClass is required because takeSnapshot defaults to true.
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:         "bitnami/postgresql:latest",
		KubeTaskWithBackupPVCCommandArg:       []string{"sh", "-c", "echo hi > /backup/x"},
		KubeTaskWithBackupPVCSnapshotClassArg: "kopia-snapshot-class",
	}
	f := &kubeTaskWithBackupPVCFunc{}
	parsed, err := f.parseArgs(ctx, tp, args)
	c.Assert(err, check.IsNil)
	c.Check(parsed.image, check.Equals, "bitnami/postgresql:latest")
	c.Check(parsed.mountPath, check.Equals, defaultBackupPVCMountPath)
	c.Check(parsed.storageClass, check.Equals, defaultBackupPVCStorageClass)
	c.Check(parsed.size.String(), check.Equals, defaultBackupPVCSize)
	c.Check(parsed.timeout, check.Equals, defaultBackupPVCTimeout)
	c.Check(parsed.namespace, check.Equals, "demo")
	c.Check(parsed.workloadName, check.Equals, "pg")
	c.Check(parsed.workloadNamespace, check.Equals, "demo")
	c.Check(parsed.actionSetUID, check.Equals, "uid-abc-123")
	c.Check(parsed.pvcName, check.HasLen, len("pg-backup-")+6)
	c.Check(parsed.keepPVCOnFailure, check.Equals, false)
	c.Check(parsed.takeSnapshot, check.Equals, true)
	c.Check(parsed.snapshotClass, check.Equals, "kopia-snapshot-class")
	c.Check(parsed.cleanup, check.Equals, true)
}

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsHookModeDefaults(c *check.C) {
	// In hook mode (takeSnapshot=false), cleanup must be explicitly false
	// since the posthook owns staging-PVC cleanup. snapshotClass is irrelevant.
	ctx := field.Context(context.Background(), consts.ActionsetUIDKey, "u")
	tp := param.TemplateParams{StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"}}
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:                   "img",
		KubeTaskWithBackupPVCCommandArg:                 []string{"true"},
		KubeTaskWithBackupPVCTakeSnapshotArg:            false,
		KubeTaskWithBackupPVCCleanupArg:                 false,
		KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg: 300,
	}
	parsed, err := (&kubeTaskWithBackupPVCFunc{}).parseArgs(ctx, tp, args)
	c.Assert(err, check.IsNil)
	c.Check(parsed.takeSnapshot, check.Equals, false)
	c.Check(parsed.cleanup, check.Equals, false)
	c.Check(parsed.keepPodAliveSeconds, check.Equals, 300)
}

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsTakeSnapshotWithExplicitKeepAlive(c *check.C) {
	// takeSnapshot=true cooperates with keepPodAliveForSnapshot — they are
	// NOT mutually exclusive. The CSI driver needs a live mount during
	// CreateSnapshot, so keep-alive is the mechanism the function uses
	// to hold the mount; the user may override the duration.
	ctx := field.Context(context.Background(), consts.ActionsetUIDKey, "u")
	tp := param.TemplateParams{StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"}}
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:                   "img",
		KubeTaskWithBackupPVCCommandArg:                 []string{"true"},
		KubeTaskWithBackupPVCSnapshotClassArg:           "snap",
		KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg: 600,
	}
	parsed, err := (&kubeTaskWithBackupPVCFunc{}).parseArgs(ctx, tp, args)
	c.Assert(err, check.IsNil)
	c.Check(parsed.takeSnapshot, check.Equals, true)
	c.Check(parsed.keepPodAliveSeconds, check.Equals, 600)
}

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsTakeSnapshotDefaultsKeepAliveToTimeout(c *check.C) {
	// When takeSnapshot=true and keepPodAliveForSnapshot is unset, parseArgs
	// auto-fills the keep-alive duration to the function timeout.
	ctx := field.Context(context.Background(), consts.ActionsetUIDKey, "u")
	tp := param.TemplateParams{StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"}}
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:         "img",
		KubeTaskWithBackupPVCCommandArg:       []string{"true"},
		KubeTaskWithBackupPVCSnapshotClassArg: "snap",
		KubeTaskWithBackupPVCTimeoutArg:       "10m",
	}
	parsed, err := (&kubeTaskWithBackupPVCFunc{}).parseArgs(ctx, tp, args)
	c.Assert(err, check.IsNil)
	c.Check(parsed.takeSnapshot, check.Equals, true)
	c.Check(parsed.keepPodAliveSeconds, check.Equals, 600)
}

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsRejectsTakeSnapshotWithoutClass(c *check.C) {
	ctx := field.Context(context.Background(), consts.ActionsetUIDKey, "u")
	tp := param.TemplateParams{StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"}}
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:   "img",
		KubeTaskWithBackupPVCCommandArg: []string{"true"},
		// takeSnapshot defaults to true; snapshotClass missing → reject
	}
	_, err := (&kubeTaskWithBackupPVCFunc{}).parseArgs(ctx, tp, args)
	c.Assert(err, check.NotNil)
	c.Check(err.Error(), check.Matches, ".*snapshotClass.*")
}

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsRejectsHookModeWithCleanup(c *check.C) {
	ctx := field.Context(context.Background(), consts.ActionsetUIDKey, "u")
	tp := param.TemplateParams{StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"}}
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:        "img",
		KubeTaskWithBackupPVCCommandArg:      []string{"true"},
		KubeTaskWithBackupPVCTakeSnapshotArg: false,
		KubeTaskWithBackupPVCCleanupArg:      true,
	}
	_, err := (&kubeTaskWithBackupPVCFunc{}).parseArgs(ctx, tp, args)
	c.Assert(err, check.NotNil)
	c.Check(err.Error(), check.Matches, ".*cleanup=true requires takeSnapshot=true.*")
}

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsOverrides(c *check.C) {
	ctx := field.Context(context.Background(), consts.ActionsetUIDKey, "u")
	tp := param.TemplateParams{
		Deployment: &param.DeploymentParams{Name: "app", Namespace: "ns1"},
	}
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:            "img",
		KubeTaskWithBackupPVCCommandArg:          []string{"true"},
		KubeTaskWithBackupPVCNamespaceArg:        "alt-ns",
		KubeTaskWithBackupPVCPVCNameArg:          "explicit-name",
		KubeTaskWithBackupPVCStorageClassArg:     "my-sc",
		KubeTaskWithBackupPVCSizeArg:             "5Gi",
		KubeTaskWithBackupPVCPathArg:             "/data",
		KubeTaskWithBackupPVCTimeoutArg:          "5m",
		KubeTaskWithBackupPVCKeepPVCOnFailureArg: true,
		KubeTaskWithBackupPVCEnvFromSecretArg:    "pgcreds",
		// takeSnapshot defaults to true → snapshotClass required
		KubeTaskWithBackupPVCSnapshotClassArg: "snapshotclassname",
	}
	parsed, err := (&kubeTaskWithBackupPVCFunc{}).parseArgs(ctx, tp, args)
	c.Assert(err, check.IsNil)
	c.Check(parsed.namespace, check.Equals, "alt-ns")
	c.Check(parsed.pvcName, check.Equals, "explicit-name")
	c.Check(parsed.storageClass, check.Equals, "my-sc")
	c.Check(parsed.size.String(), check.Equals, "5Gi")
	c.Check(parsed.mountPath, check.Equals, "/data")
	c.Check(parsed.timeout.String(), check.Equals, "5m0s")
	c.Check(parsed.keepPVCOnFailure, check.Equals, true)
	c.Check(parsed.envFromSecret, check.Equals, "pgcreds")
}

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsMissingActionSetUID(c *check.C) {
	tp := param.TemplateParams{StatefulSet: &param.StatefulSetParams{Name: "pg", Namespace: "demo"}}
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:         "img",
		KubeTaskWithBackupPVCCommandArg:       []string{"true"},
		KubeTaskWithBackupPVCSnapshotClassArg: "snap", // required for takeSnapshot default=true
	}
	_, err := (&kubeTaskWithBackupPVCFunc{}).parseArgs(context.Background(), tp, args)
	c.Assert(err, check.NotNil)
}

func (s *KubeTaskWithBackupPVCSuite) TestParseArgsMissingNamespace(c *check.C) {
	ctx := field.Context(context.Background(), consts.ActionsetUIDKey, "u")
	args := map[string]interface{}{
		KubeTaskWithBackupPVCImageArg:         "img",
		KubeTaskWithBackupPVCCommandArg:       []string{"true"},
		KubeTaskWithBackupPVCSnapshotClassArg: "snap",
	}
	_, err := (&kubeTaskWithBackupPVCFunc{}).parseArgs(ctx, param.TemplateParams{}, args)
	c.Assert(err, check.NotNil)
}

func (s *KubeTaskWithBackupPVCSuite) TestStagingPVCLabels(c *check.C) {
	a := &kubeTaskWithBackupPVCArgs{
		workloadName:      "pg",
		workloadNamespace: "demo",
		actionSetUID:      "uid-1",
		bpLabels:          map[string]string{"team": "platform"},
	}
	labels := stagingPVCLabels(a)
	c.Check(labels[LabelKeyIncludeInBackup], check.Equals, "true")
	c.Check(labels[LabelKeyStagingPVC], check.Equals, "true")
	c.Check(labels[LabelKeyOwnerAction], check.Equals, "uid-1")
	c.Check(labels[LabelKeyWorkloadName], check.Equals, "pg")
	c.Check(labels[LabelKeyWorkloadNamespace], check.Equals, "demo")
	c.Check(labels["team"], check.Equals, "platform")
}

func (s *KubeTaskWithBackupPVCSuite) TestStagingPVCLabelsNoWorkloadNamespace(c *check.C) {
	a := &kubeTaskWithBackupPVCArgs{
		workloadName: "pg",
		actionSetUID: "uid-1",
	}
	labels := stagingPVCLabels(a)
	_, hasNs := labels[LabelKeyWorkloadNamespace]
	c.Check(hasNs, check.Equals, false)
}

func (s *KubeTaskWithBackupPVCSuite) TestCreateStagingPVCShape(c *check.C) {
	cli := fake.NewSimpleClientset()
	a := &kubeTaskWithBackupPVCArgs{
		image:             "img",
		command:           []string{"true"},
		mountPath:         "/backup",
		storageClass:      "kopia-backup",
		size:              resource.MustParse("1Gi"),
		pvcName:           "pg-backup-x",
		namespace:         "demo",
		workloadName:      "pg",
		workloadNamespace: "demo",
		actionSetUID:      "uid-1",
	}
	pvc, err := (&kubeTaskWithBackupPVCFunc{}).createStagingPVC(context.Background(), cli, a)
	c.Assert(err, check.IsNil)
	c.Assert(pvc, check.NotNil)
	c.Check(pvc.Name, check.Equals, "pg-backup-x")
	c.Check(pvc.Namespace, check.Equals, "demo")
	c.Check(*pvc.Spec.StorageClassName, check.Equals, "kopia-backup")
	c.Check(pvc.Spec.AccessModes, check.DeepEquals, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce})
	storage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	c.Check(storage.String(), check.Equals, "1Gi")
	c.Check(pvc.Labels[LabelKeyIncludeInBackup], check.Equals, "true")
	c.Check(pvc.Labels[LabelKeyOwnerAction], check.Equals, "uid-1")
	c.Check(pvc.Labels[LabelKeyStagingPVC], check.Equals, "true")
}

func (s *KubeTaskWithBackupPVCSuite) TestBuildPodOptionsEnvFromSecret(c *check.C) {
	a := &kubeTaskWithBackupPVCArgs{
		image:         "img",
		command:       []string{"echo", "hi"},
		envFromSecret: "pgcreds",
		mountPath:     "/backup",
		namespace:     "demo",
	}
	opts, err := (&kubeTaskWithBackupPVCFunc{}).buildPodOptions(a, "the-pvc")
	c.Assert(err, check.IsNil)
	c.Assert(opts.EnvFromSources, check.HasLen, 1)
	c.Assert(opts.EnvFromSources[0].SecretRef, check.NotNil)
	c.Check(opts.EnvFromSources[0].SecretRef.Name, check.Equals, "pgcreds")
	mount, ok := opts.Volumes["the-pvc"]
	c.Assert(ok, check.Equals, true)
	c.Check(mount.MountPath, check.Equals, "/backup")
	c.Check(mount.ReadOnly, check.Equals, false)
}

func (s *KubeTaskWithBackupPVCSuite) TestBuildPodOptionsNoSecret(c *check.C) {
	a := &kubeTaskWithBackupPVCArgs{
		image:     "img",
		command:   []string{"true"},
		mountPath: "/backup",
		namespace: "demo",
	}
	opts, err := (&kubeTaskWithBackupPVCFunc{}).buildPodOptions(a, "pvc")
	c.Assert(err, check.IsNil)
	c.Check(opts.EnvFromSources, check.IsNil)
}

func (s *KubeTaskWithBackupPVCSuite) TestRequiredAndArguments(c *check.C) {
	f := &kubeTaskWithBackupPVCFunc{}
	c.Check(f.Name(), check.Equals, KubeTaskWithBackupPVCFuncName)
	required := f.RequiredArgs()
	c.Check(required, check.DeepEquals, []string{KubeTaskWithBackupPVCImageArg, KubeTaskWithBackupPVCCommandArg})

	all := f.Arguments()
	// Required args must appear in Arguments() too.
	for _, r := range required {
		found := false
		for _, a := range all {
			if a == r {
				found = true
				break
			}
		}
		c.Check(found, check.Equals, true, check.Commentf("required arg %q must appear in Arguments()", r))
	}
}

func (s *KubeTaskWithBackupPVCSuite) TestValidateRejectsUnknownArg(c *check.C) {
	f := &kubeTaskWithBackupPVCFunc{}
	err := f.Validate(map[string]any{
		KubeTaskWithBackupPVCImageArg:   "img",
		KubeTaskWithBackupPVCCommandArg: []string{"true"},
		"totallyUnknownArg":             true,
	})
	c.Assert(err, check.NotNil)
}

func (s *KubeTaskWithBackupPVCSuite) TestValidateRejectsMissingRequired(c *check.C) {
	f := &kubeTaskWithBackupPVCFunc{}
	err := f.Validate(map[string]any{
		KubeTaskWithBackupPVCImageArg: "img",
	})
	c.Assert(err, check.NotNil)
}

// Sanity check: PodOptions stays consistent on metadata
var _ = metav1.ObjectMeta{}
