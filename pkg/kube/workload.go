package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	v1beta1ext "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	workloadMaxBackoff = 10 * time.Minute
	workloadMinBackoff = 10 * time.Millisecond
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
	return ss.Status.ReadyReplicas == *ss.Spec.Replicas, nil
}

// WaitOnStatefulSetReady waits for the stateful set to be ready
func WaitOnStatefulSetReady(ctx context.Context, kubeCli kubernetes.Interface, ss *v1beta1.StatefulSet) bool {
	boff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    workloadMinBackoff,
		Max:    workloadMaxBackoff,
	}

	for {
		ok, err := StatefulSetReady(ctx, kubeCli, ss.GetNamespace(), ss.GetName())
		if err != nil {
			return false
		}
		if ok {
			return true
		}
		// Bail if we hit the max backoff
		if boff.ForAttempt(boff.Attempt()) == boff.Max {
			return false
		}
		time.Sleep(boff.Duration())
	}
}

// DeploymentReady checks to see if the deployment has the desired number of
// available replicas.
func DeploymentReady(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string) (bool, error) {
	d, err := kubeCli.AppsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return d.Status.AvailableReplicas == *d.Spec.Replicas, nil
}

// WaitOnDeploymentReady waits for the deployment to be ready
func WaitOnDeploymentReady(ctx context.Context, kubeCli kubernetes.Interface, d *v1beta1.Deployment) bool {
	boff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    workloadMinBackoff,
		Max:    workloadMaxBackoff,
	}

	for {
		ok, err := DeploymentReady(ctx, kubeCli, d.GetNamespace(), d.GetName())
		if err != nil {
			return false
		}
		if ok {
			return true
		}
		//Bail if we hit the max backoff
		if boff.ForAttempt(boff.Attempt()) == boff.Max {
			return false
		}
		time.Sleep(boff.Duration())
	}
}

// We omit helm specific labels.
var labelBlackList = map[string]struct{}{
	"chart":    struct{}{},
	"heritage": struct{}{},
	"release":  struct{}{},
}

func labelSelector(labels map[string]string) string {
	ls := make([]string, 0, len(labels))
	for k, v := range labels {
		if _, ok := labelBlackList[k]; ok {
			continue
		}
		ls = append(ls, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(ls, ",")
}

// FetchReplicaSet fetches the replicaset matching the specified labels and owner UID
func FetchReplicaSet(cli kubernetes.Interface, namespace string, uid types.UID, labels map[string]string) (*v1beta1ext.ReplicaSet, error) {
	sel := labelSelector(labels)
	opts := metav1.ListOptions{LabelSelector: sel}
	rss, err := cli.Extensions().ReplicaSets(namespace).List(opts)
	if err != nil {
		return nil, err
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
		// We ignore ReplicaSets that have been scaled down.
		if rs.Status.Replicas == 0 {
			continue
		}
		return &rs, nil
	}
	return nil, fmt.Errorf("Could not find a single replicaset for deployment")
}

// FetchPods fetches the running pods matching the specified labels and owner UID
func FetchPods(cli kubernetes.Interface, namespace string, uid types.UID, labels map[string]string) ([]v1.Pod, error) {
	sel := labelSelector(labels)
	opts := metav1.ListOptions{LabelSelector: sel}
	pods, err := cli.Core().Pods(namespace).List(opts)
	if err != nil {
		return nil, err
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

func ScaleStatefulSet(ctx context.Context, kubeCli kubernetes.Interface, namespace string, statefulSetName string, scaleNumber int32) error {
	ss, err := kubeCli.AppsV1beta1().StatefulSets(namespace).Get(statefulSetName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Could not get Statefulset %s", statefulSetName)
	}
	ss.Spec.Replicas = &scaleNumber
	ss, err = kubeCli.AppsV1beta1().StatefulSets(namespace).Update(ss)
	if err != nil {
		return errors.Wrapf(err, "Could not update Statefulset %s", statefulSetName)
	}
	if !WaitOnStatefulSetReady(ctx, kubeCli, ss) {
		return errors.New(fmt.Sprintf("Failed to scale Statefulset %s\n", statefulSetName))
	}
	return nil
}

func ScaleDeployment(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string, replicas int32) error {
	d, err := kubeCli.AppsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Could not get Deployment %s", name)
	}
	d.Spec.Replicas = &replicas
	d, err = kubeCli.AppsV1beta1().Deployments(namespace).Update(d)
	if err != nil {
		return errors.Wrapf(err, "Could not update Deployment %s", name)
	}
	if !WaitOnDeploymentReady(ctx, kubeCli, d) {
		return errors.New(fmt.Sprintf("Failed to scale Deployment %s\n", name))
	}
	return nil
}
