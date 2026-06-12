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
	// KubeTaskWithBackupPVCFuncName is the registered Kanister function name.
	KubeTaskWithBackupPVCFuncName = "KubeTaskWithBackupPVC"

	KubeTaskWithBackupPVCImageArg            = "image"
	KubeTaskWithBackupPVCCommandArg          = "command"
	KubeTaskWithBackupPVCEnvArg              = "env"
	KubeTaskWithBackupPVCPathArg             = "path"
	KubeTaskWithBackupPVCStorageClassArg     = "storageClassName"
	KubeTaskWithBackupPVCSizeArg             = "size"
	KubeTaskWithBackupPVCPVCNameArg          = "pvcName"
	KubeTaskWithBackupPVCNamespaceArg        = "namespace"
	KubeTaskWithBackupPVCServiceAccountArg   = "serviceAccountName"
	KubeTaskWithBackupPVCTimeoutArg          = "timeout"
	KubeTaskWithBackupPVCKeepPVCOnFailureArg = "keepPVCOnFailure"
	// KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg is a workaround for CSI
	// drivers that require the volume to be actively mounted at the moment of
	// CreateSnapshot. When > 0, the function wraps the user's command with a
	// completion marker + sleep so the pod stays alive holding the mount past
	// command exit. The function returns success when the marker appears in the
	// pod logs; the keep-alive pod is cleaned up by the backupPosthook.
	// Only meaningful when `takeSnapshot=false`; mutually exclusive with
	// `takeSnapshot=true` (validated at Validate()).
	KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg = "keepPodAliveForSnapshot"

	// KubeTaskWithBackupPVCTakeSnapshotArg controls whether THIS function
	// drives the CSI snapshot of the staging PVC, or whether it leaves the
	// PVC alive for an external snapshot phase (K10's snapshotVolumes phase
	// under ActionHooks / BlueprintBinding wiring).
	//
	// Default: true. When set, the function calls CreateCSISnapshot's
	// internal helper after the user command exits, waits for the snapshot
	// to reach a terminal state (readyToUse=true OR Status.Error set OR
	// ctx cancel), and returns the snapshot info as part of the output map.
	// Suitable for blueprints using the `actions.backup` action TYPE.
	//
	// When false, the function returns immediately after the user command
	// exits and the caller (postBackupHook) is responsible for cleaning up
	// the PVC. Suitable for blueprints using ActionHooks/BlueprintBinding.
	KubeTaskWithBackupPVCTakeSnapshotArg = "takeSnapshot"
	// KubeTaskWithBackupPVCSnapshotClassArg names the VolumeSnapshotClass to
	// use when `takeSnapshot=true`. Required in that mode; rejected otherwise.
	KubeTaskWithBackupPVCSnapshotClassArg = "snapshotClass"
	// KubeTaskWithBackupPVCCleanupArg controls whether the staging PVC is
	// deleted at the end of Exec. Default: true. Combined with the existing
	// `keepPVCOnFailure` arg:
	//   cleanup=true,  keepPVCOnFailure=false  → always delete (default)
	//   cleanup=true,  keepPVCOnFailure=true   → delete on success, keep on failure
	//   cleanup=false                          → never delete (debug)
	// When `takeSnapshot=false`, cleanup MUST be false; the postBackupHook owns
	// cleanup in that path (Validate enforces this).
	KubeTaskWithBackupPVCCleanupArg = "cleanup"

	// Defaults match the backup-csi-driver shipped storage class and the
	// path the example blueprints expect.
	defaultBackupPVCStorageClass = "kopia-backup"
	defaultBackupPVCMountPath    = "/backup"
	defaultBackupPVCSize         = "1Ti"
	defaultBackupPVCTimeout      = 30 * time.Minute

	// keepAliveCommandDoneMarker is emitted to the pod's stdout immediately after
	// the user's command exits. The function watches the log stream for this
	// marker and returns once observed, even though the container keeps sleeping.
	keepAliveCommandDoneMarker = "###KANISTER-COMMAND-DONE###"
	// LabelKeyKeepAlivePod is stamped on the keep-alive pod so the posthook can
	// find and delete it by selector.
	LabelKeyKeepAlivePod = "kanister.io/keep-alive-for-snapshot"

	backupPVCJobPrefix = "kanister-backup-pvc-"
)

func init() {
	_ = kanister.Register(&kubeTaskWithBackupPVCFunc{})
}

var _ kanister.Func = (*kubeTaskWithBackupPVCFunc)(nil)

type kubeTaskWithBackupPVCFunc struct {
	progressPercent string
}

func (*kubeTaskWithBackupPVCFunc) Name() string {
	return KubeTaskWithBackupPVCFuncName
}

type kubeTaskWithBackupPVCArgs struct {
	image            string
	command          []string
	env              []corev1.EnvVar
	mountPath        string
	storageClass     string
	size             resource.Quantity
	pvcName          string
	namespace        string
	podOverride      crv1alpha1.JSONMap
	serviceAccount   string
	timeout          time.Duration
	keepPVCOnFailure bool
	// keepPodAliveSeconds, when > 0, keeps the pod alive for this many seconds
	// past the user command's exit so the CSI driver still sees the volume
	// mounted at snapshot time. Zero disables the keep-alive path entirely.
	keepPodAliveSeconds int
	// takeSnapshot controls whether this function drives the CSI snapshot
	// itself (true) or leaves the PVC alive for an external snapshot phase
	// (false). See KubeTaskWithBackupPVCTakeSnapshotArg doc.
	takeSnapshot bool
	// snapshotClass is the VolumeSnapshotClass name; required when takeSnapshot
	// is true, rejected otherwise.
	snapshotClass string
	// cleanup controls whether the staging PVC is deleted at the end of Exec.
	cleanup       bool
	bpAnnotations map[string]string
	bpLabels      map[string]string

	workloadName      string
	workloadNamespace string
	// actionSetTag scopes the staging-PVC owner-action label to a specific
	// ActionSet. Sourced from the controller-injected `ActionsetNameKey` in
	// the phase context (Kanister already plumbs this).
	actionSetTag string
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
		KubeTaskWithBackupPVCKeepPVCOnFailureArg,
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

// validateSnapshotArgs enforces the configuration rules for the
// snapshot-related args. Failures here surface at admission time (webhook),
// not at Exec time — they are configuration errors, not runtime errors.
//
// Note on keep-alive vs takeSnapshot interaction:
//   - When takeSnapshot=true, the function INTERNALLY uses the keep-alive
//     path to hold the volume mount until CreateSnapshot returns terminal
//     state, then actively deletes the pod. The CSI driver requires a live
//     mount at snapshot time. Users may explicitly set keepPodAliveForSnapshot
//     to override the safety-net duration; if omitted, the function defaults
//     it to the function timeout. The two are NOT mutually exclusive.
//   - When takeSnapshot=false, keep-alive is required when an EXTERNAL phase
//     will fire CreateSnapshot. The user supplies it via the arg as before.
func (*kubeTaskWithBackupPVCFunc) validateSnapshotArgs(args map[string]any) error {
	takeSnapshot := true // default
	if v, ok := args[KubeTaskWithBackupPVCTakeSnapshotArg]; ok {
		if b, ok := v.(bool); ok {
			takeSnapshot = b
		}
	}
	cleanup := true // default
	if v, ok := args[KubeTaskWithBackupPVCCleanupArg]; ok {
		if b, ok := v.(bool); ok {
			cleanup = b
		}
	}
	_, snapshotClassSet := args[KubeTaskWithBackupPVCSnapshotClassArg]

	// takeSnapshot=true requires snapshotClass.
	if takeSnapshot && !snapshotClassSet {
		return errkit.New("snapshotClass is required when takeSnapshot=true")
	}

	// takeSnapshot=false forbids cleanup=true. With takeSnapshot=false,
	// an external snapshot phase needs the PVC alive after this function returns;
	// deleting it here would break that flow. The postBackupHook owns cleanup.
	if !takeSnapshot && cleanup {
		return errkit.New(
			"cleanup=true requires takeSnapshot=true; in hook patterns " +
				"(takeSnapshot=false) the postBackupHook owns staging-PVC cleanup",
		)
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
	var err error

	if err = Arg(args, KubeTaskWithBackupPVCImageArg, &parsed.image); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeTaskWithBackupPVCCommandArg, &parsed.command); err != nil {
		return nil, err
	}
	if parsed.env, err = parseEnvVars(args, KubeTaskWithBackupPVCEnvArg); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCPathArg, &parsed.mountPath, defaultBackupPVCMountPath); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCStorageClassArg, &parsed.storageClass, defaultBackupPVCStorageClass); err != nil {
		return nil, err
	}
	var sizeStr string
	if err = OptArg(args, KubeTaskWithBackupPVCSizeArg, &sizeStr, defaultBackupPVCSize); err != nil {
		return nil, err
	}
	if parsed.size, err = resource.ParseQuantity(sizeStr); err != nil {
		return nil, errkit.Wrap(err, "Failed to parse size", "size", sizeStr)
	}
	if err = OptArg(args, KubeTaskWithBackupPVCPVCNameArg, &parsed.pvcName, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCNamespaceArg, &parsed.namespace, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCServiceAccountArg, &parsed.serviceAccount, ""); err != nil {
		return nil, err
	}
	var timeoutStr string
	if err = OptArg(args, KubeTaskWithBackupPVCTimeoutArg, &timeoutStr, defaultBackupPVCTimeout.String()); err != nil {
		return nil, err
	}
	if parsed.timeout, err = time.ParseDuration(timeoutStr); err != nil {
		return nil, errkit.Wrap(err, "Failed to parse timeout", "timeout", timeoutStr)
	}
	if err = OptArg(args, KubeTaskWithBackupPVCKeepPVCOnFailureArg, &parsed.keepPVCOnFailure, false); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg, &parsed.keepPodAliveSeconds, 0); err != nil {
		return nil, err
	}
	if parsed.keepPodAliveSeconds < 0 {
		return nil, errkit.New("keepPodAliveForSnapshot must be >= 0", "value", parsed.keepPodAliveSeconds)
	}
	if err = OptArg(args, KubeTaskWithBackupPVCTakeSnapshotArg, &parsed.takeSnapshot, true); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCSnapshotClassArg, &parsed.snapshotClass, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskWithBackupPVCCleanupArg, &parsed.cleanup, true); err != nil {
		return nil, err
	}
	// Mirror Validate()'s rules so the same errors fire even when callers
	// bypass Validate (e.g. unit tests building args by hand).
	if parsed.takeSnapshot && parsed.snapshotClass == "" {
		return nil, errkit.New("snapshotClass is required when takeSnapshot=true")
	}
	if !parsed.takeSnapshot && parsed.cleanup {
		return nil, errkit.New(
			"cleanup=true requires takeSnapshot=true; in hook patterns the " +
				"postBackupHook owns staging-PVC cleanup",
		)
	}
	// takeSnapshot=true requires the volume to be mounted at CreateSnapshot
	// time (CSI driver constraint). Default the keep-alive sleep to the
	// function timeout so the pod is guaranteed alive while we drive the
	// snapshot. The function will actively delete the pod once the snapshot
	// reaches a terminal state — the sleep is a safety net for crash paths.
	if parsed.takeSnapshot && parsed.keepPodAliveSeconds == 0 {
		parsed.keepPodAliveSeconds = int(parsed.timeout.Seconds())
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

	if parsed.pvcName == "" {
		base := parsed.workloadName
		if base == "" {
			base = "kanister"
		}
		parsed.pvcName = fmt.Sprintf("%s-backup-%s", base, rand.String(6))
	}

	parsed.actionSetTag = actionSetTagFromContext(ctx)
	if parsed.actionSetTag == "" {
		return nil, errkit.New("Unable to read ActionSet name from context; required to stamp the owner-action label")
	}

	return parsed, nil
}

func (f *kubeTaskWithBackupPVCFunc) run(ctx context.Context, cli kubernetes.Interface, a *kubeTaskWithBackupPVCArgs) (out map[string]interface{}, retErr error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	pvc, err := f.createStagingPVC(ctx, cli, a)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create staging PVC",
			"namespace", a.namespace, "pvcName", a.pvcName, "storageClass", a.storageClass)
	}

	// Deferred staging-PVC cleanup. By the time this fires we are guaranteed
	// to be past any in-flight snapshot work (the snapshot call below blocks
	// until the snapshot reaches a terminal state — readyToUse=true, Status.Error
	// set, or ctx cancel — before returning, so PVC deletion is never
	// attempted while a snapshot is still being finalized).
	//
	// The three cases the cleanup honours:
	//   1. a.cleanup=false        → never delete (debug)
	//   2. retErr != nil and a.keepPVCOnFailure=true → leave for debug
	//   3. otherwise              → delete (success or failure)
	//
	// When a.takeSnapshot=false, Validate forces a.cleanup=false; this defer
	// is then a no-op and the postBackupHook owns cleanup.
	defer func() {
		if !a.cleanup {
			return
		}
		if retErr != nil && a.keepPVCOnFailure {
			log.WithContext(ctx).Print("Leaving staging PVC alive for debugging (keepPVCOnFailure=true)",
				field.M{"namespace": a.namespace, "pvcName": pvc.Name})
			return
		}
		if delErr := pvcGracefulDelete(ctx, cli, a.namespace, pvc.Name); delErr != nil {
			log.WithError(delErr).WithContext(ctx).Print("Failed to delete staging PVC",
				field.M{"namespace": a.namespace, "pvcName": pvc.Name, "snapshotTaken": a.takeSnapshot, "execErr": retErr != nil})
		}
	}()

	if err := waitForPVCBound(ctx, cli, a.namespace, pvc.Name); err != nil {
		return nil, errkit.Wrap(err, "Staging PVC did not become Bound",
			"namespace", a.namespace, "pvcName", pvc.Name, "storageClass", a.storageClass)
	}

	podOpts, err := f.buildPodOptions(a, pvc.Name)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to build pod options", "pvcName", pvc.Name)
	}
	if err := ephemeral.PodOptions.Apply(podOpts); err != nil {
		return nil, errkit.Wrap(err, "Failed to apply ephemeral pod options")
	}
	kube.AddLabelsToPodOptionsFromContext(ctx, podOpts, path.Join(consts.LabelPrefix, consts.LabelSuffixJobID))

	// keepPodAliveForSnapshot path: wrap command so the pod stays alive holding
	// the volume mount past command exit. The CSI driver requires a live mount
	// at CreateSnapshot time. The function will actively delete the pod via a
	// deferred cleanup on every exit path — the sleep duration inside the
	// wrapped command is just a safety net for crash paths.
	//
	// In hook mode (takeSnapshot=false) the external posthook deletes the
	// keep-alive pod via its label, so the function-owned defer is a no-op
	// in that branch. In function-owned mode (takeSnapshot=true) parseArgs()
	// defaulted keepPodAliveSeconds to the timeout so this path is always taken.
	var (
		podOut       map[string]interface{}
		keepAlivePod string // name of the keep-alive pod we need to delete after snapshot
	)
	if a.keepPodAliveSeconds > 0 {
		podOut, keepAlivePod, err = f.runWithKeepAlivePod(ctx, cli, podOpts, a)
	} else {
		pr := kube.NewPodRunner(cli, podOpts)
		podOut, err = pr.Run(ctx, kubeTaskWithBackupPVCPodFunc())
	}

	// Defer keep-alive pod deletion on EVERY exit path (success, error, panic).
	// Registered AFTER the PVC cleanup defer → fires BEFORE it (LIFO), so the
	// FUSE mount is released before PVC delete is attempted; otherwise the PVC
	// gets stuck Terminating because something has it mounted.
	//
	// Captured by reference via the closure; keepAlivePod may be set even when
	// runWithKeepAlivePod returns an error (pod was created but never became
	// Ready, etc.). We want to clean those up too.
	//
	// Skipped in hook mode (takeSnapshot=false) — the external postBackupHook
	// owns the pod's lifecycle, matching the PVC's own defer skip.
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

	// Base output map carries the staging PVC coordinates; any `kando output`
	// lines from the pod merge on top.
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

	// Hook-mode path: leave PVC + keep-alive pod alive for the external phase.
	// Both defers are no-ops here (cleanup=false, takeSnapshot=false).
	if !a.takeSnapshot {
		return out, nil
	}

	// Function-owned snapshot path: take the CSI snapshot now while the pod
	// (and therefore the mount) is still alive. Wait until the snapshot reaches
	// a terminal state (readyToUse=true OR Status.Error set OR ctx cancel —
	// all observed by the shared isReadyToUse predicate in
	// pkg/kube/snapshot/snapshot_stable.go). On return, the deferred keep-alive
	// pod kill fires (releases mount), then the deferred PVC cleanup fires.
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

// takeStagingSnapshot drives a CSI snapshot of the staging PVC by reusing the
// package-private createCSISnapshot helper from create_csi_snapshot.go.
//
// We do NOT reimplement the snapshot loop here — createCSISnapshot already:
//   - Builds the snapshotMeta
//   - Calls snapshotter.Create with waitForReady=true
//   - waitForReady triggers WaitOnReadyToUse which polls via
//     poll.WaitWithRetries, honouring ctx cancellation
//   - The isReadyToUse predicate returns success on ReadyToUse=true AND
//     returns the driver's error on Status.Error — both terminal states are
//     observed by the very first poll iteration thanks to immediate=true
//     equivalent inside poll.WaitWithRetries.
//
// On any non-nil error from createCSISnapshot — API failure, driver-reported
// snapshot error, or context cancel — this returns a wrapped loud error. The
// caller's deferred PVC cleanup still fires (cleanup honours retErr separately).
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

	// Deterministic-ish name: <pvc>-snap-<random6>. defaultSnapshotName lives
	// in create_csi_snapshot.go and uses the same convention.
	snapName := defaultSnapshotName(pvcName, 6)

	// Tag the snapshot so an operator can correlate it with the ActionSet that
	// produced it. Same intent as the labels we stamp on the staging PVC.
	snapLabels := map[string]string{
		LabelKeyOwnerAction:       a.actionSetTag,
		LabelKeyWorkloadNamespace: a.workloadNamespace,
	}
	if a.workloadName != "" {
		snapLabels[LabelKeyWorkloadName] = a.workloadName
	}
	if a.workloadNamespace == "" {
		// Avoid stamping an empty workload-namespace label (matches
		// stagingPVCLabels' guard for unit-test contexts).
		delete(snapLabels, LabelKeyWorkloadNamespace)
	}

	// Blocks here until snapshot reaches terminal state. PVC delete defer is
	// suspended for the duration of this call.
	vs, err := createCSISnapshot(ctx, snapshotter, snapName, a.namespace, pvcName, a.snapshotClass, true /* waitForReady */, snapLabels)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{
		OutputKeySnapshotName:      snapName,
		OutputKeySnapshotNamespace: a.namespace,
	}
	// restoreSize: prefer the snapshot's own RestoreSize when populated.
	// Streaming/FUSE CSI drivers (backup-csi-driver / kopia) often leave
	// vs.Status.RestoreSize nil because they don't have a block-level size to
	// report. Fall back to the PVC's actual provisioned capacity, then to the
	// PVC's requested size. We emit *something* unconditionally so blueprint
	// outputArtifacts templates referencing restoreSize never fail to render.
	var snapRestoreSize *resource.Quantity
	if vs.Status != nil {
		snapRestoreSize = vs.Status.RestoreSize
	}
	out[OutputKeySnapshotRestoreSize] = f.deriveRestoreSize(ctx, cli, a.namespace, pvcName, snapRestoreSize)
	if vs.Status != nil && vs.Status.BoundVolumeSnapshotContentName != nil {
		out[OutputKeySnapshotContent] = *vs.Status.BoundVolumeSnapshotContentName
	}
	return out, nil
}

// deriveRestoreSize returns a non-empty size string suitable for emitting
// as the snapshot's `restoreSize` output. Order of preference:
//  1. VolumeSnapshot.status.restoreSize (block-level CSI drivers populate this)
//  2. PVC.status.capacity[storage] (actual provisioned size; populated once Bound)
//  3. PVC.spec.resources.requests[storage] (what we asked for at creation)
//  4. defaultBackupPVCSize (last-resort default; matches our PVC create default)
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
		// Ensure no empty workload-namespace label leaks into the selector
		// when called outside a workload context (e.g. unit tests).
		delete(labels, LabelKeyWorkloadNamespace)
	}
	for k, v := range a.bpLabels {
		labels[k] = v
	}
	return labels
}

// runWithKeepAlivePod runs the user command in a pod that intentionally stays
// alive past the command's exit. The pod sleeps for keepPodAliveSeconds after
// emitting a marker; the function returns as soon as the marker appears in the
// stream, so the calling phase completes while the pod (and its mount) keep
// running.
//
// Returns (parsed output, pod name, error). The caller is responsible for the
// pod's eventual deletion:
//   - In function-owned snapshot mode (takeSnapshot=true), run() actively
//     deletes the pod after CreateSnapshot reaches a terminal state.
//   - In hook mode (takeSnapshot=false), the backupPosthook deletes the pod
//     via the keep-alive label selector after the external snapshot phase.
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

// wrapCommandForKeepAlive composes the user's command into a shell pipeline
// that, after the user command exits, prints a marker (with the original exit
// code) and sleeps for `seconds`, holding the mount. Only supports the
// `bash|sh -c <script>` form.
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

// waitForKeepAliveMarker scans pod log lines until the marker is seen, then
// returns the embedded exit code and everything emitted before the marker (so
// callers can run kando-output parsing on it).
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

func (f *kubeTaskWithBackupPVCFunc) buildPodOptions(a *kubeTaskWithBackupPVCArgs, pvcName string) (*kube.PodOptions, error) {
	annotations := a.bpAnnotations
	labels := a.bpLabels

	opts := &kube.PodOptions{
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
		Annotations:          annotations,
		Labels:               labels,
	}
	return opts, nil
}

func kubeTaskWithBackupPVCPodFunc() func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
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
			return nil, errkit.Wrap(err, "Backup pod did not complete successfully", "pod", pc.PodName())
		}
		return out, nil
	}
}
