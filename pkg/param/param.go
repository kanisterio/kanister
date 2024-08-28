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
	"fmt"
	"strconv"
	"strings"
	"time"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	timeFormat         = time.RFC3339Nano
	clusterLocalDomain = "svc.cluster.local"
)

// TemplateParams are the values that will change between separate runs of Phases.
type TemplateParams struct {
	StatefulSet      *StatefulSetParams
	DeploymentConfig *DeploymentConfigParams
	Deployment       *DeploymentParams
	PVC              *PVCParams
	Namespace        *NamespaceParams
	ArtifactsIn      map[string]crv1alpha1.Artifact
	ConfigMaps       map[string]corev1.ConfigMap
	Secrets          map[string]corev1.Secret
	Time             string
	Profile          *Profile
	RepositoryServer *RepositoryServer
	Options          map[string]string
	Object           map[string]interface{}
	Phases           map[string]*Phase
	DeferPhase       *Phase
	PodOverride      crv1alpha1.JSONMap
	PodAnnotations   map[string]string
	PodLabels        map[string]string
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
	Name      string
	Namespace string
}

// Profile contains where to store artifacts and how to access them.
type Profile struct {
	Location      crv1alpha1.Location
	Credential    Credential
	SkipSSLVerify bool
}

type CredentialType string

const (
	CredentialTypeKeyPair CredentialType = "keyPair"
	CredentialTypeSecret  CredentialType = "secret"
	CredentialTypeKopia   CredentialType = "kopia"
)

// Credential resolves the storage
type Credential struct {
	Type              CredentialType
	KeyPair           *KeyPair
	Secret            *corev1.Secret
	KopiaServerSecret *KopiaServerCreds
}

// KeyPair is a credential that contains two strings: an ID and a secret.
type KeyPair struct {
	ID     string
	Secret string
}

// KopiaServerCreds contains creds to communicate with Kopia server
type KopiaServerCreds struct {
	Username       string
	Hostname       string
	Password       string
	Cert           string
	ConnectOptions map[string]int
}

// RepositoryServer contains fields from Repository server CR that will be used to resolve go templates for repository server in blueprint
type RepositoryServer struct {
	Name            string
	Namespace       string
	ServerInfo      crv1alpha1.ServerInfo
	Username        string
	Credentials     RepositoryServerCredentials
	Address         string
	ContentCacheMB  int
	MetadataCacheMB int
}

type RepositoryServerCredentials struct {
	ServerTLS        corev1.Secret
	ServerUserAccess corev1.Secret
}

// Phase represents a Blueprint phase and contains the phase output
type Phase struct {
	Secrets map[string]corev1.Secret
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
	repoServer, err := fetchRepositoryServer(ctx, cli, crCli, as.RepositoryServer)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	tp := TemplateParams{
		ArtifactsIn:      as.Artifacts,
		ConfigMaps:       cms,
		Secrets:          secrets,
		Profile:          prof,
		RepositoryServer: repoServer,
		Time:             now.Format(timeFormat),
		Options:          as.Options,
		PodOverride:      as.PodOverride,
		PodAnnotations:   as.PodAnnotations,
		PodLabels:        as.PodLabels,
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
		tp.Namespace = &NamespaceParams{Name: as.Object.Namespace, Namespace: as.Object.Namespace}
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
	u, err := kube.FetchUnstructuredObjectWithCli(ctx, dynCli, gvr, namespace, as.Object.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "could not fetch object name: %s, namespace: %s, group: %s, version: %s, resource: %s", as.Object.Name, namespace, gvr.Group, gvr.Version, gvr.Resource)
	}
	tp.Object = u.UnstructuredContent()

	return &tp, nil
}

func fetchProfile(ctx context.Context, cli kubernetes.Interface, crCli versioned.Interface, ref *crv1alpha1.ObjectReference) (*Profile, error) {
	if ref == nil {
		log.Debug().Print("Executing the action without a profile")
		return nil, nil
	}
	p, err := crCli.CrV1alpha1().Profiles(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
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

func fetchRepositoryServer(ctx context.Context, cli kubernetes.Interface, crCli versioned.Interface, ref *crv1alpha1.ObjectReference) (*RepositoryServer, error) {
	if ref == nil {
		log.Debug().Print("Executing the action without a repository-server")
		return nil, nil
	}
	r, err := crCli.CrV1alpha1().RepositoryServers(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	serverTLS, err := secretFromSecretRef(ctx, cli, r.Spec.Server.TLSSecretRef)
	if err != nil {
		return nil, err
	}
	serverUserAccess, err := secretFromSecretRef(ctx, cli, r.Spec.Server.UserAccess.UserAccessSecretRef)
	if err != nil {
		return nil, err
	}
	repoServerSecrets := RepositoryServerCredentials{
		ServerTLS:        *serverTLS,
		ServerUserAccess: *serverUserAccess,
	}
	repositoryServerService, err := cli.CoreV1().Services(r.Namespace).Get(ctx, r.Status.ServerInfo.ServiceName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Error Fetching Repository Server Service")
	}
	repositoryServerAddress := fmt.Sprintf("https://%s.%s.%s:%d", repositoryServerService.Name, repositoryServerService.Namespace, clusterLocalDomain, repositoryServerService.Spec.Ports[0].Port)
	cacheSizeSettings := getKopiaRepositoryCacheSize(r)

	return &RepositoryServer{
		Name:            r.Name,
		Namespace:       r.Namespace,
		ServerInfo:      r.Status.ServerInfo,
		Username:        r.Spec.Server.UserAccess.Username,
		Credentials:     repoServerSecrets,
		Address:         repositoryServerAddress,
		ContentCacheMB:  *cacheSizeSettings.Content,
		MetadataCacheMB: *cacheSizeSettings.Metadata,
	}, nil
}

func getKopiaRepositoryCacheSize(rs *crv1alpha1.RepositoryServer) crv1alpha1.CacheSizeSettings {
	defaultContentCacheMB, defaultMetadataCacheMB := command.GetGeneralCacheSizeSettings()
	cacheSizeSettings := crv1alpha1.CacheSizeSettings{
		Metadata: &defaultMetadataCacheMB,
		Content:  &defaultContentCacheMB,
	}
	if rs.Spec.Repository.CacheSizeSettings.Content != nil {
		cacheSizeSettings.Content = rs.Spec.Repository.CacheSizeSettings.Content
	}
	if rs.Spec.Repository.CacheSizeSettings.Metadata != nil {
		cacheSizeSettings.Metadata = rs.Spec.Repository.CacheSizeSettings.Metadata
	}
	return cacheSizeSettings
}

func fetchCredential(ctx context.Context, cli kubernetes.Interface, c crv1alpha1.Credential) (*Credential, error) {
	switch c.Type {
	case crv1alpha1.CredentialTypeKeyPair:
		return fetchKeyPairCredential(ctx, cli, c.KeyPair)
	case crv1alpha1.CredentialTypeSecret:
		return fetchSecretCredential(ctx, cli, c.Secret)
	case crv1alpha1.CredentialTypeKopia:
		return fetchKopiaCredential(ctx, cli, c.KopiaServerSecret)
	default:
		return nil, errors.Errorf("CredentialType '%s' not supported", c.Type)
	}
}

func fetchKeyPairCredential(ctx context.Context, cli kubernetes.Interface, c *crv1alpha1.KeyPair) (*Credential, error) {
	if c == nil {
		return nil, errors.New("KVSecret cannot be nil")
	}
	s, err := cli.CoreV1().Secrets(c.Secret.Namespace).Get(ctx, c.Secret.Name, metav1.GetOptions{})
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
	s, err := cli.CoreV1().Secrets(sr.Namespace).Get(ctx, sr.Name, metav1.GetOptions{})
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

func secretFromSecretRef(ctx context.Context, cli kubernetes.Interface, ref corev1.SecretReference) (*corev1.Secret, error) {
	secret, err := cli.CoreV1().Secrets(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error fetching secret %s from namespace %s", ref.Name, ref.Namespace))
	}
	return secret, nil
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

func fetchSecrets(ctx context.Context, cli kubernetes.Interface, refs map[string]crv1alpha1.ObjectReference) (map[string]corev1.Secret, error) {
	secrets := make(map[string]corev1.Secret, len(refs))
	for name, ref := range refs {
		s, err := cli.CoreV1().Secrets(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		secrets[name] = *s
	}
	return secrets, nil
}

func fetchConfigMaps(ctx context.Context, cli kubernetes.Interface, refs map[string]crv1alpha1.ObjectReference) (map[string]corev1.ConfigMap, error) {
	configs := make(map[string]corev1.ConfigMap, len(refs))
	for name, ref := range refs {
		c, err := cli.CoreV1().ConfigMaps(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		configs[name] = *c
	}
	return configs, nil
}

func fetchStatefulSetParams(ctx context.Context, cli kubernetes.Interface, namespace, name string) (*StatefulSetParams, error) {
	ss, err := cli.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
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
	dc, err := osCli.AppsV1().DeploymentConfigs(namespace).Get(ctx, name, metav1.GetOptions{})
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

	// deployment configs are managed by replicationcontrollers not replicaset
	// get the replication controller of the deploymentconfig
	rc, err := kube.FetchReplicationController(cli, namespace, dc.UID, strconv.FormatInt(dc.Status.LatestVersion, 10))
	if err != nil {
		return nil, err
	}

	pods, _, err := kube.FetchPods(cli, namespace, rc.UID)
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
	d, err := cli.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
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

func containerNames(pod corev1.Pod) []string {
	cs := make([]string, 0, len(pod.Status.ContainerStatuses))
	for _, c := range pod.Status.ContainerStatuses {
		cs = append(cs, c.Name)
	}
	return cs
}

func volumes(pod corev1.Pod, volToPvc map[string]string) map[string]string {
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
	_, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &PVCParams{
		Name:      name,
		Namespace: namespace,
	}, nil
}

func fetchKopiaCredential(ctx context.Context, cli kubernetes.Interface, ks *crv1alpha1.KopiaServerSecret) (*Credential, error) {
	if ks == nil {
		return nil, errors.New("Kopia Secret reference cannot be nil")
	}

	if ks.UserPassphrase.Secret == nil {
		return nil, errors.New("Kopia UserPassphrase Secret reference cannot be nil")
	}

	passSecret, err := cli.CoreV1().Secrets(ks.UserPassphrase.Secret.Namespace).Get(ctx, ks.UserPassphrase.Secret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch the secret %s/%s", ks.UserPassphrase.Secret.Namespace, ks.UserPassphrase.Secret.Name)
	}
	password, ok := passSecret.Data[ks.UserPassphrase.Key]
	if !ok {
		return nil, errors.New("Failed to fetch user passphrase from secret")
	}

	if ks.TLSCert == nil || ks.TLSCert.Secret == nil {
		return nil, errors.New("Kopia TLS cert Secret reference cannot be nil")
	}
	tlsSecret, err := cli.CoreV1().Secrets(ks.TLSCert.Secret.Namespace).Get(ctx, ks.TLSCert.Secret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch the secret %s/%s", ks.TLSCert.Secret.Namespace, ks.TLSCert.Secret.Name)
	}
	tlsCert, ok := tlsSecret.Data[ks.TLSCert.Key]
	if !ok {
		return nil, errors.New("Failed to fetch TLS cert from secret")
	}
	return &Credential{
		Type: CredentialTypeKopia,
		KopiaServerSecret: &KopiaServerCreds{
			Hostname:       ks.Hostname,
			Username:       ks.Username,
			Password:       string(password),
			Cert:           string(tlsCert),
			ConnectOptions: ks.ConnectOptions,
		},
	}, nil
}

// UpdatePhaseParams updates the TemplateParams with Phase information
func UpdatePhaseParams(ctx context.Context, tp *TemplateParams, phaseName string, output map[string]interface{}) {
	tp.Phases[phaseName].Output = output
}

// UpdateDeferPhaseParams updates the TemplateParams deferPhase output with passed output
// This output would be generated/passed by execution of the phase
func UpdateDeferPhaseParams(ctx context.Context, tp *TemplateParams, output map[string]interface{}) {
	tp.DeferPhase.Output = output
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

	tp.DeferPhase = &Phase{
		Secrets: secrets,
	}

	return nil
}
