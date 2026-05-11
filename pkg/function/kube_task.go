// Copyright 2019 The Kanister Authors.
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
	"time"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	jobPrefix = "kanister-job-"

	// KubeTaskFuncName gives the function name
	KubeTaskFuncName           = "KubeTask"
	KubeTaskNamespaceArg       = "namespace"
	KubeTaskImageArg           = "image"
	KubeTaskCommandArg         = "command"
	KubeTaskVolumesArg         = "volumes"
	KubeTaskSnapshotVolumesArg = "snapshotVolumes"

	// Defaults used when KubeTask auto-creates a restore PVC from a snapshot
	// artifact in ArtifactsIn without an explicit snapshotVolumes entry.
	kubeTaskDefaultRestoreMountPath    = "/restore"
	kubeTaskDefaultRestoreStorageClass = "kopia-restore"
	kubeTaskDefaultRestoreSize         = "5Gi"
)

func init() {
	_ = kanister.Register(&kubeTaskFunc{})
}

var _ kanister.Func = (*kubeTaskFunc)(nil)

type kubeTaskFunc struct {
	progressPercent string
}

func (*kubeTaskFunc) Name() string {
	return KubeTaskFuncName
}

// ephemeralSnapshotDiscovery instructs kubeTaskPodFunc to look at the live
// pod's spec.volumes for an ephemeral CSI backup volume and, if present, poll
// for the VolumeSnapshot the CSI driver auto-creates on NodeUnpublishVolume.
// The pod's resolved spec is the source of truth — it survives strategic
// merge of blueprint podOverride with the ActionSet's podOverride.
type ephemeralSnapshotDiscovery struct {
	cli       kubernetes.Interface
	namespace string
	after     time.Time
}

// podHasEphemeralBackupVolume returns true if any of the pod's spec.volumes
// is an inline CSI volume signed as the backup-csi-driver's ephemeral backup
// volume (csi.storage.k8s.io/ephemeral=="true" AND mode=="backup").
func podHasEphemeralBackupVolume(pod *corev1.Pod) bool {
	for _, v := range pod.Spec.Volumes {
		if v.CSI == nil {
			continue
		}
		if v.CSI.VolumeAttributes["csi.storage.k8s.io/ephemeral"] != "true" {
			continue
		}
		if v.CSI.VolumeAttributes["mode"] != "backup" {
			continue
		}
		return true
	}
	return false
}

func kubeTask(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	image string,
	command []string,
	vols map[string]string,
	podOverride crv1alpha1.JSONMap,
	annotations,
	labels map[string]string,
	disc *ephemeralSnapshotDiscovery,
) (map[string]interface{}, error) {
	// Validate and build volume mount options.
	validatedVols := make(map[string]kube.VolumeMountOptions)
	for pvcName, mountPath := range vols {
		pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to retrieve PVC", "namespace", namespace, "name", pvcName)
		}
		validatedVols[pvcName] = kube.VolumeMountOptions{
			MountPath: mountPath,
			ReadOnly:  kube.PVCContainsReadOnlyAccessMode(pvc),
		}
	}

	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      command,
		Volumes:      validatedVols,
		PodOverride:  podOverride,
		Annotations:  annotations,
		Labels:       labels,
	}

	// Apply the registered ephemeral pod changes.
	if err := ephemeral.PodOptions.Apply(options); err != nil {
		return nil, errkit.Wrap(err, "Failed to apply ephemeral pod options")
	}

	// Mark pod with label having key `kanister.io/JobID`, the value of which is a reference to the origin of the pod.
	kube.AddLabelsToPodOptionsFromContext(ctx, options, path.Join(consts.LabelPrefix, consts.LabelSuffixJobID))
	pr := kube.NewPodRunner(cli, options)
	podFunc := kubeTaskPodFunc(disc)
	return pr.Run(ctx, podFunc)
}

// resolveRestoreSize picks the size to request for the restore PVC.
// Order of precedence:
//  1. Explicit "size" in the snapshotVolumes spec (if non-empty).
//  2. The VolumeSnapshot's status.restoreSize, if reported by the CSI driver.
//  3. Built-in fallback (kubeTaskDefaultRestoreSize).
func resolveRestoreSize(ctx context.Context, dynCli dynamic.Interface, namespace, snapName, explicit string) (resource.Quantity, error) {
	if explicit != "" {
		q, err := resource.ParseQuantity(explicit)
		if err != nil {
			return resource.Quantity{}, fmt.Errorf("invalid size %q: %w", explicit, err)
		}
		return q, nil
	}
	snap, err := dynCli.Resource(snapshot.VolSnapGVR).Namespace(namespace).Get(ctx, snapName, metav1.GetOptions{})
	if err == nil {
		if restoreSize, found, _ := unstructured.NestedString(snap.Object, "status", "restoreSize"); found && restoreSize != "" {
			if q, perr := resource.ParseQuantity(restoreSize); perr == nil {
				return q, nil
			}
		}
	}
	return resource.MustParse(kubeTaskDefaultRestoreSize), nil
}

func kubeTaskPodFunc(disc *ephemeralSnapshotDiscovery) func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errkit.Wrap(err, "Failed while waiting for Pod to be ready", "pod", pc.PodName())
		}

		// Inspect the live pod once it is scheduled so we know whether to
		// poll for an ephemeral CSI snapshot after it exits. Reading the
		// actual pod is robust against blueprint+ActionSet podOverride
		// merging that would obscure the JSONMap form.
		var pollSnapshotAfterExit bool
		if disc != nil {
			pod, gErr := disc.cli.CoreV1().Pods(disc.namespace).Get(ctx, pc.PodName(), metav1.GetOptions{})
			if gErr == nil && podHasEphemeralBackupVolume(pod) {
				pollSnapshotAfterExit = true
			}
		}

		ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
		// Fetch logs from the pod
		r, err := pc.StreamPodLogs(ctx)
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to fetch logs from the pod")
		}
		out, err := output.LogAndParse(ctx, r)
		if err != nil {
			return nil, err
		}
		// Wait for pod completion
		if err := pc.WaitForPodCompletion(ctx); err != nil {
			return nil, errkit.Wrap(err, "Failed while waiting for Pod to complete", "pod", pc.PodName())
		}
		// If this KubeTask used an ephemeral CSI backup volume, the CSI driver
		// auto-creates a VolumeSnapshot in NodeUnpublishVolume after the pod
		// exits. Poll for it and surface name/namespace as phase outputs so
		// the action's outputArtifacts can reference them without a separate
		// WaitForEphemeralSnapshot phase.
		if pollSnapshotAfterExit {
			snapName, sErr := findEphemeralVolumeSnapshot(ctx, disc.namespace, disc.after, pc.PodName())
			if sErr == nil {
				if out == nil {
					out = make(map[string]interface{})
				}
				out[WaitForEphemeralSnapshotNameOutput] = snapName
				out[WaitForEphemeralSnapshotNamespaceOutput] = disc.namespace
			}
		}
		return out, err
	}
}

func (ktf *kubeTaskFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	ktf.progressPercent = progress.StartedPercent
	defer func() { ktf.progressPercent = progress.CompletedPercent }()

	var namespace, image string
	var command []string
	var vols map[string]string
	var err error
	var bpAnnotations, bpLabels map[string]string
	if err = Arg(args, KubeTaskImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeTaskCommandArg, &command); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskNamespaceArg, &namespace, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskVolumesArg, &vols, nil); err != nil {
		return nil, err
	}
	var snapVols map[string]map[string]string
	if err = OptArg(args, KubeTaskSnapshotVolumesArg, &snapVols, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}

	podOverride, err := GetPodSpecOverride(tp, args, PodOverrideArg)
	if err != nil {
		return nil, err
	}

	annotations := bpAnnotations
	labels := bpLabels
	if tp.PodAnnotations != nil {
		// merge the actionset annotations with blueprint annotations
		var actionSetAnn ActionSetAnnotations = tp.PodAnnotations
		annotations = actionSetAnn.MergeBPAnnotations(bpAnnotations)
	}

	if tp.PodLabels != nil {
		// merge the actionset labels with blueprint labels
		var actionSetLabels ActionSetLabels = tp.PodLabels
		labels = actionSetLabels.MergeBPLabels(bpLabels)
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}
	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create dynamic Kubernetes client")
	}

	// Auto-detect: if no explicit snapshotVolumes was provided but ArtifactsIn
	// contains a snapshot artifact (KeyValue.volumeSnapshotName), pick that one
	// snapshot and let KubeTask transparently create a restore PVC for it.
	// Only one snapshot is auto-mounted: the user explicitly selects the restore
	// point via the artifact passed into the action, so there is exactly one.
	if len(snapVols) == 0 && len(tp.ArtifactsIn) > 0 {
		for _, art := range tp.ArtifactsIn {
			if art.KeyValue == nil {
				continue
			}
			snapName := art.KeyValue["volumeSnapshotName"]
			if snapName == "" {
				continue
			}
			snapVols = map[string]map[string]string{
				snapName: {
					"namespace": art.KeyValue["volumeSnapshotNamespace"],
				},
			}
			break
		}
	}

	if len(snapVols) > 0 {
		if vols == nil {
			vols = make(map[string]string)
		}
		createdPVCs := make([]string, 0, len(snapVols))
		defer func() {
			for _, pvcName := range createdPVCs {
				err := cli.CoreV1().PersistentVolumeClaims(namespace).Delete(
					context.Background(), pvcName, metav1.DeleteOptions{})
				if err != nil && !apierrors.IsNotFound(err) {
					_ = err // best-effort cleanup; pod already finished
				}
			}
		}()
		for snapName, spec := range snapVols {
			mountPath := spec["mountPath"]
			if mountPath == "" {
				mountPath = kubeTaskDefaultRestoreMountPath
			}
			storageClass := spec["storageClass"]
			if storageClass == "" {
				storageClass = kubeTaskDefaultRestoreStorageClass
			}
			snapNS := spec["namespace"]
			if snapNS == "" {
				snapNS = namespace
			}

			size, err := resolveRestoreSize(ctx, dynCli, snapNS, snapName, spec["size"])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve restore size for snapshot %q: %w", snapName, err)
			}

			pvcName := "restore-" + rand.String(5)
			snapshotAPIGroup := SnapshotAPIGroup
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: namespace,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					StorageClassName: &storageClass,
					DataSource: &corev1.TypedLocalObjectReference{
						APIGroup: &snapshotAPIGroup,
						Kind:     "VolumeSnapshot",
						Name:     snapName,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: size,
						},
					},
				},
			}
			if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
				return nil, fmt.Errorf("failed to create restore PVC for snapshot %q: %w", snapName, err)
			}
			createdPVCs = append(createdPVCs, pvcName)
			if err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
				p, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				return p.Status.Phase == corev1.ClaimBound, nil
			}); err != nil {
				return nil, fmt.Errorf("restore PVC %q for snapshot %q did not become Bound: %w", pvcName, snapName, err)
			}
			vols[pvcName] = mountPath
		}
	}

	// Always set up discovery; kubeTaskPodFunc inspects the live pod's spec
	// to decide whether to actually poll for an ephemeral snapshot. Capture
	// "now" before the pod is created so the poll only considers snapshots
	// created by this run, not earlier backups.
	disc := &ephemeralSnapshotDiscovery{
		cli:       cli,
		namespace: namespace,
		after:     time.Now(),
	}

	return kubeTask(
		ctx,
		cli,
		namespace,
		image,
		command,
		vols,
		podOverride,
		annotations,
		labels,
		disc,
	)
}

func (*kubeTaskFunc) RequiredArgs() []string {
	return []string{
		KubeTaskImageArg,
		KubeTaskCommandArg,
	}
}

func (*kubeTaskFunc) Arguments() []string {
	return []string{
		KubeTaskImageArg,
		KubeTaskCommandArg,
		KubeTaskNamespaceArg,
		KubeTaskVolumesArg,
		KubeTaskSnapshotVolumesArg,
		PodOverrideArg,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (ktf *kubeTaskFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(ktf.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(ktf.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(ktf.RequiredArgs(), args)
}

func (ktf *kubeTaskFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    ktf.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
