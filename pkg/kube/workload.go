package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jpillora/backoff"
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

// WaitOnStatefulSetReady waits for the stateful set to be ready
func WaitOnStatefulSetReady(ctx context.Context, kubeCli kubernetes.Interface, ss *v1beta1.StatefulSet) bool {
	boff := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    workloadMinBackoff,
		Max:    workloadMaxBackoff,
	}

	for {
		var err error
		options := metav1.GetOptions{}
		ss, err = kubeCli.AppsV1beta1().StatefulSets(ss.Namespace).Get(ss.Name, options)
		if err != nil {
			return false
		}

		if ss.Status.ReadyReplicas == *ss.Spec.Replicas {
			return true
		}

		// Bail if we hit the max backoff
		if boff.ForAttempt(boff.Attempt()) == boff.Max {
			return false
		}
		time.Sleep(boff.Duration())
	}
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
		var err error
		options := metav1.GetOptions{}
		d, err = kubeCli.AppsV1beta1().Deployments(d.Namespace).Get(d.Name, options)
		if err != nil {
			return false
		}
		if d.Status.AvailableReplicas == *d.Spec.Replicas {
			return true
		}

		//Bail if we hit the max backoff
		if boff.ForAttempt(boff.Attempt()) == boff.Max {
			return false
		}
		time.Sleep(boff.Duration())
	}
}

// TODO(tom): We should figure out why these helm labels are not getting passed on.
var labelBlackList = map[string]struct{}{
	"chart":    struct{}{},
	"heritage": struct{}{},
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

// FetchPods fetches the pods matching the specified labels and owner UID
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
		ps = append(ps, pod)
	}
	return ps, nil
}

// DeploymentVolumes returns the PVCs referenced by this deployment as a [PVC name]->[pod spec volume name] map
func DeploymentVolumes(cli kubernetes.Interface, d *v1beta1.Deployment) (pvcToVolName map[string]string) {
	pvcToVolName = make(map[string]string)
	for _, v := range d.Spec.Template.Spec.Volumes {
		// We only care about persistent volume claims for now.
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvcToVolName[v.PersistentVolumeClaim.ClaimName] = v.Name
	}
	return pvcToVolName
}

// From getPersistentVolumeClaimName() in stateful_set_utils.go in the K8s repository
// Format is "<claim name>-<stateful set name>-<ordinal>"
const ssetVolumeClaimFmt = "%s-%s-%d"

// StatefulSetVolumes returns the PVCs referenced by this statefulset as a [PVC name]->[pod spec volume name] map
func StatefulSetVolumes(cli kubernetes.Interface, sset *v1beta1.StatefulSet) (pvcToVolName map[string]string) {
	replicas := int(*sset.Spec.Replicas)
	pvcToVolName = make(map[string]string)
	for _, v := range sset.Spec.Template.Spec.Volumes {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		for i := 0; i < replicas; i++ {
			claimName := fmt.Sprintf(ssetVolumeClaimFmt, v.PersistentVolumeClaim.ClaimName, sset.Name, i)
			pvcToVolName[claimName] = v.Name
		}
	}
	return pvcToVolName
}
