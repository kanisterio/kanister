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
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// KubeTaskWithRestorePVCFuncName gives the name of the function
	KubeTaskWithRestorePVCFuncName = "KubeTaskWithRestorePVC"
	// KubeTaskWithRestorePVCImageArg provides the container image used for the restore worker pod
	KubeTaskWithRestorePVCImageArg = "image"
	// KubeTaskWithRestorePVCCommandArg provides the shell-form command that reads from the staging PVC
	KubeTaskWithRestorePVCCommandArg = "command"
	// KubeTaskWithRestorePVCEnvArg provides the list of environment variables injected into the worker pod
	KubeTaskWithRestorePVCEnvArg = "env"
	// KubeTaskWithRestorePVCPathArg provides the read-only mount path for the staging PVC inside the worker pod
	KubeTaskWithRestorePVCPathArg = "path"
	// KubeTaskWithRestorePVCStorageClassArg provides the StorageClass used when provisioning a fresh staging PVC from a snapshot
	KubeTaskWithRestorePVCStorageClassArg = "storageClassName"
	// KubeTaskWithRestorePVCPVCSelectorArg provides a LabelSelector to discover an existing staging PVC to restore into
	KubeTaskWithRestorePVCPVCSelectorArg = "pvcSelector"
	// KubeTaskWithRestorePVCNamespaceArg provides the namespace to create the staging PVC and worker pod in
	KubeTaskWithRestorePVCNamespaceArg = "namespace"
	// KubeTaskWithRestorePVCServiceAccountArg provides the ServiceAccount for the worker pod
	KubeTaskWithRestorePVCServiceAccountArg = "serviceAccountName"
	// KubeTaskWithRestorePVCTimeoutArg provides the overall timeout for the phase
	KubeTaskWithRestorePVCTimeoutArg = "timeout"
	// KubeTaskWithRestorePVCCleanupPVCArg controls whether the staging PVC is deleted at phase exit
	KubeTaskWithRestorePVCCleanupPVCArg = "cleanupPVC"
	// KubeTaskWithRestorePVCSourcePVCNameArg names an existing PVC or the original backup PVC used for snapshot lookup
	KubeTaskWithRestorePVCSourcePVCNameArg = "sourcePVCName"
	// KubeTaskWithRestorePVCSizeArg overrides the requested size of the staging PVC when provisioning from a snapshot
	KubeTaskWithRestorePVCSizeArg = "size"
	// KubeTaskWithRestorePVCVolumeSnapshotNameArg names the VolumeSnapshot the fresh staging PVC is provisioned from
	KubeTaskWithRestorePVCVolumeSnapshotNameArg = "volumeSnapshotName"
	// KubeTaskWithRestorePVCVolumeSnapshotNamespaceArg names the namespace of the VolumeSnapshot referenced by volumeSnapshotName
	KubeTaskWithRestorePVCVolumeSnapshotNamespaceArg = "volumeSnapshotNamespace"
	// KubeTaskWithRestorePVCRestoreSizeArg provides the fallback PVC size used when the VolumeSnapshot status carries none
	KubeTaskWithRestorePVCRestoreSizeArg = "restoreSize"

	defaultRestorePVCMountPath = "/restore"
	defaultRestorePVCTimeout   = 30 * time.Minute

	restorePVCJobPrefix = "kanister-restore-pvc-"
)

func init() {
	_ = kanister.Register(&kubeTaskWithRestorePVCFunc{})
}

// NewKubeTaskWithRestorePVCFunc returns a new instance of the generic
// KubeTaskWithRestorePVC function. Versioned overrides (e.g. a downstream
// v1.0.0-alpha registration) embed the returned value to reuse the generic
// restore orchestration while adding their own pre/post behaviour such as
// cross-cluster snapshot bridging.
func NewKubeTaskWithRestorePVCFunc() kanister.Func {
	return &kubeTaskWithRestorePVCFunc{}
}

var _ kanister.Func = (*kubeTaskWithRestorePVCFunc)(nil)

// kubeTaskWithRestorePVCFunc provisions a staging PVC from a backed-up source
// and runs a worker pod against it to restore data. It is CSI-driver agnostic:
// the source can be a named VolumeSnapshot (volumeSnapshotName), an existing
// PVC (sourcePVCName), or a label-selected staging PVC (pvcSelector), and the
// staging StorageClass is supplied as an arg, so it works with any CSI driver
// that supports provisioning a PVC from a VolumeSnapshot dataSource.
//
// This generic implementation operates within a single cluster: the referenced
// VolumeSnapshot must already exist on the cluster. (Cross-cluster restore from
// a foreign snapshot handle is an opt-in capability layered on by a versioned
// override, not part of the generic function.)
type kubeTaskWithRestorePVCFunc struct {
	progressPercent string
}

func (*kubeTaskWithRestorePVCFunc) Name() string {
	return KubeTaskWithRestorePVCFuncName
}

type kubeTaskWithRestorePVCArgs struct {
	image                   string
	command                 []string
	env                     []corev1.EnvVar
	mountPath               string
	storageClass            string
	pvcSelector             metav1.LabelSelector
	namespace               string
	podOverride             crv1alpha1.JSONMap
	serviceAccount          string
	timeout                 time.Duration
	cleanupPVC              bool
	bpAnnotations           map[string]string
	bpLabels                map[string]string
	sourcePVCName           string
	size                    resource.Quantity
	volumeSnapshotName      string
	volumeSnapshotNamespace string
	restoreSize             resource.Quantity

	workloadName      string
	workloadNamespace string
}

func (*kubeTaskWithRestorePVCFunc) RequiredArgs() []string {
	return []string{
		KubeTaskWithRestorePVCImageArg,
		KubeTaskWithRestorePVCCommandArg,
	}
}

func (*kubeTaskWithRestorePVCFunc) Arguments() []string {
	return []string{
		KubeTaskWithRestorePVCImageArg,
		KubeTaskWithRestorePVCCommandArg,
		KubeTaskWithRestorePVCEnvArg,
		KubeTaskWithRestorePVCPathArg,
		KubeTaskWithRestorePVCStorageClassArg,
		KubeTaskWithRestorePVCPVCSelectorArg,
		KubeTaskWithRestorePVCNamespaceArg,
		KubeTaskWithRestorePVCServiceAccountArg,
		KubeTaskWithRestorePVCTimeoutArg,
		KubeTaskWithRestorePVCCleanupPVCArg,
		KubeTaskWithRestorePVCSourcePVCNameArg,
		KubeTaskWithRestorePVCSizeArg,
		KubeTaskWithRestorePVCVolumeSnapshotNameArg,
		KubeTaskWithRestorePVCVolumeSnapshotNamespaceArg,
		KubeTaskWithRestorePVCRestoreSizeArg,
		PodOverrideArg,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (f *kubeTaskWithRestorePVCFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(f.Name(), args); err != nil {
		return err
	}
	if err := utils.CheckSupportedArgs(f.Arguments(), args); err != nil {
		return err
	}
	return utils.CheckRequiredArgs(f.RequiredArgs(), args)
}

func (f *kubeTaskWithRestorePVCFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    f.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func (f *kubeTaskWithRestorePVCFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	f.progressPercent = progress.StartedPercent
	defer func() { f.progressPercent = progress.CompletedPercent }()

	parsed, err := f.parseArgs(tp, args)
	if err != nil {
		return nil, err
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}

	return f.run(ctx, cli, parsed)
}

func (f *kubeTaskWithRestorePVCFunc) parseArgs(tp param.TemplateParams, args map[string]interface{}) (*kubeTaskWithRestorePVCArgs, error) {
	parsed := &kubeTaskWithRestorePVCArgs{}
	if err := f.parseRestoreCoreArgs(tp, args, parsed); err != nil {
		return nil, err
	}
	if err := f.parseRestoreSnapshotArgs(args, parsed); err != nil {
		return nil, err
	}
	if err := f.parseRestorePodArgs(tp, args, parsed); err != nil {
		return nil, err
	}
	return parsed, f.resolveRestoreContext(tp, parsed)
}

func (f *kubeTaskWithRestorePVCFunc) parseRestoreCoreArgs(tp param.TemplateParams, args map[string]interface{}, parsed *kubeTaskWithRestorePVCArgs) error {
	var err error
	if err = Arg(args, KubeTaskWithRestorePVCImageArg, &parsed.image); err != nil {
		return err
	}
	if err = Arg(args, KubeTaskWithRestorePVCCommandArg, &parsed.command); err != nil {
		return err
	}
	if parsed.env, err = ParseEnvVars(args, KubeTaskWithRestorePVCEnvArg); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCPathArg, &parsed.mountPath, defaultRestorePVCMountPath); err != nil {
		return err
	}
	// Empty storageClass => use the cluster's default StorageClass (nil pointer
	// at PVC-creation time). The blueprint passes storageClassName to override.
	if err = OptArg(args, KubeTaskWithRestorePVCStorageClassArg, &parsed.storageClass, ""); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCPVCSelectorArg, &parsed.pvcSelector, metav1.LabelSelector{}); err != nil {
		return errkit.Wrap(err, "Failed to parse pvcSelector")
	}
	if err = OptArg(args, KubeTaskWithRestorePVCNamespaceArg, &parsed.namespace, ""); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCServiceAccountArg, &parsed.serviceAccount, ""); err != nil {
		return err
	}
	var timeoutStr string
	if err = OptArg(args, KubeTaskWithRestorePVCTimeoutArg, &timeoutStr, defaultRestorePVCTimeout.String()); err != nil {
		return err
	}
	if parsed.timeout, err = time.ParseDuration(timeoutStr); err != nil {
		return errkit.Wrap(err, "Failed to parse timeout", "timeout", timeoutStr)
	}
	return OptArg(args, KubeTaskWithRestorePVCCleanupPVCArg, &parsed.cleanupPVC, true)
}

func (f *kubeTaskWithRestorePVCFunc) parseRestoreSnapshotArgs(args map[string]interface{}, parsed *kubeTaskWithRestorePVCArgs) error {
	if err := OptArg(args, KubeTaskWithRestorePVCSourcePVCNameArg, &parsed.sourcePVCName, ""); err != nil {
		return err
	}
	if err := parseOptionalQuantity(args, KubeTaskWithRestorePVCSizeArg, "size", &parsed.size); err != nil {
		return err
	}
	if err := OptArg(args, KubeTaskWithRestorePVCVolumeSnapshotNameArg, &parsed.volumeSnapshotName, ""); err != nil {
		return err
	}
	if err := OptArg(args, KubeTaskWithRestorePVCVolumeSnapshotNamespaceArg, &parsed.volumeSnapshotNamespace, ""); err != nil {
		return err
	}
	return parseOptionalQuantity(args, KubeTaskWithRestorePVCRestoreSizeArg, "restoreSize", &parsed.restoreSize)
}

func parseOptionalQuantity(args map[string]interface{}, argName, errLabel string, out *resource.Quantity) error {
	var s string
	if err := OptArg(args, argName, &s, ""); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return errkit.Wrap(err, "Failed to parse "+errLabel, errLabel, s)
	}
	*out = q
	return nil
}

func (f *kubeTaskWithRestorePVCFunc) parseRestorePodArgs(tp param.TemplateParams, args map[string]interface{}, parsed *kubeTaskWithRestorePVCArgs) error {
	if err := OptArg(args, PodAnnotationsArg, &parsed.bpAnnotations, nil); err != nil {
		return err
	}
	if err := OptArg(args, PodLabelsArg, &parsed.bpLabels, nil); err != nil {
		return err
	}
	var err error
	parsed.podOverride, err = GetPodSpecOverride(tp, args, PodOverrideArg)
	return err
}

func (f *kubeTaskWithRestorePVCFunc) resolveRestoreContext(tp param.TemplateParams, parsed *kubeTaskWithRestorePVCArgs) error {
	parsed.workloadName, parsed.workloadNamespace = WorkloadFromTemplateParams(tp)
	if parsed.namespace == "" {
		parsed.namespace = parsed.workloadNamespace
	}
	if parsed.namespace == "" {
		return errkit.New("Unable to resolve namespace; pass the namespace arg or run the function against a workload action context")
	}
	if len(parsed.pvcSelector.MatchLabels) > 0 || len(parsed.pvcSelector.MatchExpressions) > 0 {
		return nil
	}
	matchLabels := map[string]string{LabelKeyStagingPVC: "true"}
	if parsed.workloadName != "" {
		matchLabels[LabelKeyWorkloadName] = parsed.workloadName
	}
	if parsed.workloadNamespace != "" {
		matchLabels[LabelKeyWorkloadNamespace] = parsed.workloadNamespace
	}
	// With no workload context the only label is staging-pvc=true, which would
	// match every staging PVC in the namespace. Require an explicit
	// sourcePVCName or volumeSnapshotName to disambiguate in that case.
	if len(matchLabels) == 1 && parsed.sourcePVCName == "" && parsed.volumeSnapshotName == "" {
		return errkit.New("Unable to derive default pvcSelector: no workload context. Pass an explicit pvcSelector, sourcePVCName, or volumeSnapshotName arg.")
	}
	parsed.pvcSelector = metav1.LabelSelector{MatchLabels: matchLabels}
	return nil
}

func (f *kubeTaskWithRestorePVCFunc) run(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithRestorePVCArgs) (out map[string]interface{}, retErr error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// PVC resolution: try in order until one succeeds.
	//   1. volumeSnapshotName  → materialize fresh PVC from the named VS
	//   2. sourcePVCName       → use existing PVC by exact name
	//   3. pvcSelector         → discover an existing staging PVC
	//   4. sourcePVCName fallback → find VS whose source matched, then restore
	var (
		pvc *corev1.PersistentVolumeClaim
		err error
	)
	if a.volumeSnapshotName != "" {
		log.WithContext(ctx).Print("Restoring fresh PVC from named VolumeSnapshot (function-owned)",
			field.M{"volumeSnapshotName": a.volumeSnapshotName,
				"volumeSnapshotNamespace": a.volumeSnapshotNamespace,
				"namespace":               a.namespace})

		pvc, err = f.restorePVCFromNamedSnapshot(ctx, cli, a)
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to restore PVC from named VolumeSnapshot",
				"volumeSnapshotName", a.volumeSnapshotName, "namespace", a.namespace)
		}
	}
	if pvc == nil && a.sourcePVCName != "" {
		existing, getErr := cli.CoreV1().PersistentVolumeClaims(a.namespace).Get(ctx, a.sourcePVCName, metav1.GetOptions{})
		switch {
		case getErr == nil:
			log.WithContext(ctx).Print("Using pre-existing staging PVC named by sourcePVCName",
				field.M{"namespace": a.namespace, "pvcName": a.sourcePVCName})
			pvc = existing
		case !apierrors.IsNotFound(getErr):
			return nil, errkit.Wrap(getErr, "Failed to look up PVC referenced by sourcePVCName",
				"namespace", a.namespace, "pvcName", a.sourcePVCName)
		}
	}
	if pvc == nil {
		pvc, err = f.findStagingPVC(ctx, cli, a)
		if err != nil && a.sourcePVCName != "" {
			log.WithContext(ctx).Print("No live staging PVC; restoring fresh PVC from VolumeSnapshot",
				field.M{"sourcePVCName": a.sourcePVCName, "namespace": a.namespace})
			pvc, err = f.restorePVCFromSnapshot(ctx, cli, a)
		}
		if err != nil {
			return nil, err
		}
	}

	// Delete staging PVC on exit (when cleanupPVC=true, the default). The
	// VolumeSnapshot CR survives independently.
	defer func() {
		if !a.cleanupPVC {
			return
		}
		if delErr := PVCGracefulDelete(ctx, cli, a.namespace, pvc.Name); delErr != nil {
			log.WithError(delErr).WithContext(ctx).Print("Failed to delete restored staging PVC",
				field.M{"namespace": a.namespace, "pvcName": pvc.Name})
		}
	}()

	podOpts := f.buildPodOptions(a, pvc.Name)
	if err := ephemeral.PodOptions.Apply(podOpts); err != nil {
		return nil, errkit.Wrap(err, "Failed to apply ephemeral pod options")
	}
	kube.AddLabelsToPodOptionsFromContext(ctx, podOpts, path.Join(consts.LabelPrefix, consts.LabelSuffixJobID))

	pr := kube.NewPodRunner(cli, podOpts)
	podOut, err := pr.Run(ctx, StagingPodRunner("Restore pod did not complete successfully"))
	if err != nil {
		return nil, errkit.Wrap(err, "Restore command failed",
			"namespace", a.namespace, "pvcName", pvc.Name)
	}
	return podOut, nil
}

// restorePVCFromSnapshot locates the VolumeSnapshot whose source matches the
// backup-side staging PVC name and provisions a fresh PVC from it. The VS
// survives for the RestorePoint's lifetime, so this lookup is reliable as long
// as the RestorePoint still exists.
func (f *kubeTaskWithRestorePVCFunc) restorePVCFromSnapshot(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithRestorePVCArgs) (*corev1.PersistentVolumeClaim, error) {
	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create dynamic client")
	}
	snapshotter := snapshot.NewSnapshotter(cli, dynCli)
	vsList, err := snapshotter.List(ctx, a.namespace, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to list VolumeSnapshots", "namespace", a.namespace)
	}
	var matches []string
	var match *stagingSnapshotRef
	for i := range vsList.Items {
		vs := &vsList.Items[i]
		if vs.Spec.Source.PersistentVolumeClaimName == nil {
			continue
		}
		if *vs.Spec.Source.PersistentVolumeClaimName != a.sourcePVCName {
			continue
		}
		if vs.Status == nil || vs.Status.ReadyToUse == nil || !*vs.Status.ReadyToUse {
			matches = append(matches, vs.Name+"(notReady)")
			continue
		}
		matches = append(matches, vs.Name)
		if match == nil {
			match = &stagingSnapshotRef{name: vs.Name, restoreSize: vs.Status.RestoreSize}
		}
	}
	switch {
	case match == nil && len(matches) == 0:
		return nil, errkit.New("No VolumeSnapshot found for source staging PVC; restore point may be retired",
			"namespace", a.namespace, "sourcePVCName", a.sourcePVCName)
	case match == nil:
		return nil, errkit.New("Found VolumeSnapshot(s) matching source staging PVC but none are ready",
			"namespace", a.namespace, "sourcePVCName", a.sourcePVCName, "matches", strings.Join(matches, ","))
	}
	if len(matches) > 1 {
		log.WithContext(ctx).Print("Multiple VolumeSnapshots matched source staging PVC; using the first ready one",
			field.M{"namespace": a.namespace, "sourcePVCName": a.sourcePVCName, "matches": strings.Join(matches, ","), "chosen": match.name})
	}

	// Size: explicit `size` arg, else snapshot's RestoreSize, else default.
	size := a.size
	if size.IsZero() && match.restoreSize != nil {
		size = *match.restoreSize
	}
	if size.IsZero() {
		size = resource.MustParse(defaultBackupPVCSize)
	}

	pvcName := f.deriveRestoredPVCName(a)
	// nil StorageClassName => cluster default StorageClass. A pointer to "" would
	// instead disable dynamic provisioning, so only set it when resolved.
	var scPtr *string
	if a.storageClass != "" {
		sc := a.storageClass
		scPtr = &sc
	}
	apiGroup := "snapshot.storage.k8s.io"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: a.namespace,
			Labels:    restoredPVCLabels(a),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: scPtr,
			DataSource: &corev1.TypedLocalObjectReference{
				APIGroup: &apiGroup,
				Kind:     "VolumeSnapshot",
				Name:     match.name,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}
	created, err := cli.CoreV1().PersistentVolumeClaims(a.namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create restored staging PVC",
			"namespace", a.namespace, "pvcName", pvcName, "volumeSnapshot", match.name)
	}
	log.WithContext(ctx).Print("Created restored staging PVC from VolumeSnapshot",
		field.M{"namespace": a.namespace, "pvcName": created.Name, "volumeSnapshot": match.name, "storageClass": a.storageClass, "size": size.String()})

	if err := WaitForPVCBound(ctx, cli, a.namespace, created.Name); err != nil {
		return nil, errkit.Wrap(err, "Restored staging PVC did not become Bound",
			"namespace", a.namespace, "pvcName", created.Name, "volumeSnapshot", match.name)
	}
	return created, nil
}

// restorePVCFromNamedSnapshot materializes a fresh PVC from the named
// VolumeSnapshot via the package-private restoreCSISnapshot helper. PVC
// name is auto-generated (<workload>-restore-<random6>) to avoid retry
// collisions.
func (f *kubeTaskWithRestorePVCFunc) restorePVCFromNamedSnapshot(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithRestorePVCArgs) (*corev1.PersistentVolumeClaim, error) {
	// CSI dataSource constraint: snapshot and PVC must share a namespace.
	ns := a.volumeSnapshotNamespace
	if ns == "" {
		ns = a.namespace
	}

	size := f.resolveRestoreSize(ctx, cli, ns, a)
	pvcName := f.deriveRestoredPVCName(a)
	restoreArgs := restoreCSISnapshotArgs{
		Name:         a.volumeSnapshotName,
		PVC:          pvcName,
		Namespace:    ns,
		StorageClass: a.storageClass,
		RestoreSize:  &size,
		AccessModes:  []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		VolumeMode:   corev1.PersistentVolumeFilesystem,
		Labels:       restoredPVCLabels(a),
	}
	if _, err := restoreCSISnapshot(ctx, cli, restoreArgs); err != nil {
		return nil, errkit.Wrap(err, "Failed to create PVC from VolumeSnapshot",
			"volumeSnapshotName", a.volumeSnapshotName, "namespace", ns, "pvcName", pvcName)
	}
	log.WithContext(ctx).Print("Created restored staging PVC from named VolumeSnapshot",
		field.M{"namespace": ns, "pvcName": pvcName, "volumeSnapshotName": a.volumeSnapshotName,
			"storageClass": a.storageClass, "size": size.String()})

	if err := WaitForPVCBound(ctx, cli, ns, pvcName); err != nil {
		return nil, errkit.Wrap(err, "Restored staging PVC did not become Bound",
			"namespace", ns, "pvcName", pvcName, "volumeSnapshotName", a.volumeSnapshotName)
	}
	bound, err := cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to Get restored staging PVC after Bound",
			"namespace", ns, "pvcName", pvcName)
	}
	return bound, nil
}

// defaultRestoreSize is the last-resort PVC size when neither the snapshot
// status nor blueprint args carry a usable value.
const defaultRestoreSize = "5Gi"

// resolveRestoreSize picks the PVC size in this order:
//  1. VolumeSnapshot.status.restoreSize
//  2. a.restoreSize (blueprint arg; escape hatch for drivers that leave status empty)
//  3. a.size (generic override)
//  4. defaultRestoreSize
func (f *kubeTaskWithRestorePVCFunc) resolveRestoreSize(ctx context.Context, cli kubernetes.Interface, ns string, a *kubeTaskWithRestorePVCArgs) resource.Quantity {
	dynCli, err := kube.NewDynamicClient()
	if err == nil {
		snapshotter := snapshot.NewSnapshotter(cli, dynCli)
		vs, getErr := snapshotter.Get(ctx, a.volumeSnapshotName, ns)
		if getErr == nil && vs.Status != nil && vs.Status.RestoreSize != nil && !vs.Status.RestoreSize.IsZero() {
			log.WithContext(ctx).Print("Restore size resolved from VolumeSnapshot.status.restoreSize",
				field.M{"volumeSnapshotName": a.volumeSnapshotName, "namespace": ns, "size": vs.Status.RestoreSize.String()})
			return *vs.Status.RestoreSize
		}
	}
	if !a.restoreSize.IsZero() {
		log.WithContext(ctx).Print("Restore size taken from explicit restoreSize arg",
			field.M{"volumeSnapshotName": a.volumeSnapshotName, "namespace": ns, "size": a.restoreSize.String()})
		return a.restoreSize
	}
	if !a.size.IsZero() {
		log.WithContext(ctx).Print("Restore size taken from generic size arg",
			field.M{"volumeSnapshotName": a.volumeSnapshotName, "namespace": ns, "size": a.size.String()})
		return a.size
	}
	def := resource.MustParse(defaultRestoreSize)
	log.WithContext(ctx).Print("Restore size unavailable from snapshot status or args; using default",
		field.M{"volumeSnapshotName": a.volumeSnapshotName, "namespace": ns, "default": def.String()})
	return def
}

// stagingSnapshotRef carries the chosen VolumeSnapshot name and its restore
// size out of the matching loop.
type stagingSnapshotRef struct {
	name        string
	restoreSize *resource.Quantity
}

// deriveRestoredPVCName names the freshly restored PVC with a workload prefix.
func (f *kubeTaskWithRestorePVCFunc) deriveRestoredPVCName(a *kubeTaskWithRestorePVCArgs) string {
	base := a.workloadName
	if base == "" {
		base = "kanister"
	}
	return fmt.Sprintf("%s-restore-%s", base, rand.String(6))
}

// restoredPVCLabels stamps the staging-PVC labels for downstream discovery.
func restoredPVCLabels(a *kubeTaskWithRestorePVCArgs) map[string]string {
	out := map[string]string{
		LabelKeyStagingPVC: "true",
	}
	if a.workloadName != "" {
		out[LabelKeyWorkloadName] = a.workloadName
	}
	if a.workloadNamespace != "" {
		out[LabelKeyWorkloadNamespace] = a.workloadNamespace
	}
	for k, v := range a.bpLabels {
		out[k] = v
	}
	return out
}

// findStagingPVC asserts exactly one PVC matches the selector; zero or
// multiple matches return a diagnostic.
func (f *kubeTaskWithRestorePVCFunc) findStagingPVC(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithRestorePVCArgs) (*corev1.PersistentVolumeClaim, error) {
	sel, err := metav1.LabelSelectorAsSelector(&a.pvcSelector)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to build label selector")
	}
	if sel.Empty() {
		return nil, errkit.New("Refusing to list PVCs with an empty selector; would match every PVC in the namespace",
			"namespace", a.namespace)
	}
	list, err := cli.CoreV1().PersistentVolumeClaims(a.namespace).List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to list PVCs", "namespace", a.namespace, "selector", sel.String())
	}
	switch len(list.Items) {
	case 0:
		return nil, errkit.New("No staging PVC matched the selector; backup/restore mismatch likely (different blueprint or workload)",
			"namespace", a.namespace, "selector", sel.String())
	case 1:
		return &list.Items[0], nil
	default:
		names := make([]string, 0, len(list.Items))
		for i := range list.Items {
			names = append(names, list.Items[i].Name)
		}
		return nil, errkit.New("Multiple PVCs matched the selector; narrow the pvcSelector arg",
			"namespace", a.namespace, "selector", sel.String(), "matches", strings.Join(names, ","))
	}
}

func (f *kubeTaskWithRestorePVCFunc) buildPodOptions(a *kubeTaskWithRestorePVCArgs, pvcName string) *kube.PodOptions {
	return &kube.PodOptions{
		Namespace:          a.namespace,
		GenerateName:       restorePVCJobPrefix,
		Image:              a.image,
		Command:            a.command,
		ServiceAccountName: a.serviceAccount,
		Volumes: map[string]kube.VolumeMountOptions{
			pvcName: {
				MountPath: a.mountPath,
				ReadOnly:  true,
			},
		},
		EnvironmentVariables: a.env,
		PodOverride:          a.podOverride,
		Annotations:          a.bpAnnotations,
		Labels:               a.bpLabels,
	}
}
