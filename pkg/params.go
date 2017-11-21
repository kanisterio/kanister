package kanister

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

const timeFormat = time.RFC3339Nano

func NewTemplateParams(ctx context.Context, cli kubernetes.Interface, as crv1alpha1.ActionSpec) (*TemplateParams, error) {
	secrets, err := fetchSecrets(ctx, cli, as.Secrets)
	if err != nil {
		return nil, err
	}
	cms, err := fetchConfigMaps(ctx, cli, as.ConfigMaps)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	tp := TemplateParams{
		ArtifactsIn: as.Artifacts,
		ConfigMaps:  cms,
		Secrets:     secrets,
		Time:        now.Format(timeFormat),
	}
	switch as.Object.Kind {
	case "statefulset":
		ssp, err := fetchStatefulSetParams(ctx, cli, as.Object.Namespace, as.Object.Name)
		if err != nil {
			return nil, err
		}
		tp.StatefulSet = ssp
	case "deployment":
		dp, err := fetchDeploymentParams(ctx, cli, as.Object.Namespace, as.Object.Name)
		if err != nil {
			return nil, err
		}
		tp.Deployment = dp
	default:
		return nil, errors.Errorf("Resource '%s' not supported", as.Object.Kind)
	}
	return &tp, nil
}

func fetchSecrets(ctx context.Context, cli kubernetes.Interface, refs map[string]crv1alpha1.ObjectReference) (map[string]v1.Secret, error) {
	secrets := make(map[string]v1.Secret, len(refs))
	for name, ref := range refs {
		s, err := cli.CoreV1().Secrets(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		secrets[name] = *s
	}
	return secrets, nil
}

func fetchConfigMaps(ctx context.Context, cli kubernetes.Interface, refs map[string]crv1alpha1.ObjectReference) (map[string]v1.ConfigMap, error) {
	configs := make(map[string]v1.ConfigMap, len(refs))
	for name, ref := range refs {
		c, err := cli.CoreV1().ConfigMaps(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		configs[name] = *c
	}
	return configs, nil
}

func fetchStatefulSetParams(ctx context.Context, cli kubernetes.Interface, namespace, name string) (*StatefulSetParams, error) {
	ss, err := cli.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ssp := &StatefulSetParams{
		Name:       name,
		Namespace:  namespace,
		Pods:       []string{},
		Containers: [][]string{},
	}
	pods, err := kube.FetchPods(cli, namespace, ss.UID, ss.GetLabels())
	if err != nil {
		return nil, err
	}
	for _, p := range pods {
		ssp.Pods = append(ssp.Pods, p.Name)
		ssp.Containers = append(ssp.Containers, containerNames(p))
		ssp.PersistentVolumeClaims = append(ssp.PersistentVolumeClaims, volumeNames(p))
	}
	return ssp, nil
}

func fetchDeploymentParams(ctx context.Context, cli kubernetes.Interface, namespace, name string) (*DeploymentParams, error) {
	d, err := cli.AppsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	dp := &DeploymentParams{
		Name:       name,
		Namespace:  namespace,
		Pods:       []string{},
		Containers: [][]string{},
	}
	rs, err := kube.FetchReplicaSet(cli, namespace, d.UID, d.GetLabels())
	if err != nil {
		return nil, err
	}
	pods, err := kube.FetchPods(cli, namespace, rs.UID, d.GetLabels())
	if err != nil {
		return nil, err
	}
	for _, p := range pods {
		dp.Pods = append(dp.Pods, p.Name)
		dp.Containers = append(dp.Containers, containerNames(p))
		dp.PersistentVolumeClaims = append(dp.PersistentVolumeClaims, volumeNames(p))
	}
	return dp, nil
}

func containerNames(pod v1.Pod) []string {
	cs := make([]string, 0, len(pod.Status.ContainerStatuses))
	for _, c := range pod.Status.ContainerStatuses {
		cs = append(cs, c.Name)
	}
	return cs
}

func volumeNames(pod v1.Pod) []string {
	vs := make([]string, 0, len(pod.Status.ContainerStatuses))
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim != nil {
			vs = append(vs, v.Name)
		}
	}
	return vs
}
