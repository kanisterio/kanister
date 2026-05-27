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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// KubeTaskWithRestorePVCFuncName is the registered Kanister function name.
	KubeTaskWithRestorePVCFuncName = "KubeTaskWithRestorePVC"

	KubeTaskWithRestorePVCImageArg          = "image"
	KubeTaskWithRestorePVCCommandArg        = "command"
	KubeTaskWithRestorePVCEnvFromSecretArg  = "envFromSecret"
	KubeTaskWithRestorePVCEnvArg            = "env"
	KubeTaskWithRestorePVCPathArg           = "path"
	KubeTaskWithRestorePVCStorageClassArg   = "storageClassName"
	KubeTaskWithRestorePVCPVCSelectorArg    = "pvcSelector"
	KubeTaskWithRestorePVCNamespaceArg      = "namespace"
	KubeTaskWithRestorePVCServiceAccountArg = "serviceAccountName"
	KubeTaskWithRestorePVCTimeoutArg        = "timeout"
	KubeTaskWithRestorePVCCleanupPVCArg     = "cleanupPVC"
	// KubeTaskWithRestorePVCSourcePVCNameArg is the original (backup-side) staging
	// PVC name. When set (typically via the backupPrehook's published artifact:
	// `{{ index .ArtifactsIn.stagingPVC.KeyValue "pvcName" }}`), and no live PVC
	// matches the label selector, the function falls back to locating the
	// VolumeSnapshot whose `.spec.source.persistentVolumeClaimName` equals this
	// value and creating a fresh PVC from it. This makes the function the sole
	// owner of staging-PVC lifecycle on both backup and restore sides.
	KubeTaskWithRestorePVCSourcePVCNameArg = "sourcePVCName"
	// KubeTaskWithRestorePVCSizeArg is the size for the freshly-created PVC when
	// restoring from a VolumeSnapshot. Defaults to the snapshot's RestoreSize.
	KubeTaskWithRestorePVCSizeArg = "size"

	defaultRestorePVCStorageClass = "kopia-restore"
	defaultRestorePVCMountPath    = "/restore"
	defaultRestorePVCTimeout      = 30 * time.Minute

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
	image          string
	command        []string
	envFromSecret  string
	env            []corev1.EnvVar
	mountPath      string
	storageClass   string
	pvcSelector    metav1.LabelSelector
	namespace      string
	podOverride    crv1alpha1.JSONMap
	serviceAccount string
	timeout        time.Duration
	cleanupPVC     bool
	bpAnnotations  map[string]string
	bpLabels       map[string]string
	// sourcePVCName is the original (backup-side) staging PVC name; when set and
	// no live PVC matches the selector, the function restores a fresh PVC from
	// the VolumeSnapshot whose source matches this name.
	sourcePVCName string
	// size optionally overrides the size used when creating the fresh PVC from
	// a snapshot. When empty, the snapshot's RestoreSize is used.
	size resource.Quantity

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
		KubeTaskWithRestorePVCEnvFromSecretArg,
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
	var err error

	if err = Arg(args, KubeTaskWithRestorePVCImageArg, &parsed.image); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeTaskWithRestorePVCCommandArg, &parsed.command); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCEnvFromSecretArg, &parsed.envFromSecret, ""); err != nil {
		return nil, err
	}
	if parsed.env, err = parseEnvVars(args, KubeTaskWithRestorePVCEnvArg); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCPathArg, &parsed.mountPath, defaultRestorePVCMountPath); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCStorageClassArg, &parsed.storageClass, defaultRestorePVCStorageClass); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCPVCSelectorArg, &parsed.pvcSelector, metav1.LabelSelector{}); err != nil {
		return nil, errkit.Wrap(err, "Failed to parse pvcSelector")
	}
	if err = OptArg(args, KubeTaskWithRestorePVCNamespaceArg, &parsed.namespace, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCServiceAccountArg, &parsed.serviceAccount, ""); err != nil {
		return nil, err
	}
	var timeoutStr string
	if err = OptArg(args, KubeTaskWithRestorePVCTimeoutArg, &timeoutStr, defaultRestorePVCTimeout.String()); err != nil {
		return nil, err
	}
	if parsed.timeout, err = time.ParseDuration(timeoutStr); err != nil {
		return nil, errkit.Wrap(err, "Failed to parse timeout", "timeout", timeoutStr)
	}
	if err = OptArg(args, KubeTaskWithRestorePVCCleanupPVCArg, &parsed.cleanupPVC, true); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithRestorePVCSourcePVCNameArg, &parsed.sourcePVCName, ""); err != nil {
		return nil, err
	}
	var sizeStr string
	if err = OptArg(args, KubeTaskWithRestorePVCSizeArg, &sizeStr, ""); err != nil {
		return nil, err
	}
	if sizeStr != "" {
		if parsed.size, err = resource.ParseQuantity(sizeStr); err != nil {
			return nil, errkit.Wrap(err, "Failed to parse size", "size", sizeStr)
		}
	}
	if err = OptArg(args, PodAnnotationsArg, &parsed.bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &parsed.bpLabels, nil); err != nil {
		return nil, err
	}
	if parsed.podOverride, err = GetPodSpecOverride(tp, args, PodOverrideArg); err != nil {
		return nil, err
	}

	parsed.workloadName, parsed.workloadNamespace = workloadFromTemplateParams(tp)

	if parsed.namespace == "" {
		parsed.namespace = parsed.workloadNamespace
	}
	if parsed.namespace == "" {
		return nil, errkit.New("Unable to resolve namespace; pass the namespace arg or run the function against a workload action context")
	}

	// If the caller passed no selector, derive the default Kasten-native selector
	// from the workload context. We require at least one resolved label here so
	// we never accidentally match a foreign PVC. When sourcePVCName is set the
	// function can recover by restoring from a VolumeSnapshot, so a missing
	// workload context is recoverable.
	if len(parsed.pvcSelector.MatchLabels) == 0 && len(parsed.pvcSelector.MatchExpressions) == 0 {
		matchLabels := map[string]string{
			LabelKeyStagingPVC: "true",
		}
		if parsed.workloadName != "" {
			matchLabels[LabelKeyWorkloadName] = parsed.workloadName
		}
		if parsed.workloadNamespace != "" {
			matchLabels[LabelKeyWorkloadNamespace] = parsed.workloadNamespace
		}
		if len(matchLabels) == 1 && parsed.sourcePVCName == "" {
			return nil, errkit.New("Unable to derive default pvcSelector: no workload context. Pass an explicit pvcSelector arg or sourcePVCName arg.")
		}
		parsed.pvcSelector = metav1.LabelSelector{MatchLabels: matchLabels}
	}

	return parsed, nil
}

func (f *kubeTaskWithRestorePVCFunc) run(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithRestorePVCArgs) (out map[string]interface{}, retErr error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// First look for an existing PVC by selector. This preserves the
	// "Kasten-restores-the-PVC, function-discovers-it" path used when K10
	// recreates the staging PVC for us.
	pvc, err := f.findStagingPVC(ctx, cli, a)
	if err != nil && a.sourcePVCName != "" {
		// Fall back to the "function owns staging-PVC lifecycle on restore too"
		// path: locate the VolumeSnapshot whose source matched the original
		// staging PVC name and provision a fresh PVC from it.
		log.WithContext(ctx).Print("No live staging PVC; restoring fresh PVC from VolumeSnapshot",
			field.M{"sourcePVCName": a.sourcePVCName, "namespace": a.namespace})
		pvc, err = f.restorePVCFromSnapshot(ctx, cli, a)
	}
	if err != nil {
		return nil, err
	}

	// On any exit (success or failure), delete the staging PVC if the caller
	// asked for it (default true). The Kopia snapshot in S3 and the
	// VolumeSnapshot reference in the RestorePoint persist regardless.
	defer func() {
		if !a.cleanupPVC {
			return
		}
		if delErr := pvcGracefulDelete(ctx, cli, a.namespace, pvc.Name); delErr != nil {
			log.WithError(delErr).WithContext(ctx).Print("Failed to delete restored staging PVC",
				field.M{"namespace": a.namespace, "pvcName": pvc.Name})
		}
	}()

	podOpts, err := f.buildPodOptions(a, pvc.Name)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to build pod options", "pvcName", pvc.Name)
	}
	if err := ephemeral.PodOptions.Apply(podOpts); err != nil {
		return nil, errkit.Wrap(err, "Failed to apply ephemeral pod options")
	}
	kube.AddLabelsToPodOptionsFromContext(ctx, podOpts, path.Join(consts.LabelPrefix, consts.LabelSuffixJobID))

	pr := kube.NewPodRunner(cli, podOpts)
	podOut, err := pr.Run(ctx, kubeTaskWithRestorePVCPodFunc())
	if err != nil {
		return nil, errkit.Wrap(err, "Restore command failed",
			"namespace", a.namespace, "pvcName", pvc.Name)
	}
	return podOut, nil
}

// restorePVCFromSnapshot locates the VolumeSnapshot whose source matches the
// original (backup-side) staging PVC name carried in the artifact, then
// provisions a fresh PVC bound to that snapshot. The restored PVC is stamped
// with our staging labels so any downstream tooling can still find it.
//
// The cluster keeps the VolumeSnapshot alive for the lifetime of the RestorePoint
// even though the source PVC was deleted by the posthook, so this lookup is
// reliable as long as the user is restoring a RestorePoint that still exists.
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
	var match *snapshotName
	for i := range vsList.Items {
		vs := &vsList.Items[i]
		if vs.Spec.Source.PersistentVolumeClaimName == nil {
			continue
		}
		if *vs.Spec.Source.PersistentVolumeClaimName != a.sourcePVCName {
			continue
		}
		if vs.Status == nil || vs.Status.ReadyToUse == nil || !*vs.Status.ReadyToUse {
			// Snapshot exists but is not ready; record and continue so we can
			// fail loudly if no ready snapshot matches.
			matches = append(matches, vs.Name+"(notReady)")
			continue
		}
		matches = append(matches, vs.Name)
		if match == nil {
			match = &snapshotName{name: vs.Name, restoreSize: vs.Status.RestoreSize}
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

	// Determine PVC size. Prefer the explicit `size` arg if given; otherwise
	// use the snapshot's RestoreSize; otherwise fall back to a sensible default.
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
		field.M{"namespace": a.namespace, "pvcName": created.Name, "volumeSnapshot": match.name, "storageClass": sc, "size": size.String()})

	if err := waitForPVCBound(ctx, cli, a.namespace, created.Name); err != nil {
		return nil, errkit.Wrap(err, "Restored staging PVC did not become Bound",
			"namespace", a.namespace, "pvcName", created.Name, "volumeSnapshot", match.name)
	}
	return created, nil
}

// snapshotName is a small carrier so we can pass both the chosen VolumeSnapshot
// name and its restore size out of the matching loop in one value.
type snapshotName struct {
	name        string
	restoreSize *resource.Quantity
}

// deriveRestoredPVCName produces a deterministic-ish name for the freshly
// restored PVC. Keeps the workload prefix so the PVC is easy to recognise in
// `kubectl get pvc` listings.
func (f *kubeTaskWithRestorePVCFunc) deriveRestoredPVCName(a *kubeTaskWithRestorePVCArgs) string {
	base := a.workloadName
	if base == "" {
		base = "kanister"
	}
	return fmt.Sprintf("%s-restore-%s", base, rand.String(6))
}

// restoredPVCLabels stamps the staging-PVC labels so the rest of the system
// (cleanup scripts, alternate restore tools) can still discover the PVC.
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

// findStagingPVC asserts exactly one PVC in the namespace matches the selector.
// Zero or multiple matches return a diagnostic naming what was found.
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

func (f *kubeTaskWithRestorePVCFunc) buildPodOptions(a *kubeTaskWithRestorePVCArgs, pvcName string) (*kube.PodOptions, error) {
	annotations := a.bpAnnotations
	labels := a.bpLabels

	opts := &kube.PodOptions{
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
		Annotations:          annotations,
		Labels:               labels,
	}
	if a.envFromSecret != "" {
		opts.EnvFromSources = []corev1.EnvFromSource{
			{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: a.envFromSecret},
				},
			},
		}
	}
	return opts, nil
}

// pvcSelectorEnsureLabel exists so a future caller can extend the default
// selector without re-deriving the workload context. Unused today but kept
// to mirror similar helpers in kanister.
var _ = pvcSelectorEnsureLabel

func pvcSelectorEnsureLabel(sel *metav1.LabelSelector, key, value string) {
	if sel.MatchLabels == nil {
		sel.MatchLabels = map[string]string{}
	}
	sel.MatchLabels[key] = value
}

// labelsFromSelectorString is a sanity helper for tests / diagnostics that need
// to round-trip a selector representation.
func labelsFromSelectorString(s string) labels.Set {
	parsed, err := labels.ConvertSelectorToLabelsMap(s)
	if err != nil {
		return labels.Set{}
	}
	return parsed
}

var _ = labelsFromSelectorString

func kubeTaskWithRestorePVCPodFunc() func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
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
			return nil, errkit.Wrap(err, "Restore pod did not complete successfully", "pod", pc.PodName())
		}
		return out, nil
	}
}
