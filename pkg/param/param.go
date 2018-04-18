package param

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
)

const timeFormat = time.RFC3339Nano

// TemplateParams are the values that will change between separate runs of Phases.
type TemplateParams struct {
	StatefulSet  *StatefulSetParams
	Deployment   *DeploymentParams
	ArtifactsIn  map[string]crv1alpha1.Artifact
	ArtifactsOut map[string]crv1alpha1.Artifact
	ConfigMaps   map[string]v1.ConfigMap
	Secrets      map[string]v1.Secret
	Time         string
	Profile      *Profile
}

// StatefulSetParams are params for stateful sets.
type StatefulSetParams struct {
	Name                   string
	Namespace              string
	Pods                   []string
	Containers             [][]string
	PersistentVolumeClaims [][]string
}

// DeploymentParams are params for deployments
type DeploymentParams struct {
	Name                   string
	Namespace              string
	Pods                   []string
	Containers             [][]string
	PersistentVolumeClaims [][]string
}

// Profile contains where to store artifacts and how to access them.
type Profile struct {
	Location      crv1alpha1.Location
	Credential    Credential
	SkipSSLVerify bool
}

// CredentialType
type CredentialType string

const (
	CredentialTypeKeyPair CredentialType = "keyPair"
)

// Credential resolves the storage
type Credential struct {
	Type    CredentialType
	KeyPair *KeyPair
}

// KeyPair is a credential that contains two strings: an ID and a secret.
type KeyPair struct {
	ID     string
	Secret string
}

// New function fetches and returns the desired params
func New(ctx context.Context, cli kubernetes.Interface, crCli versioned.Interface, as crv1alpha1.ActionSpec) (*TemplateParams, error) {
	secrets, err := fetchSecrets(ctx, cli, as.Secrets)
	if err != nil {
		return nil, err
	}
	cms, err := fetchConfigMaps(ctx, cli, as.ConfigMaps)
	if err != nil {
		return nil, err
	}
	prof, err := fetchProfile(ctx, cli, crCli, as.Profile)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	tp := TemplateParams{
		ArtifactsIn: as.Artifacts,
		ConfigMaps:  cms,
		Secrets:     secrets,
		Profile:     prof,
		Time:        now.Format(timeFormat),
	}
	switch strings.ToLower(as.Object.Kind) {
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

func fetchProfile(ctx context.Context, cli kubernetes.Interface, crCli versioned.Interface, ref *crv1alpha1.ObjectReference) (*Profile, error) {
	if ref == nil {
		return nil, nil
	}
	p, err := crCli.CrV1alpha1().Profiles(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cred, err := fetchCredential(ctx, cli, p.Credential)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Profile{
		Location:      p.Location,
		Credential:    *cred,
		SkipSSLVerify: p.SkipSSLVerify,
	}, nil
}

func fetchCredential(ctx context.Context, cli kubernetes.Interface, c crv1alpha1.Credential) (*Credential, error) {
	switch c.Type {
	case crv1alpha1.CredentialTypeKeyPair:
		return fetchKeyPairCredential(ctx, cli, c.KeyPair)
	default:
		return nil, errors.Errorf("CredentialType '%s' not supported", c.Type)
	}
}

func fetchKeyPairCredential(ctx context.Context, cli kubernetes.Interface, c *crv1alpha1.KeyPair) (*Credential, error) {
	if c == nil {
		return nil, errors.New("KVSecret cannot be nil")
	}
	s, err := cli.CoreV1().Secrets(c.Secret.Namespace).Get(c.Secret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if _, ok := s.Data[c.IDField]; !ok {
		return nil, errors.Errorf("Key '%s' not found in secret '%s:%s'", c.IDField, s.GetNamespace(), s.GetName())
	}
	if _, ok := s.Data[c.SecretField]; !ok {
		return nil, errors.Errorf("Value '%s' not found in secret '%s:%s'", c.SecretField, s.GetNamespace(), s.GetName())
	}
	return &Credential{
		Type: CredentialTypeKeyPair,
		KeyPair: &KeyPair{
			ID:     string(s.Data[c.IDField]),
			Secret: string(s.Data[c.SecretField]),
		},
	}, nil
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
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
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
