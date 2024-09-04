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

package kube

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	json "github.com/json-iterator/go"
	"github.com/kanisterio/errkit"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sp "k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	// DefaultPodReadyWaitTimeout is the time to wait for pod to be ready
	DefaultPodReadyWaitTimeout = 15 * time.Minute
	// PodReadyWaitTimeoutEnv is the env var to get pod ready wait timeout
	PodReadyWaitTimeoutEnv = "KANISTER_POD_READY_WAIT_TIMEOUT"
	errAccessingNode       = "Failed to get node"
	DefaultContainerName   = "container"
	redactedValue          = "XXXXX"
)

type VolumeMountOptions struct {
	MountPath string
	ReadOnly  bool
}

// PodOptions specifies options for `CreatePod`
type PodOptions struct {
	Annotations        map[string]string
	Command            []string
	ContainerName      string
	Name               string
	GenerateName       string
	Image              string
	Labels             map[string]string
	Namespace          string
	ServiceAccountName string
	Volumes            map[string]VolumeMountOptions
	BlockVolumes       map[string]string
	// PodSecurityContext and ContainerSecurityContext can be used to set the security context
	// at the pod level and container level respectively.
	// You can still use podOverride to set the pod security context, but these fields will take precedence.
	// We chose these fields to specify security context instead of just using podOverride because
	// the merge behaviour of the pods spec is confusing in case of podOverride, and this is more readable.
	PodSecurityContext       *corev1.PodSecurityContext
	ContainerSecurityContext *corev1.SecurityContext
	PodOverride              crv1alpha1.JSONMap
	Resources                corev1.ResourceRequirements
	RestartPolicy            corev1.RestartPolicy
	OwnerReferences          []metav1.OwnerReference
	EnvironmentVariables     []corev1.EnvVar
	Lifecycle                *corev1.Lifecycle
}

func (po *PodOptions) AddLabels(labels map[string]string) {
	if po == nil {
		return
	}

	if po.Labels == nil {
		po.Labels = make(map[string]string)
	}

	for k, v := range labels {
		po.Labels[k] = v
	}
}

func (po *PodOptions) AddAnnotations(annotations map[string]string) {
	if po == nil {
		return
	}

	if po.Annotations == nil {
		po.Annotations = make(map[string]string)
	}

	for k, v := range annotations {
		po.Annotations[k] = v
	}
}

func GetPodObjectFromPodOptions(ctx context.Context, cli kubernetes.Interface, opts *PodOptions) (*corev1.Pod, error) {
	// If Namespace is not specified, use the controller Namespace.
	cns, err := GetControllerNamespace()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get controller namespace")
	}
	ns := opts.Namespace
	if ns == "" {
		ns = cns
	}

	// If a ServiceAccount is not specified and we are in the controller's
	// namespace, use the same service account as the controller.
	sa := opts.ServiceAccountName
	if sa == "" && ns == cns {
		sa, err = GetControllerServiceAccount(cli)
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to get Controller Service Account")
		}
	}

	if opts.RestartPolicy == "" {
		opts.RestartPolicy = corev1.RestartPolicyNever
	}

	volumeMounts, podVolumes, err := createFilesystemModeVolumeSpecs(ctx, opts.Volumes)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create volume spec")
	}
	volumeDevices, blockVolumes, err := createBlockModeVolumeSpecs(opts.BlockVolumes)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create raw block volume spec")
	}
	podVolumes = append(podVolumes, blockVolumes...)
	defaultSpecs := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            ContainerNameFromPodOptsOrDefault(opts),
				Image:           opts.Image,
				Command:         opts.Command,
				ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
				VolumeMounts:    volumeMounts,
				VolumeDevices:   volumeDevices,
				Resources:       opts.Resources,
			},
		},
		// RestartPolicy dictates when the containers of the pod should be
		// restarted.  The possible values include Always, OnFailure and Never
		// with Never being the default.  OnFailure policy will result in
		// failed containers being restarted with an exponential back-off delay.
		RestartPolicy:      opts.RestartPolicy,
		Volumes:            podVolumes,
		ServiceAccountName: sa,
	}

	if opts.EnvironmentVariables != nil && len(opts.EnvironmentVariables) > 0 {
		defaultSpecs.Containers[0].Env = opts.EnvironmentVariables
	}

	// Patch default Pod Specs if needed
	patchedSpecs, err := patchDefaultPodSpecs(defaultSpecs, opts.PodOverride)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create pod. Failed to override pod specs.", "namespace", opts.Namespace, "nameFmt", opts.GenerateName)
	}

	// Always put the main container the first
	sort.Slice(patchedSpecs.Containers, func(i, j int) bool {
		return patchedSpecs.Containers[i].Name == ContainerNameFromPodOptsOrDefault(opts)
	})

	return createPodSpec(opts, patchedSpecs, ns), nil
}

func createFilesystemModeVolumeSpecs(
	ctx context.Context,
	vols map[string]VolumeMountOptions,
) (volumeMounts []corev1.VolumeMount, podVolumes []corev1.Volume, error error) {
	// Build filesystem mode volume specs
	for pvcName, mountOpts := range vols {
		id, err := uuid.NewV1()
		if err != nil {
			return nil, nil, err
		}

		if mountOpts.ReadOnly {
			log.Debug().WithContext(ctx).Print("PVC will be mounted in read-only mode", field.M{"pvcName": pvcName})
		}

		podVolName := fmt.Sprintf("vol-%s", id.String())
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: podVolName, MountPath: mountOpts.MountPath, ReadOnly: mountOpts.ReadOnly})
		podVolumes = append(podVolumes,
			corev1.Volume{
				Name: podVolName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
						ReadOnly:  mountOpts.ReadOnly,
					},
				},
			},
		)
	}
	return volumeMounts, podVolumes, nil
}

func createBlockModeVolumeSpecs(blockVols map[string]string) (volumeDevices []corev1.VolumeDevice, podVolumes []corev1.Volume, error error) {
	// Build block mode volume specs
	for pvc, devicePath := range blockVols {
		id, err := uuid.NewV1()
		if err != nil {
			return nil, nil, err
		}
		podBlockVolName := fmt.Sprintf("block-%s", id.String())
		volumeDevices = append(volumeDevices, corev1.VolumeDevice{Name: podBlockVolName, DevicePath: devicePath})
		podVolumes = append(podVolumes,
			corev1.Volume{
				Name: podBlockVolName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc,
					},
				},
			},
		)
	}
	return volumeDevices, podVolumes, nil
}

func createPodSpec(opts *PodOptions, patchedSpecs corev1.PodSpec, ns string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: opts.GenerateName,
			Labels: map[string]string{
				consts.LabelKeyCreatedBy: consts.LabelValueKanister,
			},
		},
		Spec: patchedSpecs,
	}

	// Override `GenerateName` if `Name` option is provided
	if opts.Name != "" {
		pod.Name = opts.Name
	}

	// Add Annotations and Labels, if specified
	if opts.Annotations != nil {
		pod.ObjectMeta.Annotations = opts.Annotations
	}
	if pod.ObjectMeta.Labels == nil {
		pod.ObjectMeta.Labels = map[string]string{}
	}

	if opts.OwnerReferences != nil {
		pod.SetOwnerReferences(opts.OwnerReferences)
	}

	if opts.PodSecurityContext != nil {
		pod.Spec.SecurityContext = opts.PodSecurityContext
	}

	if opts.ContainerSecurityContext != nil {
		pod.Spec.Containers[0].SecurityContext = opts.ContainerSecurityContext
	}

	if opts.Lifecycle != nil {
		pod.Spec.Containers[0].Lifecycle = opts.Lifecycle
	}

	for key, value := range opts.Labels {
		pod.ObjectMeta.Labels[key] = value
	}

	pod.Namespace = ns
	return pod
}

// ContainerNameFromPodOptsOrDefault returns the container name if it's set in
// the passed `podOptions` value. If not, it's returns the default container
// name. This should be used whenever we create pods for Kanister functions.
func ContainerNameFromPodOptsOrDefault(po *PodOptions) string {
	if po == nil || po.ContainerName == "" {
		return DefaultContainerName
	}

	return po.ContainerName
}

// CreatePod creates a pod with a single container based on the specified image
func CreatePod(ctx context.Context, cli kubernetes.Interface, opts *PodOptions) (*corev1.Pod, error) {
	pod, err := GetPodObjectFromPodOptions(ctx, cli, opts)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get pod from podOptions", "namespace", opts.Namespace, "nameFmt", opts.GenerateName)
	}

	log.Debug().WithContext(ctx).Print("Creating POD", field.M{"name": pod.Name, "namespace": pod.Namespace})

	pod, err = cli.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		log.Error().WithContext(ctx).WithError(err).Print("Failed to create pod.", field.M{"pod": getRedactedPod(pod), "options": getRedactedOptions(opts)})
		return nil, errkit.Wrap(err, "Failed to create pod", "namespace", opts.Namespace, "nameFmt", opts.GenerateName)
	}
	return pod, nil
}

// DeletePod deletes the specified pod
func DeletePod(ctx context.Context, cli kubernetes.Interface, pod *corev1.Pod) error {
	if err := cli.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
		log.WithError(err).Print("DeletePod failed")
	}
	return nil
}

func StreamPodLogs(ctx context.Context, cli kubernetes.Interface, namespace, podName, containerName string) (io.ReadCloser, error) {
	plo := &corev1.PodLogOptions{
		Follow:    true,
		Container: containerName,
	}
	return cli.CoreV1().Pods(namespace).GetLogs(podName, plo).Stream(ctx)
}

// GetPodLogs fetches the logs from the given pod
func GetPodLogs(ctx context.Context, cli kubernetes.Interface, namespace, podName, containerName string) (string, error) {
	reader, err := cli.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Container: containerName}).Stream(ctx)
	if err != nil {
		return "", err
	}
	defer reader.Close() //nolint:errcheck
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// getErrorFromLogs fetches logs from pod and constructs error containing last ten lines of log and specified error message
func getErrorFromLogs(ctx context.Context, cli kubernetes.Interface, namespace, podName, containerName string, err error, errorMessage string) error {
	r, logErr := StreamPodLogs(ctx, cli, namespace, podName, containerName)
	if logErr != nil {
		return errkit.Wrap(logErr, "Failed to fetch logs from the pod")
	}
	defer r.Close() //nolint:errcheck

	// Grab last log lines and put them to an error
	lt := NewLogTail(logTailDefaultLength)
	// We are not interested in log extraction error
	io.Copy(lt, r) //nolint:errcheck

	return errkit.Wrap(errkit.Wrap(err, lt.ToString()), errorMessage)
}

// WaitForPodReady waits for a pod to exit the pending state
func WaitForPodReady(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, GetPodReadyWaitTimeout())
	defer waitCancel()
	attachLog := true
	containerForLogs := ""
	err := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		p, err := cli.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			attachLog = false
			return false, err
		}
		containerForLogs = p.Spec.Containers[0].Name

		// check if nodes are up and available
		err = checkNodesStatus(p, cli)
		if err != nil && !strings.Contains(err.Error(), errAccessingNode) {
			attachLog = false
			return false, err
		}

		// check for memory or resource issues
		if p.Status.Phase == corev1.PodPending {
			if p.Status.Reason == "OutOfmemory" || p.Status.Reason == "OutOfcpu" {
				attachLog = false
				return false, errkit.New("Pod stuck in pending state", "reason", p.Status.Reason)
			}
		}

		// check if pvc and pv are up and ready to mount
		if err := getVolStatus(timeoutCtx, p, cli, namespace); err != nil {
			attachLog = false
			return false, err
		}

		return p.Status.Phase != corev1.PodPending && p.Status.Phase != "", nil
	})

	if err == nil {
		return nil
	}

	errorMessage := fmt.Sprintf("Pod did not transition into running state. Timeout:%v  Namespace:%s, Name:%s", GetPodReadyWaitTimeout(), namespace, name)
	if attachLog {
		return getErrorFromLogs(ctx, cli, namespace, name, containerForLogs, err, errorMessage)
	}

	return errkit.Wrap(err, errorMessage)
}

func checkNodesStatus(p *corev1.Pod, cli kubernetes.Interface) error {
	n := strings.Split(p.Spec.NodeName, "/")
	if n[0] != "" {
		node, err := cli.CoreV1().Nodes().Get(context.TODO(), n[0], metav1.GetOptions{})
		if err != nil {
			return errkit.Wrap(err, errAccessingNode, "node", n[0])
		}
		if !IsNodeReady(node) || !IsNodeSchedulable(node) {
			return errkit.New("Node is currently not ready/schedulable", "node", n[0])
		}
	}
	return nil
}

// checkPVCAndPVStatus does the following:
//   - if PVC is present then check the status of PVC
//   - if PVC is pending then check if the PV status is VolumeFailed return error if so. if not then wait for timeout.
//   - if PVC not present then wait for timeout
func getVolStatus(ctx context.Context, p *corev1.Pod, cli kubernetes.Interface, namespace string) error {
	for _, vol := range p.Spec.Volumes {
		if err := checkPVCAndPVStatus(ctx, vol, p, cli, namespace); err != nil {
			return err
		}
	}
	return nil
}

// checkPVCAndPVStatus does the following:
//   - if PVC is present then check the status of PVC
//   - if PVC is pending then check if the PV status is VolumeFailed return error if so. if not then wait for timeout.
//   - if PVC not present then wait for timeout
func checkPVCAndPVStatus(ctx context.Context, vol corev1.Volume, p *corev1.Pod, cli kubernetes.Interface, namespace string) error {
	if vol.VolumeSource.PersistentVolumeClaim == nil {
		// wait for timeout
		return nil
	}
	pvcName := vol.VolumeSource.PersistentVolumeClaim.ClaimName
	pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(errors.Cause(err)) {
			// Do not return err, wait for timeout, since sometimes in case of statefulsets, they trigger creation of a volume
			return nil
		}

		return errkit.Wrap(err, "Failed to get PVC", "pvcName", pvcName)
	}

	switch pvc.Status.Phase {
	case corev1.ClaimLost:
		return errkit.New("PVC associated with pod has unexpected status", "pvcName", pvcName, "podName", p.Name, "status", corev1.ClaimLost)
	case corev1.ClaimPending:
		pvName := pvc.Spec.VolumeName
		if pvName == "" {
			// wait for timeout
			return nil
		}
		pv, err := cli.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(errors.Cause(err)) {
				// wait for timeout
				return nil
			}

			return errkit.Wrap(err, "Failed to get PV", "pvName", pvName)
		}
		if pv.Status.Phase == corev1.VolumeFailed {
			return errkit.New("PV associated with PVC has unexpected status",
				"pvName", pvName, "pvcName", pvcName, "status", corev1.VolumeFailed,
				"message", pv.Status.Message, "reason", pv.Status.Reason, "namespace", namespace)
		}
	}

	return nil
}

// WaitForPodCompletion waits for a pod to reach a terminal state, or timeout
func WaitForPodCompletion(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	attachLog := true
	containerForLogs := ""
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		p, err := cli.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			attachLog = false
			return true, err
		}
		containerForLogs = p.Spec.Containers[0].Name
		if p.Status.Phase == corev1.PodFailed {
			return false, errkit.New("Pod failed", "podName", name, "status", p.Status.String())
		}
		return p.Status.Phase == corev1.PodSucceeded, nil
	})

	if err == nil {
		return err
	}

	errorMessage := "Pod failed or did not transition into complete state"
	if attachLog {
		return getErrorFromLogs(ctx, cli, namespace, name, containerForLogs, err, errorMessage)
	}
	return errkit.Wrap(err, errorMessage)
}

// use Strategic Merge to patch default pod specs with the passed specs
func patchDefaultPodSpecs(defaultPodSpecs corev1.PodSpec, override crv1alpha1.JSONMap) (corev1.PodSpec, error) {
	// Merge default specs and override specs with StrategicMergePatch
	mergedPatch, err := strategicMergeJSONPatch(defaultPodSpecs, override)
	if err != nil {
		return corev1.PodSpec{}, err
	}

	// Convert merged json to corev1.PodSPec object
	podSpec := corev1.PodSpec{}
	err = json.Unmarshal(mergedPatch, &podSpec)
	if err != nil {
		return podSpec, err
	}
	return podSpec, err
}

// CreateAndMergeJSONPatch uses Strategic Merge to merge two Pod spec configuration
func CreateAndMergeJSONPatch(original, override crv1alpha1.JSONMap) (crv1alpha1.JSONMap, error) {
	// Merge json specs with StrategicMerge
	mergedPatch, err := strategicMergeJSONPatch(original, override)
	if err != nil {
		return nil, err
	}

	// Convert merged json to map[string]interface{}
	var merged map[string]interface{}
	err = json.Unmarshal(mergedPatch, &merged)
	if err != nil {
		return nil, err
	}
	return merged, err
}

func strategicMergeJSONPatch(original, override interface{}) ([]byte, error) {
	// Convert override specs to json
	overrideJSON, err := json.Marshal(override)
	if err != nil {
		return nil, err
	}

	// Convert original specs to json
	originalJSON, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}

	// Merge json specs with StrategicMerge
	mergedPatch, err := sp.StrategicMergePatch(originalJSON, overrideJSON, corev1.PodSpec{})
	if err != nil {
		return nil, err
	}
	return mergedPatch, nil
}

// GetPodReadyWaitTimeout returns the pod ready wait timeout from ENV if configured
// returns the default of 15 minutes otherwise
func GetPodReadyWaitTimeout() time.Duration {
	if v, ok := os.LookupEnv(PodReadyWaitTimeoutEnv); ok {
		iv, err := strconv.Atoi(v)
		if err == nil {
			return time.Duration(iv) * time.Minute
		}
		log.Debug().Print("Using default timeout value because of invalid environment variable", field.M{"envVar": v})
	}

	return DefaultPodReadyWaitTimeout
}

// getRedactedEnvVariables returns array of variables with removed values
// This function should be used every time when env variables are logged
func getRedactedEnvVariables(env []corev1.EnvVar) []corev1.EnvVar {
	if len(env) == 0 {
		return nil
	}

	result := make([]corev1.EnvVar, len(env))
	for i, ev := range env {
		result[i] = corev1.EnvVar{
			Name:  ev.Name,
			Value: redactedValue,
		}
	}

	return result
}

func getRedactedContainers(containers []corev1.Container) []corev1.Container {
	if len(containers) == 0 {
		return nil
	}

	result := make([]corev1.Container, len(containers))
	for i, c := range containers {
		result[i] = c
		result[i].Env = getRedactedEnvVariables(c.Env)
		result[i].Command = getRedactedStringSlice(c.Command)
		result[i].Args = getRedactedStringSlice(c.Args)
	}
	return result
}

func getRedactedStringSlice(slice []string) []string {
	if len(slice) == 0 {
		return nil
	}
	result := make([]string, len(slice))
	for j := range slice {
		result[j] = redactedValue
	}
	return result
}

func getRedactedPodOverride(podOverride crv1alpha1.JSONMap) crv1alpha1.JSONMap {
	if len(podOverride) == 0 {
		return nil
	}

	result := make(crv1alpha1.JSONMap, len(podOverride))
	for k, v := range podOverride {
		if c, ok := v.([]corev1.Container); ok {
			result[k] = getRedactedContainers(c)
		} else {
			result[k] = v
		}
	}

	return result
}

// getRedactedPod hides all sensitive information from pod object (env variables, commands)
// Should be used when pod structure is logged
func getRedactedPod(pod *corev1.Pod) *corev1.Pod {
	if pod == nil {
		return nil
	}

	result := *pod // Make shallow copy

	result.Spec.Containers = getRedactedContainers(result.Spec.Containers)
	result.Spec.InitContainers = getRedactedContainers(result.Spec.InitContainers)

	return &result
}

// getRedactedOptions hides all values of env variables from pod options, so that they should be safely logged
func getRedactedOptions(opts *PodOptions) *PodOptions {
	if opts == nil {
		return nil
	}

	result := *opts // Make shallow copy

	result.EnvironmentVariables = getRedactedEnvVariables(result.EnvironmentVariables)
	result.Command = getRedactedStringSlice(result.Command)
	result.PodOverride = getRedactedPodOverride(result.PodOverride)
	return &result
}
