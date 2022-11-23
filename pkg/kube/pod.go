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
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	json "github.com/json-iterator/go"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
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
	defaultContainerName   = "container"
)

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
	Volumes            map[string]string
	BlockVolumes       map[string]string
	// PodSecurityContext and ContainerSecurityContext can be used to set the security context
	// at the pod level and container level respectively.
	// You can still use podOverride to set the pod security context, but these fields will take precedence.
	// We chose these fields to specify security context instead of just using podOverride because
	// the merge behaviour of the pods spec is confusing in case of podOverride, and this is more readable.
	PodSecurityContext       *v1.PodSecurityContext
	ContainerSecurityContext *v1.SecurityContext
	PodOverride              crv1alpha1.JSONMap
	Resources                v1.ResourceRequirements
	RestartPolicy            v1.RestartPolicy
	OwnerReferences          []metav1.OwnerReference
	EnvironmentVariables     []v1.EnvVar
}

func GetPodObjectFromPodOptions(cli kubernetes.Interface, opts *PodOptions) (*v1.Pod, error) {
	// If Namespace is not specified, use the controller Namespace.
	cns, err := GetControllerNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get controller namespace")
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
			return nil, errors.Wrap(err, "Failed to get Controller Service Account")
		}
	}

	if opts.RestartPolicy == "" {
		opts.RestartPolicy = v1.RestartPolicyNever
	}

	volumeMounts, podVolumes, err := createFilesystemModeVolumeSpecs(opts.Volumes)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create volume spec")
	}
	volumeDevices, blockVolumes, err := createBlockModeVolumeSpecs(opts.BlockVolumes)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create raw block volume spec")
	}
	podVolumes = append(podVolumes, blockVolumes...)
	defaultSpecs := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:            defaultContainerName,
				Image:           opts.Image,
				Command:         opts.Command,
				ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
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
		return nil, errors.Wrapf(err, "Failed to create pod. Failed to override pod specs. Namespace: %s, NameFmt: %s", opts.Namespace, opts.GenerateName)
	}

	pod := &v1.Pod{
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

	// Override default container name if applicable
	if opts.ContainerName != "" {
		pod.Spec.Containers[0].Name = opts.ContainerName
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

	for key, value := range opts.Labels {
		pod.ObjectMeta.Labels[key] = value
	}

	pod.Namespace = ns

	return pod, nil
}

// CreatePod creates a pod with a single container based on the specified image
func CreatePod(ctx context.Context, cli kubernetes.Interface, opts *PodOptions) (*v1.Pod, error) {
	pod, err := GetPodObjectFromPodOptions(cli, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get pod from podOptions. Namespace: %s, NameFmt: %s", opts.Namespace, opts.GenerateName)
	}

	pod, err = cli.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod. Namespace: %s, NameFmt: %s", pod.Namespace, opts.GenerateName)
	}
	return pod, nil
}

// DeletePod deletes the specified pod
func DeletePod(ctx context.Context, cli kubernetes.Interface, pod *v1.Pod) error {
	if err := cli.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
		log.WithError(err).Print("DeletePod failed")
	}
	return nil
}

func StreamPodLogs(ctx context.Context, cli kubernetes.Interface, namespace, name string) (io.ReadCloser, error) {
	plo := &v1.PodLogOptions{
		Follow: true,
	}
	return cli.CoreV1().Pods(namespace).GetLogs(name, plo).Stream(ctx)
}

// GetPodLogs fetches the logs from the given pod
func GetPodLogs(ctx context.Context, cli kubernetes.Interface, namespace, name string) (string, error) {
	reader, err := cli.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{}).Stream(ctx)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// WaitForPodReady waits for a pod to exit the pending state
func WaitForPodReady(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, GetPodReadyWaitTimeout())
	defer waitCancel()
	err := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		p, err := cli.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// check if nodes are up and available
		err = checkNodesStatus(p, cli)
		if err != nil && !strings.Contains(err.Error(), errAccessingNode) {
			return false, err
		}

		// check for memory or resource issues
		if p.Status.Phase == v1.PodPending {
			if p.Status.Reason == "OutOfmemory" || p.Status.Reason == "OutOfcpu" {
				return false, errors.Errorf("Pod stuck in pending state, reason: %s", p.Status.Reason)
			}
		}

		// check if pvc and pv are up and ready to mount
		if err := getVolStatus(timeoutCtx, p, cli, namespace); err != nil {
			return false, err
		}

		return p.Status.Phase != v1.PodPending && p.Status.Phase != "", nil
	})
	return errors.Wrapf(err, "Pod did not transition into running state. Timeout:%v  Namespace:%s, Name:%s", GetPodReadyWaitTimeout(), namespace, name)
}

func checkNodesStatus(p *v1.Pod, cli kubernetes.Interface) error {
	n := strings.Split(p.Spec.NodeName, "/")
	if n[0] != "" {
		node, err := cli.CoreV1().Nodes().Get(context.TODO(), n[0], metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "%s %s", errAccessingNode, n[0])
		}
		if !IsNodeReady(node) || !IsNodeSchedulable(node) {
			return errors.Errorf("Node %s is currently not ready/schedulable", n[0])
		}
	}
	return nil
}

// checkPVCAndPVStatus does the following:
//   - if PVC is present then check the status of PVC
//   - if PVC is pending then check if the PV status is VolumeFailed return error if so. if not then wait for timeout.
//   - if PVC not present then wait for timeout
func getVolStatus(ctx context.Context, p *v1.Pod, cli kubernetes.Interface, namespace string) error {
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
func checkPVCAndPVStatus(ctx context.Context, vol v1.Volume, p *v1.Pod, cli kubernetes.Interface, namespace string) error {
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
		} else {
			return errors.Wrapf(err, "Failed to get PVC %s", pvcName)
		}
	}

	switch pvc.Status.Phase {
	case v1.ClaimLost:
		return errors.Errorf("PVC %s assoicated with pod %s has status: %s", pvcName, p.Name, v1.ClaimLost)
	case v1.ClaimPending:
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
			} else {
				return errors.Wrapf(err, "Failed to get PV %s", pvName)
			}
		}
		if pv.Status.Phase == v1.VolumeFailed {
			return errors.Errorf("PV %s associated with PVC %s has status: %s message: %s reason: %s namespace: %s", pvName, pvcName, v1.VolumeFailed, pv.Status.Message, pv.Status.Reason, namespace)
		}
	}

	return nil
}

// WaitForPodCompletion waits for a pod to reach a terminal state, or timeout
func WaitForPodCompletion(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		p, err := cli.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		switch p.Status.Phase {
		case v1.PodFailed:
			return false, errors.Errorf("Pod %s failed. Pod status: %s", name, p.Status.String())
		}
		return p.Status.Phase == v1.PodSucceeded, nil
	})
	return errors.Wrap(err, "Pod failed or did not transition into complete state")
}

// use Strategic Merge to patch default pod specs with the passed specs
func patchDefaultPodSpecs(defaultPodSpecs v1.PodSpec, override crv1alpha1.JSONMap) (v1.PodSpec, error) {
	// Merge default specs and override specs with StrategicMergePatch
	mergedPatch, err := strategicMergeJsonPatch(defaultPodSpecs, override)
	if err != nil {
		return v1.PodSpec{}, err
	}

	// Convert merged json to v1.PodSPec object
	podSpec := v1.PodSpec{}
	err = json.Unmarshal(mergedPatch, &podSpec)
	if err != nil {
		return podSpec, err
	}
	return podSpec, err
}

// CreateAndMergeJsonPatch uses Strategic Merge to merge two Pod spec configuration
func CreateAndMergeJsonPatch(original, override crv1alpha1.JSONMap) (crv1alpha1.JSONMap, error) {
	// Merge json specs with StrategicMerge
	mergedPatch, err := strategicMergeJsonPatch(original, override)
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

func strategicMergeJsonPatch(original, override interface{}) ([]byte, error) {
	// Convert override specs to json
	overrideJson, err := json.Marshal(override)
	if err != nil {
		return nil, err
	}

	// Convert original specs to json
	originalJson, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}

	// Merge json specs with StrategicMerge
	mergedPatch, err := sp.StrategicMergePatch(originalJson, overrideJson, v1.PodSpec{})
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
