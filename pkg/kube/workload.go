package kube

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	v1beta1ext "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/poll"
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
func CreateDeployment(ctx context.Context, cli kubernetes.Interface, namespace string, spec string) (*v1beta1.Deployment, error) {
	dep := &v1beta1.Deployment{}
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, dep); err != nil {
		return nil, err
	}
	return cli.AppsV1beta1().Deployments(namespace).Create(dep)
}

// CreateStatefulSet creates a stateful set from a yaml spec.
func CreateStatefulSet(ctx context.Context, cli kubernetes.Interface, namespace string, spec string) (*v1beta1.StatefulSet, error) {
	ss := &v1beta1.StatefulSet{}
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, ss); err != nil {
		return nil, err
	}
	return cli.AppsV1beta1().StatefulSets(namespace).Create(ss)
}

// StatefulSetReady checks if a statefulset has the desired number of ready
// replicas.
func StatefulSetReady(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string) (bool, error) {
	ss, err := kubeCli.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "could not get StatefulSet{Namespace: %s, Name: %s}", namespace, name)
	}
	if ss.Status.ReadyReplicas != *ss.Spec.Replicas {
		return false, nil
	}
	pods, err := FetchRunningPods(kubeCli, namespace, ss.GetUID())
	if err != nil {
		return false, err
	}
	return len(pods) == int(*ss.Spec.Replicas), nil
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
	d, err := kubeCli.AppsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "could not get Deployment{Namespace: %s, Name: %s}", namespace, name)
	}
	if d.Status.AvailableReplicas != *d.Spec.Replicas {
		return false, nil
	}
	rs, err := FetchReplicaSet(kubeCli, namespace, d.GetUID())
	if err != nil {
		return false, err
	}
	pods, err := FetchRunningPods(kubeCli, namespace, rs.GetUID())
	if err != nil {
		return false, err
	}
	return len(pods) == int(d.Status.AvailableReplicas), nil
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
func FetchReplicaSet(cli kubernetes.Interface, namespace string, uid types.UID) (*v1beta1ext.ReplicaSet, error) {
	opts := metav1.ListOptions{}
	rss, err := cli.Extensions().ReplicaSets(namespace).List(opts)
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
		return &rs, nil
	}
	return nil, errors.Wrap(errNotFound, "Could not find a ReplicaSet for Deployment")
}

// FetchRunningPods fetches the running pods matching the specified owner UID
func FetchRunningPods(cli kubernetes.Interface, namespace string, uid types.UID) ([]v1.Pod, error) {
	opts := metav1.ListOptions{}
	pods, err := cli.Core().Pods(namespace).List(opts)
	if err != nil {
		return nil, errors.Wrap(err, "Could not list Pods")
	}
	ps := make([]v1.Pod, 0, len(pods.Items))
	for _, pod := range pods.Items {
		if len(pod.OwnerReferences) != 1 ||
			pod.OwnerReferences[0].UID != uid {
			continue
		}
		if pod.Status.Phase != v1.PodRunning {
			continue
		}
		ps = append(ps, pod)
	}
	return ps, nil
}

func ScaleStatefulSet(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string, replicas int32) error {
	ss, err := kubeCli.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Could not get Statefulset{Namespace %s, Name: %s}", namespace, name)
	}
	ss.Spec.Replicas = &replicas
	ss, err = kubeCli.AppsV1beta1().StatefulSets(namespace).Update(ss)
	if err != nil {
		return errors.Wrapf(err, "Could not update Statefulset{Namespace %s, Name: %s}", namespace, name)
	}
	return WaitOnStatefulSetReady(ctx, kubeCli, namespace, name)
}

func ScaleDeployment(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string, replicas int32) error {
	d, err := kubeCli.AppsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Could not get Deployment{Namespace %s, Name: %s}", namespace, name)
	}
	d.Spec.Replicas = &replicas
	d, err = kubeCli.AppsV1beta1().Deployments(namespace).Update(d)
	if err != nil {
		return errors.Wrapf(err, "Could not update Deployment{Namespace %s, Name: %s}", namespace, name)
	}
	return WaitOnDeploymentReady(ctx, kubeCli, namespace, name)
}
