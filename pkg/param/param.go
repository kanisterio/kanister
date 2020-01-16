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

package param

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/secrets"
	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
)

const timeFormat = time.RFC3339Nano

// TemplateParams are the values that will change between separate runs of Phases.
type TemplateParams struct {
	StatefulSet      *StatefulSetParams
	DeploymentConfig *DeploymentConfigParams
	Deployment       *DeploymentParams
	PVC              *PVCParams
	Namespace        *NamespaceParams
	ArtifactsIn      map[string]crv1alpha1.Artifact
	ConfigMaps       map[string]v1.ConfigMap
	Secrets          map[string]v1.Secret
	Time             string
	Profile          *Profile
	Options          map[string]string
	Object           map[string]interface{}
	Phases           map[string]*Phase
	PodOverride      crv1alpha1.JSONMap
}

// DeploymentConfigParams are params for deploymentconfig, will be used if working on open shift cluster
// https://docs.openshift.com/container-platform/4.1/applications/deployments/what-deployments-are.html
type DeploymentConfigParams struct {
	Name                   string
	Namespace              string
	Pods                   []string
	Containers             [][]string
	PersistentVolumeClaims map[string]map[string]string
}

// StatefulSetParams are params for stateful sets.
type StatefulSetParams struct {
	Name                   string
	Namespace              string
	Pods                   []string
	Containers             [][]string
	PersistentVolumeClaims map[string]map[string]string
}

// DeploymentParams are params for deployments
type DeploymentParams struct {
	Name                   string
	Namespace              string
	Pods                   []string
	Containers             [][]string
	PersistentVolumeClaims map[string]map[string]string
}

// PVCParams are params for persistent volume claims
type PVCParams struct {
	Name      string
	Namespace string
}

// NamespaceParams are params for namespaces
type NamespaceParams struct {
	Name string
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
	CredentialTypeSecret  CredentialType = "secret"
)

// Credential resolves the storage
type Credential struct {
	Type    CredentialType
	KeyPair *KeyPair
	Secret  *v1.Secret
}

// KeyPair is a credential that contains two strings: an ID and a secret.
type KeyPair struct {
	ID     string
	Secret string
}

// Phase represents a Blueprint phase and contains the phase output
type Phase struct {
	Secrets map[string]v1.Secret
	Output  map[string]interface{}
}

const (
	DeploymentKind       = "deployment"
	StatefulSetKind      = "statefulset"
	DeploymentConfigKind = "deploymentconfig"
	PVCKind              = "pvc"
	NamespaceKind        = "namespace"
	SecretKind           = "secret"
)

// New function fetches and returns the desired params
func New(ctx context.Context, cli kubernetes.Interface, dynCli dynamic.Interface, crCli versioned.Interface, osCli osversioned.Interface, as crv1alpha1.ActionSpec) (*TemplateParams, error) {
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
		Options:     as.Options,
		PodOverride: as.PodOverride,
	}
	var gvr schema.GroupVersionResource
	namespace := as.Object.Namespace
	switch strings.ToLower(as.Object.Kind) {
	case StatefulSetKind:
		ssp, err := fetchStatefulSetParams(ctx, cli, as.Object.Namespace, as.Object.Name)
		if err != nil {
			return nil, err
		}
		tp.StatefulSet = ssp
		gvr = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	case DeploymentConfigKind:
		dcp, err := fetchDeploymentConfigParams(ctx, cli, osCli, as.Object.Namespace, as.Object.Name)
		if err != nil {
			return nil, err
		}
		tp.DeploymentConfig = dcp
		gvr = schema.GroupVersionResource{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}
	case DeploymentKind:
		dp, err := fetchDeploymentParams(ctx, cli, as.Object.Namespace, as.Object.Name)
		if err != nil {
			return nil, err
		}
		tp.Deployment = dp
		gvr = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	case PVCKind:
		pp, err := fetchPVCParams(ctx, cli, as.Object.Namespace, as.Object.Name)
		if err != nil {
			return nil, err
		}
		tp.PVC = pp
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
	case NamespaceKind:
		tp.Namespace = &NamespaceParams{Name: as.Object.Namespace}
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
		// `Namespace` is a global resource
		namespace = ""
	default:
		gvr = schema.GroupVersionResource{
			Group:    as.Object.Group,
			Version:  as.Object.APIVersion,
			Resource: as.Object.Resource,
		}
	}
	u, err := kube.FetchUnstructuredObjectWithCli(dynCli, gvr, namespace, as.Object.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "could not fetch object name: %s, namespace: %s, group: %s, version: %s, resource: %s", as.Object.Name, namespace, gvr.Group, gvr.Version, gvr.Resource)
	}
	tp.Object = u.UnstructuredContent()

	return &tp, nil
}

func fetchProfile(ctx context.Context, cli kubernetes.Interface, crCli versioned.Interface, ref *crv1alpha1.ObjectReference) (*Profile, error) {
	if ref == nil {
		return nil, errors.New("Cannot execute action without a profile. Specify a profile in the action set")
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
	case crv1alpha1.CredentialTypeSecret:
		return fetchSecretCredential(ctx, cli, c.Secret)
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

func fetchSecretCredential(ctx context.Context, cli kubernetes.Interface, sr *crv1alpha1.ObjectReference) (*Credential, error) {
	if sr == nil {
		return nil, errors.New("Secret reference cannot be nil")
	}
	s, err := cli.CoreV1().Secrets(sr.Namespace).Get(sr.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch the secret")
	}
	if err = secrets.ValidateCredentials(s); err != nil {
		return nil, err
	}
	return &Credential{
		Type:   CredentialTypeSecret,
		Secret: s,
	}, nil
}

func filterByKind(refs map[string]crv1alpha1.ObjectReference, kind string) map[string]crv1alpha1.ObjectReference {
	filtered := make(map[string]crv1alpha1.ObjectReference, len(refs))
	for name, ref := range refs {
		if !strings.EqualFold(ref.Kind, kind) {
			continue
		}
		filtered[name] = ref
	}
	return filtered
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
	ss, err := cli.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ssp := &StatefulSetParams{
		Name:                   name,
		Namespace:              namespace,
		Pods:                   []string{},
		Containers:             [][]string{},
		PersistentVolumeClaims: make(map[string]map[string]string),
	}
	pods, _, err := kube.FetchPods(cli, namespace, ss.UID)
	if err != nil {
		return nil, err
	}
	for _, p := range pods {
		ssp.Pods = append(ssp.Pods, p.Name)
		ssp.Containers = append(ssp.Containers, containerNames(p))
		if pvcToMountPath := volumes(p, kube.StatefulSetVolumes(cli, ss, &p)); len(pvcToMountPath) > 0 {
			ssp.PersistentVolumeClaims[p.Name] = pvcToMountPath
		}
	}
	return ssp, nil
}

func fetchDeploymentConfigParams(ctx context.Context, cli kubernetes.Interface, osCli osversioned.Interface, namespace, name string) (*DeploymentConfigParams, error) {
	// we will have to have another OpenShift cli to get the deployment config resource
	// because deploymentconfig is not standard kubernetes resource.
	dc, err := osCli.AppsV1().DeploymentConfigs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dcp := &DeploymentConfigParams{
		Name:                   name,
		Namespace:              namespace,
		Pods:                   []string{},
		Containers:             [][]string{},
		PersistentVolumeClaims: make(map[string]map[string]string),
	}

	pods, _, err := kube.FetchPods(cli, namespace, dc.UID)
	if err != nil {
		return nil, err
	}

	for _, p := range pods {
		dcp.Pods = append(dcp.Pods, p.Name)
		dcp.Containers = append(dcp.Containers, containerNames(p))
		if pvcToMountPath := volumes(p, kube.DeploymentConfigVolumes(osCli, dc, &p)); len(pvcToMountPath) > 0 {
			dcp.PersistentVolumeClaims[p.Name] = pvcToMountPath
		}
	}

	return dcp, nil
}

func fetchDeploymentParams(ctx context.Context, cli kubernetes.Interface, namespace, name string) (*DeploymentParams, error) {
	d, err := cli.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	dp := &DeploymentParams{
		Name:                   name,
		Namespace:              namespace,
		Pods:                   []string{},
		Containers:             [][]string{},
		PersistentVolumeClaims: make(map[string]map[string]string),
	}
	rs, err := kube.FetchReplicaSet(cli, namespace, d.UID, d.Annotations[kube.RevisionAnnotation])
	if err != nil {
		return nil, err
	}
	pods, _, err := kube.FetchPods(cli, namespace, rs.UID)
	if err != nil {
		return nil, err
	}
	volToPvc := make(map[string]string)
	if len(pods) > 0 {
		volToPvc = kube.DeploymentVolumes(cli, d)
	}
	for _, p := range pods {
		dp.Pods = append(dp.Pods, p.Name)
		dp.Containers = append(dp.Containers, containerNames(p))
		pvcToMountPath := volumes(p, volToPvc)
		if len(pvcToMountPath) > 0 {
			dp.PersistentVolumeClaims[p.Name] = pvcToMountPath
		}
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

func volumes(pod v1.Pod, volToPvc map[string]string) map[string]string {
	pvcToMountPath := make(map[string]string)
	for _, c := range pod.Spec.Containers {
		for _, v := range c.VolumeMounts {
			if pvc, ok := volToPvc[v.Name]; ok {
				pvcToMountPath[pvc] = v.MountPath
			}
		}
	}
	return pvcToMountPath
}

func fetchPVCParams(ctx context.Context, cli kubernetes.Interface, namespace, name string) (*PVCParams, error) {
	_, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &PVCParams{
		Name:      name,
		Namespace: namespace,
	}, nil
}

// UpdatePhaseParams updates the TemplateParams with Phase information
func UpdatePhaseParams(ctx context.Context, tp *TemplateParams, phaseName string, output map[string]interface{}) {
	tp.Phases[phaseName].Output = output
}

// InitPhaseParams initializes the TemplateParams with Phase information
func InitPhaseParams(ctx context.Context, cli kubernetes.Interface, tp *TemplateParams, phaseName string, objects map[string]crv1alpha1.ObjectReference) error {
	if tp.Phases == nil {
		tp.Phases = make(map[string]*Phase)
	}
	secrets, err := fetchSecrets(ctx, cli, filterByKind(objects, SecretKind))
	if err != nil {
		return err
	}
	tp.Phases[phaseName] = &Phase{
		Secrets: secrets,
	}
	return nil
}
