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
	KubeTaskWithBackupPVCEnvFromSecretArg    = "envFromSecret"
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
	KubeTaskWithBackupPVCKeepPodAliveForSnapshotArg = "keepPodAliveForSnapshot"

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
	envFromSecret    string
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
	bpAnnotations       map[string]string
	bpLabels            map[string]string

	workloadName      string
	workloadNamespace string
	actionSetUID      string
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
		KubeTaskWithBackupPVCEnvFromSecretArg,
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
	return utils.CheckRequiredArgs(f.RequiredArgs(), args)
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
	if err = OptArg(args, KubeTaskWithBackupPVCEnvFromSecretArg, &parsed.envFromSecret, ""); err != nil {
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

	parsed.actionSetUID = actionSetUIDFromContext(ctx)
	if parsed.actionSetUID == "" {
		return nil, errkit.New("Unable to read ActionSet UID from context; required to stamp the owner-action label")
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

	// On exit, clean up the staging PVC if the run failed. On success the PVC is
	// intentionally left alive for Kasten's snapshot phase to discover and
	// snapshot via the labels stamped below.
	defer func() {
		if retErr == nil {
			return
		}
		if a.keepPVCOnFailure {
			log.WithContext(ctx).Print("Leaving staging PVC alive for debugging (keepPVCOnFailure=true)",
				field.M{"namespace": a.namespace, "pvcName": pvc.Name})
			return
		}
		if delErr := pvcGracefulDelete(ctx, cli, a.namespace, pvc.Name); delErr != nil {
			log.WithError(delErr).WithContext(ctx).Print("Failed to delete staging PVC after backup failure",
				field.M{"namespace": a.namespace, "pvcName": pvc.Name})
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
	// the volume mount past command exit while Kasten's CSI snapshot fires.
	// Function returns when it sees the marker; posthook deletes the pod.
	var podOut map[string]interface{}
	if a.keepPodAliveSeconds > 0 {
		podOut, err = f.runWithKeepAlivePod(ctx, cli, podOpts, a)
	} else {
		pr := kube.NewPodRunner(cli, podOpts)
		podOut, err = pr.Run(ctx, kubeTaskWithBackupPVCPodFunc())
	}
	if err != nil {
		return nil, errkit.Wrap(err, "Backup command failed",
			"namespace", a.namespace, "pvcName", pvc.Name)
	}

	// Success: leave PVC alive. Surface any `kando output` lines on top of
	// our staging-PVC coordinates so downstream phases can reference both.
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
	return out, nil
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
		LabelKeyOwnerAction:       a.actionSetUID,
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
// running. The backupPosthook is responsible for deleting the pod via the
// keep-alive label selector.
func (f *kubeTaskWithBackupPVCFunc) runWithKeepAlivePod(
	ctx context.Context,
	cli kubernetes.Interface,
	podOpts *kube.PodOptions,
	a *kubeTaskWithBackupPVCArgs,
) (map[string]interface{}, error) {
	wrapped, err := wrapCommandForKeepAlive(podOpts.Command, a.keepPodAliveSeconds)
	if err != nil {
		return nil, err
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
		return nil, errkit.Wrap(err, "Failed to create keep-alive pod", "namespace", a.namespace)
	}
	pod := pc.Pod()
	log.WithContext(ctx).Print("Created keep-alive backup pod",
		field.M{"pod": pod.Name, "namespace": pod.Namespace, "keepAliveSeconds": a.keepPodAliveSeconds})

	if err := pc.WaitForPodReady(ctx); err != nil {
		return nil, errkit.Wrap(err, "Keep-alive pod did not become ready", "pod", pc.PodName())
	}

	streamCtx := field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
	r, err := pc.StreamPodLogs(streamCtx)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to stream logs from keep-alive pod", "pod", pc.PodName())
	}
	defer r.Close() //nolint:errcheck

	exitCode, captured, err := waitForKeepAliveMarker(streamCtx, r)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed waiting for command-done marker", "pod", pc.PodName())
	}
	if exitCode != 0 {
		return nil, errkit.New("Backup command exited non-zero inside keep-alive pod",
			"pod", pc.PodName(), "exitCode", exitCode)
	}
	parsedOut, err := output.LogAndParse(streamCtx, io.NopCloser(strings.NewReader(captured)))
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to parse output from keep-alive pod", "pod", pc.PodName())
	}
	// Do NOT stop the pod; the posthook deletes it via the keep-alive label.
	return parsedOut, nil
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
