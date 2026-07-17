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
	KubeTaskWithRestorePVCFuncName = "KubeTaskWithRestorePVC"

	KubeTaskWithRestorePVCImageArg                   = "image"
	KubeTaskWithRestorePVCCommandArg                 = "command"
	KubeTaskWithRestorePVCEnvArg                     = "env"
	KubeTaskWithRestorePVCPathArg                    = "path"
	KubeTaskWithRestorePVCStorageClassArg            = "storageClassName"
	KubeTaskWithRestorePVCPVCSelectorArg             = "pvcSelector"
	KubeTaskWithRestorePVCNamespaceArg               = "namespace"
	KubeTaskWithRestorePVCServiceAccountArg          = "serviceAccountName"
	KubeTaskWithRestorePVCTimeoutArg                 = "timeout"
	KubeTaskWithRestorePVCCleanupPVCArg              = "cleanupPVC"
	KubeTaskWithRestorePVCSourcePVCNameArg           = "sourcePVCName"
	KubeTaskWithRestorePVCSizeArg                    = "size"
	KubeTaskWithRestorePVCVolumeSnapshotNameArg      = "volumeSnapshotName"
	KubeTaskWithRestorePVCVolumeSnapshotNamespaceArg = "volumeSnapshotNamespace"
	KubeTaskWithRestorePVCRestoreSizeArg             = "restoreSize"

	defaultRestorePVCStorageClass = "kopia-restore"
	defaultRestorePVCMountPath    = "/restore"
	defaultRestorePVCTimeout      = 30 * time.Minute
	defaultRestoreSize            = "5Gi"

	restorePVCJobPrefix = "kanister-restore-pvc-"
)

func init() {
	_ = kanister.Register(&kubeTaskWithRestorePVCFunc{})
}

var _ kanister.Func = (*kubeTaskWithRestorePVCFunc)(nil)

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
	if err := f.parseRestoreCoreArgs(args, parsed); err != nil {
		return nil, err
	}
	if err := f.parseRestoreSnapshotArgs(args, parsed); err != nil {
		return nil, err
	}
	if err := f.parseRestorePodArgs(tp, args, parsed); err != nil {
		return nil, err
	}
	if err := f.resolveRestoreContext(tp, parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (f *kubeTaskWithRestorePVCFunc) parseRestoreCoreArgs(args map[string]interface{}, parsed *kubeTaskWithRestorePVCArgs) error {
	var err error
	if err = Arg(args, KubeTaskWithRestorePVCImageArg, &parsed.image); err != nil {
		return err
	}
	if err = Arg(args, KubeTaskWithRestorePVCCommandArg, &parsed.command); err != nil {
		return err
	}
	if parsed.env, err = parseEnvVars(args, KubeTaskWithRestorePVCEnvArg); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCPathArg, &parsed.mountPath, defaultRestorePVCMountPath); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCStorageClassArg, &parsed.storageClass, defaultRestorePVCStorageClass); err != nil {
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
	if err := parseOptionalQuantity(args, KubeTaskWithRestorePVCSizeArg, &parsed.size); err != nil {
		return err
	}
	if err := OptArg(args, KubeTaskWithRestorePVCVolumeSnapshotNameArg, &parsed.volumeSnapshotName, ""); err != nil {
		return err
	}
	if err := OptArg(args, KubeTaskWithRestorePVCVolumeSnapshotNamespaceArg, &parsed.volumeSnapshotNamespace, ""); err != nil {
		return err
	}
	return parseOptionalQuantity(args, KubeTaskWithRestorePVCRestoreSizeArg, &parsed.restoreSize)
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
	parsed.workloadName, parsed.workloadNamespace = workloadFromTemplateParams(tp)
	if parsed.namespace == "" {
		parsed.namespace = parsed.workloadNamespace
	}
	if parsed.namespace == "" {
		return errkit.New("Unable to resolve namespace; pass the namespace arg or run the function against a workload action context")
	}
	if len(parsed.pvcSelector.MatchLabels) == 0 && len(parsed.pvcSelector.MatchExpressions) == 0 {
		matchLabels := map[string]string{LabelKeyStagingPVC: "true"}
		if parsed.workloadName != "" {
			matchLabels[LabelKeyWorkloadName] = parsed.workloadName
		}
		if parsed.workloadNamespace != "" {
			matchLabels[LabelKeyWorkloadNamespace] = parsed.workloadNamespace
		}
		if len(matchLabels) == 1 && parsed.sourcePVCName == "" {
			return errkit.New("Unable to derive default pvcSelector: no workload context. Pass an explicit pvcSelector arg or sourcePVCName arg.")
		}
		parsed.pvcSelector = metav1.LabelSelector{MatchLabels: matchLabels}
	}
	return nil
}

func parseOptionalQuantity(args map[string]interface{}, key string, out *resource.Quantity) error {
	var s string
	if err := OptArg(args, key, &s, ""); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return errkit.Wrap(err, "Failed to parse quantity arg", "arg", key, "value", s)
	}
	*out = q
	return nil
}

func (f *kubeTaskWithRestorePVCFunc) run(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithRestorePVCArgs) (out map[string]interface{}, retErr error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	pvc, err := f.resolveStagingPVC(ctx, cli, a)
	if err != nil {
		return nil, err
	}

	defer func() {
		if !a.cleanupPVC {
			return
		}
		if delErr := pvcGracefulDelete(ctx, cli, a.namespace, pvc.Name); delErr != nil {
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
	podOut, err := pr.Run(ctx, stagingPodRunner("Restore pod did not complete successfully"))
	if err != nil {
		return nil, errkit.Wrap(err, "Restore command failed",
			"namespace", a.namespace, "pvcName", pvc.Name)
	}
	return podOut, nil
}

// resolveStagingPVC picks the staging PVC via (in order): named VolumeSnapshot,
// exact sourcePVCName, label selector, snapshot-search fallback by source name.
func (f *kubeTaskWithRestorePVCFunc) resolveStagingPVC(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithRestorePVCArgs) (*corev1.PersistentVolumeClaim, error) {
	if a.volumeSnapshotName != "" {
		log.WithContext(ctx).Print("Restoring fresh PVC from named VolumeSnapshot",
			field.M{"volumeSnapshotName": a.volumeSnapshotName, "namespace": a.namespace})
		pvc, err := f.restorePVCFromNamedSnapshot(ctx, cli, a)
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to restore PVC from named VolumeSnapshot",
				"volumeSnapshotName", a.volumeSnapshotName, "namespace", a.namespace)
		}
		return pvc, nil
	}
	if a.sourcePVCName != "" {
		existing, getErr := cli.CoreV1().PersistentVolumeClaims(a.namespace).Get(ctx, a.sourcePVCName, metav1.GetOptions{})
		switch {
		case getErr == nil:
			return existing, nil
		case !apierrors.IsNotFound(getErr):
			return nil, errkit.Wrap(getErr, "Failed to look up PVC referenced by sourcePVCName",
				"namespace", a.namespace, "pvcName", a.sourcePVCName)
		}
	}
	pvc, err := f.findStagingPVC(ctx, cli, a)
	if err != nil && a.sourcePVCName != "" {
		log.WithContext(ctx).Print("No live staging PVC; restoring fresh PVC from VolumeSnapshot",
			field.M{"sourcePVCName": a.sourcePVCName, "namespace": a.namespace})
		return f.restorePVCFromSnapshot(ctx, cli, a)
	}
	return pvc, err
}

// restorePVCFromSnapshot looks up the VolumeSnapshot whose source matches the
// original staging PVC name and provisions a fresh PVC from it.
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
	var match *stagingSnapshotRef
	var matches []string
	for i := range vsList.Items {
		vs := &vsList.Items[i]
		if vs.Spec.Source.PersistentVolumeClaimName == nil || *vs.Spec.Source.PersistentVolumeClaimName != a.sourcePVCName {
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

	size := a.size
	if size.IsZero() && match.restoreSize != nil {
		size = *match.restoreSize
	}
	if size.IsZero() {
		size = resource.MustParse(defaultBackupPVCSize)
	}

	pvcName := f.deriveRestoredPVCName(a)
	sc := a.storageClass
	apiGroup := "snapshot.storage.k8s.io"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: a.namespace,
			Labels:    restoredPVCLabels(a),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &sc,
			DataSource: &corev1.TypedLocalObjectReference{
				APIGroup: &apiGroup,
				Kind:     "VolumeSnapshot",
				Name:     match.name,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: size},
			},
		},
	}
	created, err := cli.CoreV1().PersistentVolumeClaims(a.namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create restored staging PVC",
			"namespace", a.namespace, "pvcName", pvcName, "volumeSnapshot", match.name)
	}
	if err := waitForPVCBound(ctx, cli, a.namespace, created.Name); err != nil {
		return nil, errkit.Wrap(err, "Restored staging PVC did not become Bound",
			"namespace", a.namespace, "pvcName", created.Name, "volumeSnapshot", match.name)
	}
	return created, nil
}

func (f *kubeTaskWithRestorePVCFunc) restorePVCFromNamedSnapshot(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithRestorePVCArgs) (*corev1.PersistentVolumeClaim, error) {
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
	if err := waitForPVCBound(ctx, cli, ns, pvcName); err != nil {
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

// resolveRestoreSize falls back from VS.status.restoreSize → restoreSize arg →
// size arg → defaultRestoreSize. Streaming CSI drivers often leave the snapshot
// status empty, so the explicit args are how the blueprint plumbs the value in.
func (f *kubeTaskWithRestorePVCFunc) resolveRestoreSize(ctx context.Context, cli kubernetes.Interface, ns string, a *kubeTaskWithRestorePVCArgs) resource.Quantity {
	if dynCli, err := kube.NewDynamicClient(); err == nil {
		snapshotter := snapshot.NewSnapshotter(cli, dynCli)
		if vs, getErr := snapshotter.Get(ctx, a.volumeSnapshotName, ns); getErr == nil &&
			vs.Status != nil && vs.Status.RestoreSize != nil && !vs.Status.RestoreSize.IsZero() {
			return *vs.Status.RestoreSize
		}
	}
	if !a.restoreSize.IsZero() {
		return a.restoreSize
	}
	if !a.size.IsZero() {
		return a.size
	}
	return resource.MustParse(defaultRestoreSize)
}

type stagingSnapshotRef struct {
	name        string
	restoreSize *resource.Quantity
}

func (f *kubeTaskWithRestorePVCFunc) deriveRestoredPVCName(a *kubeTaskWithRestorePVCArgs) string {
	base := a.workloadName
	if base == "" {
		base = "kanister"
	}
	return fmt.Sprintf("%s-restore-%s", base, rand.String(6))
}

func restoredPVCLabels(a *kubeTaskWithRestorePVCArgs) map[string]string {
	out := map[string]string{LabelKeyStagingPVC: "true"}
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

// findStagingPVC asserts exactly one PVC matches the selector.
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
