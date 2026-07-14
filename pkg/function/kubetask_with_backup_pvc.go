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
	"bufio"
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
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
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	KubeTaskWithBackupPVCFuncName = "KubeTaskWithBackupPVC"

	KubeTaskWithBackupPVCImageArg                   = "image"
	KubeTaskWithBackupPVCCommandArg                 = "command"
	KubeTaskWithBackupPVCEnvArg                     = "env"
	KubeTaskWithBackupPVCPathArg                    = "path"
	KubeTaskWithBackupPVCStorageClassArg            = "storageClassName"
	KubeTaskWithBackupPVCSizeArg                    = "size"
	KubeTaskWithBackupPVCPVCNameArg                 = "pvcName"
	KubeTaskWithBackupPVCNamespaceArg               = "namespace"
	KubeTaskWithBackupPVCServiceAccountArg          = "serviceAccountName"
	KubeTaskWithBackupPVCTimeoutArg                 = "timeout"
	KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg = "keepPodAliveForSnapshot"
	KubeTaskWithBackupPVCTakeSnapshotArg            = "takeSnapshot"
	KubeTaskWithBackupPVCSnapshotClassArg           = "snapshotClass"
	KubeTaskWithBackupPVCCleanupArg                 = "cleanup"

	// optionKeyCSIStorageClass is the ActionSet option the platform (e.g. K10)
	// injects to select the staging StorageClass dynamically per location
	// profile. When unset, defaultStorageClass falls back to the builtin below.
	optionKeyCSIStorageClass = "csiStorageClass"

	defaultBackupPVCStorageClass = "kopia-backup"
	defaultBackupPVCMountPath    = "/backup"
	defaultBackupPVCSize         = "1Ti"
	defaultBackupPVCTimeout      = 30 * time.Minute

	// keepAliveCommandDoneMarker is emitted by the wrapped command after the
	// user command exits; the function returns once it sees the marker even
	// though the container keeps sleeping to hold the FUSE mount alive for
	// the CSI snapshot.
	keepAliveCommandDoneMarker = "###KANISTER-COMMAND-DONE###"

	LabelKeyKeepAlivePod = "kanister.io/keep-alive-for-snapshot"
	backupPVCJobPrefix   = "kanister-backup-pvc-"
)

func init() {
	_ = kanister.Register(&kubeTaskWithBackupPVCFunc{})
}

// NewKubeTaskWithBackupPVCFunc returns a new instance of the generic
// KubeTaskWithBackupPVC function. Versioned overrides (e.g. a downstream
// v1.0.0-alpha registration) embed the returned value to reuse the generic
// backup orchestration while adding their own pre/post behaviour.
func NewKubeTaskWithBackupPVCFunc() kanister.Func {
	return &kubeTaskWithBackupPVCFunc{}
}

var _ kanister.Func = (*kubeTaskWithBackupPVCFunc)(nil)

type kubeTaskWithBackupPVCFunc struct {
	progressPercent string
}

func (*kubeTaskWithBackupPVCFunc) Name() string {
	return KubeTaskWithBackupPVCFuncName
}

type kubeTaskWithBackupPVCArgs struct {
	image               string
	command             []string
	env                 []corev1.EnvVar
	mountPath           string
	storageClass        string
	size                resource.Quantity
	pvcName             string
	namespace           string
	podOverride         crv1alpha1.JSONMap
	serviceAccount      string
	timeout             time.Duration
	keepPodAliveSeconds int
	takeSnapshot        bool
	snapshotClass       string
	cleanup             bool
	bpAnnotations       map[string]string
	bpLabels            map[string]string

	workloadName      string
	workloadNamespace string
	actionSetTag      string
}

// defaultStorageClass returns the staging StorageClass to use when the
// storageClassName arg is unset. The platform (e.g. K10) may inject a
// per-profile StorageClass via the csiStorageClass ActionSet option; when it
// is absent we fall back to the builtin default (kopia-backup).
func defaultStorageClass(tp param.TemplateParams, builtin string) string {
	if sc := tp.Options[optionKeyCSIStorageClass]; sc != "" {
		return sc
	}
	return builtin
}

func (*kubeTaskWithBackupPVCFunc) RequiredArgs() []string {
	return []string{
		KubeTaskWithBackupPVCImageArg,
		KubeTaskWithBackupPVCCommandArg,
	}
}

func (*kubeTaskWithBackupPVCFunc) Arguments() []string {
	return []string{
		KubeTaskWithBackupPVCImageArg,
		KubeTaskWithBackupPVCCommandArg,
		KubeTaskWithBackupPVCEnvArg,
		KubeTaskWithBackupPVCPathArg,
		KubeTaskWithBackupPVCStorageClassArg,
		KubeTaskWithBackupPVCSizeArg,
		KubeTaskWithBackupPVCPVCNameArg,
		KubeTaskWithBackupPVCNamespaceArg,
		KubeTaskWithBackupPVCServiceAccountArg,
		KubeTaskWithBackupPVCTimeoutArg,
		KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg,
		KubeTaskWithBackupPVCTakeSnapshotArg,
		KubeTaskWithBackupPVCSnapshotClassArg,
		KubeTaskWithBackupPVCCleanupArg,
		PodOverrideArg,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (f *kubeTaskWithBackupPVCFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(f.Name(), args); err != nil {
		return err
	}
	if err := utils.CheckSupportedArgs(f.Arguments(), args); err != nil {
		return err
	}
	if err := utils.CheckRequiredArgs(f.RequiredArgs(), args); err != nil {
		return err
	}
	return f.validateSnapshotArgs(args)
}

func (*kubeTaskWithBackupPVCFunc) validateSnapshotArgs(args map[string]any) error {
	takeSnapshot := true
	if v, ok := args[KubeTaskWithBackupPVCTakeSnapshotArg]; ok {
		if b, ok := v.(bool); ok {
			takeSnapshot = b
		}
	}
	cleanup := true
	if v, ok := args[KubeTaskWithBackupPVCCleanupArg]; ok {
		if b, ok := v.(bool); ok {
			cleanup = b
		}
	}
	_, snapshotClassSet := args[KubeTaskWithBackupPVCSnapshotClassArg]

	if takeSnapshot && !snapshotClassSet {
		return errkit.New("snapshotClass is required when takeSnapshot=true")
	}
	if !takeSnapshot && cleanup {
		return errkit.New("cleanup=true requires takeSnapshot=true; in hook patterns (takeSnapshot=false) the postBackupHook owns staging-PVC cleanup")
	}
	return nil
}

func (f *kubeTaskWithBackupPVCFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    f.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func (f *kubeTaskWithBackupPVCFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	f.progressPercent = progress.StartedPercent
	defer func() { f.progressPercent = progress.CompletedPercent }()

	parsed, err := f.parseArgs(ctx, tp, args)
	if err != nil {
		return nil, err
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}

	return f.run(ctx, cli, parsed)
}

func (f *kubeTaskWithBackupPVCFunc) parseArgs(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (*kubeTaskWithBackupPVCArgs, error) {
	parsed := &kubeTaskWithBackupPVCArgs{}
	if err := f.parseBackupCoreArgs(tp, args, parsed); err != nil {
		return nil, err
	}
	if err := f.parseBackupSnapshotArgs(args, parsed); err != nil {
		return nil, err
	}
	if err := f.parseBackupPodArgs(tp, args, parsed); err != nil {
		return nil, err
	}
	if err := f.resolveBackupContext(ctx, tp, parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (f *kubeTaskWithBackupPVCFunc) parseBackupCoreArgs(tp param.TemplateParams, args map[string]interface{}, parsed *kubeTaskWithBackupPVCArgs) error {
	var err error
	if err = Arg(args, KubeTaskWithBackupPVCImageArg, &parsed.image); err != nil {
		return err
	}
	if err = Arg(args, KubeTaskWithBackupPVCCommandArg, &parsed.command); err != nil {
		return err
	}
	if parsed.env, err = ParseEnvVars(args, KubeTaskWithBackupPVCEnvArg); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCPathArg, &parsed.mountPath, defaultBackupPVCMountPath); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCStorageClassArg, &parsed.storageClass, defaultStorageClass(tp, defaultBackupPVCStorageClass)); err != nil {
		return err
	}
	var sizeStr string
	if err = OptArg(args, KubeTaskWithBackupPVCSizeArg, &sizeStr, defaultBackupPVCSize); err != nil {
		return err
	}
	if parsed.size, err = resource.ParseQuantity(sizeStr); err != nil {
		return errkit.Wrap(err, "Failed to parse size", "size", sizeStr)
	}
	if err = OptArg(args, KubeTaskWithBackupPVCPVCNameArg, &parsed.pvcName, ""); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCNamespaceArg, &parsed.namespace, ""); err != nil {
		return err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCServiceAccountArg, &parsed.serviceAccount, ""); err != nil {
		return err
	}
	var timeoutStr string
	if err = OptArg(args, KubeTaskWithBackupPVCTimeoutArg, &timeoutStr, defaultBackupPVCTimeout.String()); err != nil {
		return err
	}
	if parsed.timeout, err = time.ParseDuration(timeoutStr); err != nil {
		return errkit.Wrap(err, "Failed to parse timeout", "timeout", timeoutStr)
	}
	return nil
}

func (f *kubeTaskWithBackupPVCFunc) parseBackupSnapshotArgs(args map[string]interface{}, parsed *kubeTaskWithBackupPVCArgs) error {
	if err := OptArg(args, KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg, &parsed.keepPodAliveSeconds, 0); err != nil {
		return err
	}
	if parsed.keepPodAliveSeconds < 0 {
		return errkit.New("keepPodAliveForSnapshot must be >= 0", "value", parsed.keepPodAliveSeconds)
	}
	if err := OptArg(args, KubeTaskWithBackupPVCTakeSnapshotArg, &parsed.takeSnapshot, true); err != nil {
		return err
	}
	if err := OptArg(args, KubeTaskWithBackupPVCSnapshotClassArg, &parsed.snapshotClass, ""); err != nil {
		return err
	}
	if err := OptArg(args, KubeTaskWithBackupPVCCleanupArg, &parsed.cleanup, true); err != nil {
		return err
	}
	// Mirror Validate() so unit tests that bypass Validate still fail loud.
	if parsed.takeSnapshot && parsed.snapshotClass == "" {
		return errkit.New("snapshotClass is required when takeSnapshot=true")
	}
	if !parsed.takeSnapshot && parsed.cleanup {
		return errkit.New("cleanup=true requires takeSnapshot=true; the postBackupHook owns staging-PVC cleanup in hook patterns")
	}
	// keepPodAliveForSnapshot is opt-in only. The backup-csi-driver's snapshot
	// registry is keyed off the node-level volume mount, not the calling pod's
	// process lifecycle (see NodeUnpublishVolume), so CreateSnapshot succeeds
	// against a Completed pod as long as the pod object itself isn't deleted
	// yet. We no longer force this on by default when takeSnapshot=true, so
	// command doesn't have to be the strict [bash|sh, -c, <script>] shape
	// unless the caller explicitly wants the keep-alive behavior.
	return nil
}

func (f *kubeTaskWithBackupPVCFunc) parseBackupPodArgs(tp param.TemplateParams, args map[string]interface{}, parsed *kubeTaskWithBackupPVCArgs) error {
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

func (f *kubeTaskWithBackupPVCFunc) resolveBackupContext(ctx context.Context, tp param.TemplateParams, parsed *kubeTaskWithBackupPVCArgs) error {
	parsed.workloadName, parsed.workloadNamespace = WorkloadFromTemplateParams(tp)
	if parsed.namespace == "" {
		parsed.namespace = parsed.workloadNamespace
	}
	if parsed.namespace == "" {
		return errkit.New("Unable to resolve namespace; pass the namespace arg or run the function against a workload action context")
	}
	if parsed.pvcName == "" {
		base := parsed.workloadName
		if base == "" {
			base = "kanister"
		}
		parsed.pvcName = fmt.Sprintf("%s-backup-%s", base, rand.String(6))
	}
	parsed.actionSetTag = ActionSetTagFromContext(ctx)
	if parsed.actionSetTag == "" {
		return errkit.New("Unable to read ActionSet name from context; required to stamp the owner-action label")
	}
	return nil
}

func (f *kubeTaskWithBackupPVCFunc) run(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithBackupPVCArgs) (out map[string]interface{}, retErr error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	pvc, err := f.createStagingPVC(ctx, cli, a)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create staging PVC",
			"namespace", a.namespace, "pvcName", a.pvcName, "storageClass", a.storageClass)
	}

	// PVC cleanup defer. Runs last (LIFO) so the pod-kill defer (registered
	// below) releases the FUSE mount first; otherwise PVC stays Terminating.
	defer func() {
		if !a.cleanup {
			return
		}
		if delErr := PVCGracefulDelete(ctx, cli, a.namespace, pvc.Name); delErr != nil {
			log.WithError(delErr).WithContext(ctx).Print("Failed to delete staging PVC",
				field.M{"namespace": a.namespace, "pvcName": pvc.Name, "snapshotTaken": a.takeSnapshot, "execErr": retErr != nil})
		}
	}()

	if err := WaitForPVCBound(ctx, cli, a.namespace, pvc.Name); err != nil {
		return nil, errkit.Wrap(err, "Staging PVC did not become Bound",
			"namespace", a.namespace, "pvcName", pvc.Name, "storageClass", a.storageClass)
	}

	podOpts := f.buildPodOptions(a, pvc.Name)
	if err := ephemeral.PodOptions.Apply(podOpts); err != nil {
		return nil, errkit.Wrap(err, "Failed to apply ephemeral pod options")
	}
	kube.AddLabelsToPodOptionsFromContext(ctx, podOpts, path.Join(consts.LabelPrefix, consts.LabelSuffixJobID))

	var (
		podOut       map[string]interface{}
		keepAlivePod string
	)
	if a.keepPodAliveSeconds > 0 {
		podOut, keepAlivePod, err = f.runWithKeepAlivePod(ctx, cli, podOpts, a)
	} else {
		pr := kube.NewPodRunner(cli, podOpts)
		podOut, err = pr.Run(ctx, StagingPodRunner("Backup pod did not complete successfully"))
	}

	// Pod-kill defer; LIFO before the PVC defer so the mount is released first.
	// keepAlivePod may be non-empty even on error (pod created but never Ready)
	// — clean those too. In hook mode the postBackupHook owns the pod.
	defer func() {
		if keepAlivePod == "" || !a.takeSnapshot {
			return
		}
		delCtx, delCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer delCancel()
		if delErr := cli.CoreV1().Pods(a.namespace).Delete(delCtx, keepAlivePod, metav1.DeleteOptions{}); delErr != nil {
			log.WithError(delErr).WithContext(ctx).Print("Failed to delete keep-alive pod",
				field.M{"namespace": a.namespace, "pod": keepAlivePod, "execErr": retErr != nil})
		}
	}()

	if err != nil {
		return nil, errkit.Wrap(err, "Backup command failed",
			"namespace", a.namespace, "pvcName", pvc.Name)
	}

	out = map[string]interface{}{
		OutputKeyStagingPVCName:      pvc.Name,
		OutputKeyStagingPVCNamespace: a.namespace,
	}
	for k, v := range podOut {
		if _, clash := out[k]; clash {
			continue
		}
		out[k] = v
	}

	if !a.takeSnapshot {
		return out, nil
	}

	snapInfo, err := f.takeStagingSnapshot(ctx, cli, a, pvc.Name)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to snapshot staging PVC",
			"namespace", a.namespace, "pvcName", pvc.Name, "snapshotClass", a.snapshotClass)
	}
	for k, v := range snapInfo {
		out[k] = v
	}
	return out, nil
}

// takeStagingSnapshot delegates to the package-private createCSISnapshot helper
// which calls snapshotter.Create with waitForReady=true and polls until
// readyToUse=true, Status.Error, or ctx cancel.
func (f *kubeTaskWithBackupPVCFunc) takeStagingSnapshot(
	ctx context.Context,
	cli kubernetes.Interface,
	a *kubeTaskWithBackupPVCArgs,
	pvcName string,
) (map[string]interface{}, error) {
	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create dynamic client for snapshot")
	}
	snapshotter := snapshot.NewSnapshotter(cli, dynCli)
	snapName := defaultSnapshotName(pvcName, 6)

	snapLabels := map[string]string{
		LabelKeyOwnerAction:       a.actionSetTag,
		LabelKeyWorkloadNamespace: a.workloadNamespace,
	}
	if a.workloadName != "" {
		snapLabels[LabelKeyWorkloadName] = a.workloadName
	}
	if a.workloadNamespace == "" {
		delete(snapLabels, LabelKeyWorkloadNamespace)
	}

	// Blocks until the snapshot is terminal; suspends the PVC-cleanup defer.
	vs, err := createCSISnapshot(ctx, snapshotter, snapName, a.namespace, pvcName, a.snapshotClass, true /* waitForReady */, snapLabels)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{
		OutputKeySnapshotName:      snapName,
		OutputKeySnapshotNamespace: a.namespace,
	}
	var snapRestoreSize *resource.Quantity
	if vs.Status != nil {
		snapRestoreSize = vs.Status.RestoreSize
	}
	out[OutputKeySnapshotRestoreSize] = f.deriveRestoreSize(ctx, cli, a.namespace, pvcName, snapRestoreSize)
	if vs.Status != nil && vs.Status.BoundVolumeSnapshotContentName != nil {
		out[OutputKeySnapshotContent] = *vs.Status.BoundVolumeSnapshotContentName
	}

	// Best-effort: surface the CSI snapshotHandle so blueprints can carry a
	// content-addressed identifier (kopia snapshot ID for backup-csi-driver)
	// for cross-cluster restore. Same-cluster restore via volumeSnapshotName
	// still works without it; log loud so failures are observable.
	src, srcErr := snapshotter.GetSource(ctx, snapName, a.namespace)
	switch {
	case srcErr != nil:
		log.WithError(srcErr).WithContext(ctx).Print(
			"Failed to read VolumeSnapshotContent source; cross-cluster restore will not be possible for this backup",
			field.M{"volumeSnapshotName": snapName, "namespace": a.namespace})
	case src == nil || src.Handle == "":
		log.WithContext(ctx).Print(
			"VolumeSnapshotContent source has no snapshotHandle; cross-cluster restore will not be possible for this backup",
			field.M{"volumeSnapshotName": snapName, "namespace": a.namespace})
	default:
		out[OutputKeySnapshotHandle] = src.Handle
	}
	return out, nil
}

// deriveRestoreSize falls back through snapshot.status.RestoreSize →
// PVC.status.Capacity → PVC.spec.Requests → defaultBackupPVCSize so the
// blueprint's restoreSize template always renders, even when streaming CSI
// drivers (kopia/backup-csi-driver) leave vs.Status.RestoreSize nil.
func (f *kubeTaskWithBackupPVCFunc) deriveRestoreSize(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace, pvcName string,
	snapRestoreSize *resource.Quantity,
) string {
	if snapRestoreSize != nil && !snapRestoreSize.IsZero() {
		return snapRestoreSize.String()
	}
	if pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{}); err == nil {
		if pvcCap, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok && !pvcCap.IsZero() {
			return pvcCap.String()
		}
		if req, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok && !req.IsZero() {
			return req.String()
		}
	}
	return defaultBackupPVCSize
}

func (f *kubeTaskWithBackupPVCFunc) createStagingPVC(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithBackupPVCArgs) (*corev1.PersistentVolumeClaim, error) {
	sc := a.storageClass
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.pvcName,
			Namespace: a.namespace,
			Labels:    stagingPVCLabels(a),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &sc,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: a.size,
				},
			},
		},
	}
	created, err := cli.CoreV1().PersistentVolumeClaims(a.namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func stagingPVCLabels(a *kubeTaskWithBackupPVCArgs) map[string]string {
	labels := map[string]string{
		LabelKeyIncludeInBackup:   "true",
		LabelKeyStagingPVC:        "true",
		LabelKeyOwnerAction:       a.actionSetTag,
		LabelKeyWorkloadNamespace: a.workloadNamespace,
	}
	if a.workloadName != "" {
		labels[LabelKeyWorkloadName] = a.workloadName
	}
	if a.workloadNamespace == "" {
		delete(labels, LabelKeyWorkloadNamespace)
	}
	for k, v := range a.bpLabels {
		labels[k] = v
	}
	return labels
}

// runWithKeepAlivePod runs the user command in a pod that stays alive past
// the command's exit (sleeps after emitting a marker). Returns (output, pod
// name, error). Caller is responsible for the pod's deletion.
func (f *kubeTaskWithBackupPVCFunc) runWithKeepAlivePod(
	ctx context.Context,
	cli kubernetes.Interface,
	podOpts *kube.PodOptions,
	a *kubeTaskWithBackupPVCArgs,
) (map[string]interface{}, string, error) {
	wrapped, err := wrapCommandForKeepAlive(podOpts.Command, a.keepPodAliveSeconds)
	if err != nil {
		return nil, "", err
	}
	podOpts.Command = wrapped

	if podOpts.Labels == nil {
		podOpts.Labels = map[string]string{}
	}
	podOpts.Labels[LabelKeyKeepAlivePod] = "true"
	if a.workloadName != "" {
		podOpts.Labels[LabelKeyWorkloadName] = a.workloadName
	}
	if a.workloadNamespace != "" {
		podOpts.Labels[LabelKeyWorkloadNamespace] = a.workloadNamespace
	}

	pc := kube.NewPodController(cli, podOpts)
	if err := pc.StartPod(ctx); err != nil {
		return nil, "", errkit.Wrap(err, "Failed to create keep-alive pod", "namespace", a.namespace)
	}
	pod := pc.Pod()
	log.WithContext(ctx).Print("Created keep-alive backup pod",
		field.M{"pod": pod.Name, "namespace": pod.Namespace, "keepAliveSeconds": a.keepPodAliveSeconds})

	if err := pc.WaitForPodReady(ctx); err != nil {
		return nil, pod.Name, errkit.Wrap(err, "Keep-alive pod did not become ready", "pod", pc.PodName())
	}

	streamCtx := field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
	r, err := pc.StreamPodLogs(streamCtx)
	if err != nil {
		return nil, pod.Name, errkit.Wrap(err, "Failed to stream logs from keep-alive pod", "pod", pc.PodName())
	}
	defer r.Close() //nolint:errcheck

	exitCode, captured, err := waitForKeepAliveMarker(streamCtx, r)
	if err != nil {
		return nil, pod.Name, errkit.Wrap(err, "Failed waiting for command-done marker", "pod", pc.PodName())
	}
	if exitCode != 0 {
		return nil, pod.Name, errkit.New("Backup command exited non-zero inside keep-alive pod",
			"pod", pc.PodName(), "exitCode", exitCode)
	}
	parsedOut, err := output.LogAndParse(streamCtx, io.NopCloser(strings.NewReader(captured)))
	if err != nil {
		return nil, pod.Name, errkit.Wrap(err, "Failed to parse output from keep-alive pod", "pod", pc.PodName())
	}
	return parsedOut, pod.Name, nil
}

// wrapCommandForKeepAlive composes the user command into a shell pipeline
// that prints a marker (with the user command's exit code) and then sleeps,
// holding the mount alive. Only supports the [bash|sh, -c, <script>] form.
func wrapCommandForKeepAlive(orig []string, seconds int) ([]string, error) {
	if len(orig) < 3 {
		return nil, errkit.New("keepPodAliveForSnapshot requires command of form [bash|sh, -c, <script>]",
			"command", orig)
	}
	shell := orig[0]
	flag := orig[1]
	switch shell {
	case "bash", "sh", "/bin/bash", "/bin/sh":
	default:
		return nil, errkit.New("keepPodAliveForSnapshot requires shell to be bash or sh", "shell", shell)
	}
	if flag != "-c" {
		return nil, errkit.New("keepPodAliveForSnapshot requires shell flag to be -c", "flag", flag)
	}
	user := orig[2]
	wrapped := fmt.Sprintf(
		"set +e\n( %s )\nKANISTER_RC=$?\nprintf '%%s:%%d\\n' '%s' \"$KANISTER_RC\"\nsleep %d\nexit \"$KANISTER_RC\"\n",
		user, keepAliveCommandDoneMarker, seconds,
	)
	return []string{shell, "-c", wrapped}, nil
}

// waitForKeepAliveMarker scans pod log lines until the marker is seen and
// returns the embedded exit code plus everything emitted before the marker
// (for downstream kando-output parsing).
func waitForKeepAliveMarker(ctx context.Context, r io.Reader) (int, string, error) {
	const maxLine = 1 * 1024 * 1024
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxLine)
	var before strings.Builder
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return 0, "", ctx.Err()
		default:
		}
		line := scanner.Text()
		if idx := strings.Index(line, keepAliveCommandDoneMarker+":"); idx >= 0 {
			ecStr := strings.TrimSpace(line[idx+len(keepAliveCommandDoneMarker)+1:])
			ec, err := strconv.Atoi(ecStr)
			if err != nil {
				return 0, "", errkit.Wrap(err, "Failed to parse exit code from marker", "line", line)
			}
			return ec, before.String(), nil
		}
		before.WriteString(line)
		before.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return 0, "", errkit.Wrap(err, "Log stream ended before marker was seen")
	}
	return 0, "", errkit.New("Log stream closed before keep-alive marker was emitted; pod may have crashed")
}

func (f *kubeTaskWithBackupPVCFunc) buildPodOptions(a *kubeTaskWithBackupPVCArgs, pvcName string) *kube.PodOptions {
	return &kube.PodOptions{
		Namespace:          a.namespace,
		GenerateName:       backupPVCJobPrefix,
		Image:              a.image,
		Command:            a.command,
		ServiceAccountName: a.serviceAccount,
		Volumes: map[string]kube.VolumeMountOptions{
			pvcName: {
				MountPath: a.mountPath,
				ReadOnly:  false,
			},
		},
		EnvironmentVariables: a.env,
		PodOverride:          a.podOverride,
		Annotations:          a.bpAnnotations,
		Labels:               a.bpLabels,
	}
}
