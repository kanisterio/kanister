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
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	// RevisionAnnotation is the revision annotation of a deployment's replica sets which records its rollout sequence
	RevisionAnnotation = "deployment.kubernetes.io/revision"
)

// CreateConfigMap creates a configmap set from a yaml spec.
func CreateConfigMap(ctx context.Context, cli kubernetes.Interface, namespace string, spec string) (*v1.ConfigMap, error) {
	cm := &v1.ConfigMap{}
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, cm); err != nil {
		return nil, err
	}
	return cli.CoreV1().ConfigMaps(namespace).Create(cm)
}

// CreateDeployment creates a deployment set from a yaml spec.
func CreateDeployment(ctx context.Context, cli kubernetes.Interface, namespace string, spec string) (*appsv1.Deployment, error) {
	dep := &appsv1.Deployment{}
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, dep); err != nil {
		return nil, err
	}
	return cli.AppsV1().Deployments(namespace).Create(dep)
}

// CreateStatefulSet creates a stateful set from a yaml spec.
func CreateStatefulSet(ctx context.Context, cli kubernetes.Interface, namespace string, spec string) (*appsv1.StatefulSet, error) {
	ss := &appsv1.StatefulSet{}
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, ss); err != nil {
		return nil, err
	}
	return cli.AppsV1().StatefulSets(namespace).Create(ss)
}

// StatefulSetReady checks if a statefulset has the desired number of ready
// replicas.
func StatefulSetReady(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string) (bool, error) {
	ss, err := kubeCli.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "could not get StatefulSet{Namespace: %s, Name: %s}", namespace, name)
	}
	if ss.Status.ReadyReplicas != *ss.Spec.Replicas {
		return false, nil
	}
	runningPods, _, err := FetchPods(kubeCli, namespace, ss.GetUID())
	if err != nil {
		return false, err
	}
	return len(runningPods) == int(*ss.Spec.Replicas), nil
}

// WaitOnStatefulSetReady waits for the stateful set to be ready
func WaitOnStatefulSetReady(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string) error {
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		ok, err := StatefulSetReady(ctx, kubeCli, namespace, name)
		if apierrors.IsNotFound(errors.Cause(err)) {
			return false, nil
		}
		return ok, err
	})
}

// DeploymentReady checks to see if the deployment has the desired number of
// available replicas.
func DeploymentReady(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string) (bool, error) {
	d, err := kubeCli.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "could not get Deployment{Namespace: %s, Name: %s}", namespace, name)
	}
	// Wait for deployment to complete. The deployment controller will check the downstream
	// RS and Running Pods to update the deployment status
	if deploymentComplete := d.Status.UpdatedReplicas == *d.Spec.Replicas &&
		d.Status.Replicas == *d.Spec.Replicas &&
		d.Status.AvailableReplicas == *d.Spec.Replicas &&
		d.Status.ObservedGeneration >= d.Generation; !deploymentComplete {
		return false, nil
	}
	rs, err := FetchReplicaSet(kubeCli, namespace, d.GetUID(), d.Annotations[RevisionAnnotation])
	if err != nil {
		return false, err
	}
	runningPods, notRunningPods, err := FetchPods(kubeCli, namespace, rs.GetUID())
	if err != nil {
		return false, err
	}
	// The deploymentComplete check above already validates this but we do it
	// again anyway given we have this information available
	if len(runningPods) != int(d.Status.AvailableReplicas) {
		return false, nil
	}
	// Wait for things to settle. This check *is* required since the deployment controller
	// excludes any pods not running from its replica count(s)
	return len(notRunningPods) == 0, nil
}

// WaitOnDeploymentReady waits for the deployment to be ready
func WaitOnDeploymentReady(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string) error {
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		ok, err := DeploymentReady(ctx, kubeCli, namespace, name)
		if apierrors.IsNotFound(errors.Cause(err)) {
			return false, nil
		}
		return ok, err
	})
}

var errNotFound = fmt.Errorf("not found")

// FetchReplicaSet fetches the replicaset matching the specified owner UID
func FetchReplicaSet(cli kubernetes.Interface, namespace string, uid types.UID, revision string) (*appsv1.ReplicaSet, error) {
	opts := metav1.ListOptions{}
	rss, err := cli.AppsV1().ReplicaSets(namespace).List(opts)
	if err != nil {
		return nil, errors.Wrap(err, "Could not list ReplicaSets")
	}
	for _, rs := range rss.Items {
		// We ignore ReplicaSets without a single owner.
		if len(rs.OwnerReferences) != 1 {
			continue
		}
		// We ignore ReplicaSets owned by other deployments.
		if rs.OwnerReferences[0].UID != uid {
			continue
		}
		// We ignore older ReplicaSets
		if rs.Annotations[RevisionAnnotation] != revision {
			continue
		}
		return &rs, nil
	}
	return nil, errors.Wrap(errNotFound, "Could not find a ReplicaSet for Deployment")
}

// FetchPods fetches the pods matching the specified owner UID and splits them
// into 2 groups (running/not-running)
func FetchPods(cli kubernetes.Interface, namespace string, uid types.UID) (runningPods []v1.Pod, notRunningPods []v1.Pod, err error) {
	opts := metav1.ListOptions{}
	pods, err := cli.CoreV1().Pods(namespace).List(opts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not list Pods")
	}
	for _, pod := range pods.Items {
		if len(pod.OwnerReferences) != 1 ||
			pod.OwnerReferences[0].UID != uid {
			continue
		}
		if pod.Status.Phase != v1.PodRunning {
			notRunningPods = append(notRunningPods, pod)
			continue
		}
		runningPods = append(runningPods, pod)
	}
	return runningPods, notRunningPods, nil
}

func ScaleStatefulSet(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string, replicas int32) error {
	ss, err := kubeCli.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Could not get Statefulset{Namespace %s, Name: %s}", namespace, name)
	}
	ss.Spec.Replicas = &replicas
	ss, err = kubeCli.AppsV1().StatefulSets(namespace).Update(ss)
	if err != nil {
		return errors.Wrapf(err, "Could not update Statefulset{Namespace %s, Name: %s}", namespace, name)
	}
	return WaitOnStatefulSetReady(ctx, kubeCli, namespace, name)
}

func ScaleDeployment(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string, replicas int32) error {
	d, err := kubeCli.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Could not get Deployment{Namespace %s, Name: %s}", namespace, name)
	}
	d.Spec.Replicas = &replicas
	d, err = kubeCli.AppsV1().Deployments(namespace).Update(d)
	if err != nil {
		return errors.Wrapf(err, "Could not update Deployment{Namespace %s, Name: %s}", namespace, name)
	}
	return WaitOnDeploymentReady(ctx, kubeCli, namespace, name)
}

// DeploymentVolumes returns the PVCs referenced by this deployment as a [pods spec volume name]->[PVC name] map
func DeploymentVolumes(cli kubernetes.Interface, d *appsv1.Deployment) (volNameToPvc map[string]string) {
	volNameToPvc = make(map[string]string)
	for _, v := range d.Spec.Template.Spec.Volumes {
		// We only care about persistent volume claims for now.
		if v.PersistentVolumeClaim == nil {
			continue
		}
		volNameToPvc[v.Name] = v.PersistentVolumeClaim.ClaimName
	}
	return volNameToPvc
}

// From getPersistentVolumeClaimName() in stateful_set_utils.go in the K8s repository
// Format is "<claim name>-<stateful set name>-<ordinal>"
const (
	ssetVolumeClaimFmt = "%s-%s-%d"
	ssetPodRegex       = ".*-([0-9]+)$"
)

// From getParentNameAndOrdinal() in stateful_set_utils.go in the K8s repository
func getOrdinal(pod string) int {
	ordinal := -1
	ssetPodRegex := regexp.MustCompile(ssetPodRegex)
	matches := ssetPodRegex.FindStringSubmatch(pod)
	if len(matches) != 2 {
		return ordinal
	}
	if i, err := strconv.ParseInt(matches[1], 10, 32); err == nil {
		ordinal = int(i)
	}
	return ordinal
}

// StatefulSetVolumes returns the PVCs referenced by a pod in this statefulset as a [pod spec volume name]->[PVC name] map
func StatefulSetVolumes(cli kubernetes.Interface, sset *appsv1.StatefulSet, pod *v1.Pod) (volNameToPvc map[string]string) {
	ordinal := getOrdinal(pod.Name)
	if ordinal == -1 {
		// Pod not created through the statefulset?
		return nil
	}
	claimTemplateNameToPodVolumeName := make(map[string]string)
	for _, v := range sset.Spec.Template.Spec.Volumes {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		claimTemplateNameToPodVolumeName[v.PersistentVolumeClaim.ClaimName] = v.Name
	}
	// Check if there are any PVC claims in the `volumeClaimTemplates` section not directly referenced in
	// the pod template
	for _, vct := range sset.Spec.VolumeClaimTemplates {
		if _, ok := claimTemplateNameToPodVolumeName[vct.Name]; !ok {
			// The StatefulSet controller automatically generates references for claims not explicitly
			// referenced and uses the claim template name as the pod volume name
			// to account for these.
			claimTemplateNameToPodVolumeName[vct.Name] = vct.Name
		}
	}
	volNameToPvc = make(map[string]string)
	for claimTemplateName, podVolName := range claimTemplateNameToPodVolumeName {
		claimName := fmt.Sprintf(ssetVolumeClaimFmt, claimTemplateName, sset.Name, ordinal)
		volNameToPvc[podVolName] = claimName
	}
	return volNameToPvc
}
