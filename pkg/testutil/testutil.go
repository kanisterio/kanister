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

package testutil

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/secrets"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

const (
	testBPArg                      = "key"
	S3CompliantAccessKeyIDEnv      = "S3_COMPLIANT_AWS_ACCESS_KEY_ID"
	S3CompliantSecretAccessKeyEnv  = "S3_COMPLIANT_AWS_SECRET_ACCESS_KEY"
	S3CompliantLocationEndpointEnv = "S3_COMPLIANT_LOCATION_ENDPOINT"
)

// NewTestPVC function returns a pointer to a new PVC test object
func NewTestPVC() *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}
}

// NewTestNamespace function returns a pointer to a new Namespace test object
func NewTestNamespace() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest-",
		},
	}
}

// NewTestDeployment function returns a pointer to a new Deployment test object
func NewTestDeployment(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-deployment-",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "fake-app"}},
			Template: newTestPodTemplateSpec(),
		},
	}
}

// NewTestStatefulSet function returns a pointer to a new StatefulSet test object
func NewTestStatefulSet(replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-statefulset-",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "fake-svc",
			Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app": "fake-app"}},
			Template:    newTestPodTemplateSpec(),
		},
	}
}

func newTestPodTemplateSpec() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "fake-app",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "test-container",
					Image:           consts.LatestKanisterToolsImage,
					Command:         []string{"tail"},
					Args:            []string{"-f", "/dev/null"},
					ImagePullPolicy: v1.PullAlways,
				},
			},
		},
	}
}

const TestProfileName = "test-profile"

// NewTestProfileSecret function returns a pointer to a new Secret test object.
func NewTestProfileSecret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: map[string]string{
			"id":     "foo",
			"secret": "bar",
		},
	}
}

// NewTestProfileSecretWithRole function returns a pointer to a new Secret test object with role.
func NewTestProfileSecretWithRole(role string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		Data: map[string][]byte{
			"id":     []byte("foo"),
			"secret": []byte("bar"),
			"role":   []byte(role),
		},
	}
}

// NewTestProfile function returns a pointer to a new Profile test object that
// passes validation.
func NewTestProfile(namespace string, secretName string) *crv1alpha1.Profile {
	return &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestProfileName,
			Namespace: namespace,
		},
		Location: crv1alpha1.Location{
			Type: crv1alpha1.LocationTypeS3Compliant,
		},
		Credential: crv1alpha1.Credential{
			Type: crv1alpha1.CredentialTypeKeyPair,
			KeyPair: &crv1alpha1.KeyPair{
				Secret: crv1alpha1.ObjectReference{
					Name:      secretName,
					Namespace: namespace,
				},
				IDField:     "id",
				SecretField: "secret",
			},
		},
	}
}

// NewSecretProfileFromLocation figures out Provider type from the location type and
// returns a pointer to Secret and Profile
func NewSecretProfileFromLocation(location crv1alpha1.Location) (*v1.Secret, *crv1alpha1.Profile, error) {
	var key, val string
	data := make(map[string]string)

	switch location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		key = os.Getenv(awsconfig.AccessKeyID)
		val = os.Getenv(awsconfig.SecretAccessKey)
		if role := os.Getenv(awsconfig.ConfigRole); role != "" {
			data[awsconfig.ConfigRole] = role
		}
	case crv1alpha1.LocationTypeGCS:
		os.Getenv(blockstorage.GoogleCloudCreds)
		creds, err := google.FindDefaultCredentials(context.Background(), compute.ComputeScope)
		if err != nil {
			return nil, nil, err
		}
		key = creds.ProjectID
		val = string(creds.JSON)
	case crv1alpha1.LocationTypeAzure:
		key = os.Getenv(blockstorage.AzureStorageAccount)
		val = os.Getenv(blockstorage.AzureStorageKey)
	default:
		return nil, nil, fmt.Errorf("Invalid location type '%s'", location.Type)
	}
	data["access_key_id"] = key
	data["secret_access_key"] = val
	return NewProfileSecret(data), NewProfile(location), nil
}

// NewProfileSecret function returns a pointer to a new Secret test object.
func NewProfileSecret(data map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: data,
	}
}

// NewProfile function returns a pointer to a new Profile object that
// passes validation.
func NewProfile(location crv1alpha1.Location) *crv1alpha1.Profile {
	return &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-profile-",
		},
		Location: location,
		Credential: crv1alpha1.Credential{
			Type: crv1alpha1.CredentialTypeKeyPair,
			KeyPair: &crv1alpha1.KeyPair{
				IDField:     "access_key_id",
				SecretField: "secret_access_key",
			},
		},
	}
}

// NewTestRestoreActionSet returns a pointer to a restore actionset, that is used in deferPhase blueprint
func NewTestRestoreActionSet(namespace, blueprintName, poName string, arts map[string]crv1alpha1.Artifact) *crv1alpha1.ActionSet {
	return &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-restore-actionset-",
			Namespace:    namespace,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Artifacts: arts,
					Blueprint: blueprintName,
					Name:      "restore",
					Object: crv1alpha1.ObjectReference{
						Kind:      "Deployment",
						Name:      poName,
						Namespace: namespace,
					},
					Profile: &crv1alpha1.ObjectReference{
						Kind:      consts.ProfileResourceName,
						Name:      TestProfileName,
						Namespace: namespace,
					},
				},
			},
		},
	}
}

// NewTestActionSet function returns a pointer to a new ActionSet test object
func NewTestActionSet(namespace, blueprintName, poKind, poName, poNamespace, version, action string) *crv1alpha1.ActionSet {
	return &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
			Namespace:    namespace,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Blueprint: blueprintName,
					Name:      action,
					Object: crv1alpha1.ObjectReference{
						Kind:      poKind,
						Name:      poName,
						Namespace: poNamespace,
					},
					Profile: &crv1alpha1.ObjectReference{
						Kind:      consts.ProfileResourceName,
						Name:      TestProfileName,
						Namespace: namespace,
					},
					PreferredVersion: version,
				},
			},
		},
	}
}

// NewTestConfigMap function returns a pointer to a new ConfigMap test object
func NewTestConfigMap() *v1.ConfigMap {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-configmap-",
		},
		Data: map[string]string{"myKey": "myValue"},
	}
	return cm
}

const actionName = "myAction"

// NewTestBlueprint function returns a pointer to a new Blueprint test object
func NewTestBlueprint(poKind string, phaseFuncs ...string) *crv1alpha1.Blueprint {
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			actionName: {
				Kind:   "StatefulSet",
				Phases: make([]crv1alpha1.BlueprintPhase, 0, len(phaseFuncs)),
			},
		},
	}
	for _, bpa := range bp.Actions {
		for i, pf := range phaseFuncs {
			bpa.Phases = append(bpa.Phases, crv1alpha1.BlueprintPhase{
				Name: fmt.Sprintf("myPhase-%d", i),
				Func: pf,
			})
		}
	}
	return bp
}

// ActionSetWithConfigMap function returns a pointer to a new ActionSet test object with CongigMap
func ActionSetWithConfigMap(as *crv1alpha1.ActionSet, name string) *crv1alpha1.ActionSet {
	as.Spec.Actions[0].ConfigMaps = map[string]crv1alpha1.ObjectReference{
		"myCM": {
			Name:      name,
			Namespace: as.GetNamespace(),
		},
	}
	return as
}

// BlueprintWithConfigMap function returns a pointer to a new Blueprint test object with CongigMap
func BlueprintWithConfigMap(bp *crv1alpha1.Blueprint) *crv1alpha1.Blueprint {
	cmArgs := map[string]interface{}{testBPArg: "{{ .ConfigMaps.myCM.Data.myKey  }}"}
	for i := range bp.Actions[actionName].Phases {
		bp.Actions[actionName].Phases[i].Args = cmArgs
	}
	return bp
}

func CreateSecret(cli kubernetes.Interface, namespace, name string, secrettype v1.SecretType, data map[string][]byte) (se *v1.Secret, err error) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
		},
		Data: data,
	}
	if secrettype != "" {
		secret.Type = secrettype
	}

	se, err = cli.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	// Since CLI doesnt return gvk of the object created, setting it manually
	if err == nil {
		se.APIVersion = "v1"
		se.Kind = "Secret"
	}
	return
}

func GetRepoPasswordSecretData(password string) map[string][]byte {
	return map[string][]byte{
		reposerver.RepoPasswordKey: []byte(password),
	}
}

func GetRepoServerAdminSecretData(username, password string) map[string][]byte {
	return map[string][]byte{
		reposerver.AdminUsernameKey: []byte(username),
		reposerver.AdminPasswordKey: []byte(password),
	}
}

func GetRepoServerUserAccessSecretData(hostname, password string) map[string][]byte {
	return map[string][]byte{
		hostname: []byte(password),
	}
}

func GetKopiaTLSSecretData() (map[string][]byte, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Test Organization"},
			Country:       []string{"Test Country"},
			Province:      []string{"Test Province"},
			Locality:      []string{"Test Locality"},
			StreetAddress: []string{"Test Street"},
			PostalCode:    []string{"123456"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		"tls.crt": caPEM.Bytes(),
		"tls.key": caPrivKeyPEM.Bytes(),
	}, nil
}

func GetDefaultS3StorageCreds(c *check.C) map[string][]byte {
	key := GetEnvOrSkip(c, S3CompliantAccessKeyIDEnv)
	val := GetEnvOrSkip(c, S3CompliantSecretAccessKeyEnv)

	return map[string][]byte{
		secrets.AWSAccessKeyID:     []byte(key),
		secrets.AWSSecretAccessKey: []byte(val),
	}
}

func GetDefaultS3CompliantStorageLocation() map[string][]byte {
	return map[string][]byte{
		reposerver.TypeKey:     []byte(crv1alpha1.LocationTypeS3Compliant),
		reposerver.BucketKey:   []byte(TestS3BucketName),
		reposerver.PrefixKey:   []byte(KopiaRepositoryPath),
		reposerver.RegionKey:   []byte(TestS3Region),
		reposerver.EndpointKey: []byte(os.Getenv(S3CompliantLocationEndpointEnv)),
	}
}

func CreateTestKopiaRepository(cli kubernetes.Interface, rs *v1alpha1.RepositoryServer, storageLocation map[string][]byte) error {
	contentCacheMB, metadataCacheMB := command.GetGeneralCacheSizeSettings()

	commandArgs := command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   KopiaRepositoryPassword,
			ConfigFilePath: command.DefaultConfigFilePath,
			LogDirectory:   command.DefaultLogDirectory,
		},
		CacheDirectory:  command.DefaultCacheDirectory,
		Hostname:        KopiaRepositoryServerHost,
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
		Username:        KopiaRepositoryUser,
		RepoPathPrefix:  KopiaRepositoryPath,
		Location:        storageLocation,
	}
	return repository.ConnectToOrCreateKopiaRepository(
		cli,
		rs.Namespace,
		rs.Status.ServerInfo.PodName,
		DefaultKopiaRepositoryServerContainer,
		commandArgs,
	)
}

func GetTestKopiaRepositoryServerCR(namespace string) crv1alpha1.RepositoryServer {
	repositoryServer := crv1alpha1.RepositoryServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-kopia-repo-server-",
			Namespace:    namespace,
		},
		Spec: crv1alpha1.RepositoryServerSpec{
			Storage: crv1alpha1.Storage{
				SecretRef: v1.SecretReference{
					Namespace: namespace,
				},
				CredentialSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
			},
			Repository: crv1alpha1.Repository{
				RootPath: KopiaRepositoryPath,
				Username: KopiaRepositoryUser,
				Hostname: KopiaRepositoryServerHost,
				PasswordSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
			},
			Server: crv1alpha1.Server{
				UserAccess: crv1alpha1.UserAccess{
					UserAccessSecretRef: v1.SecretReference{
						Namespace: namespace,
					},
					Username: KopiaRepositoryServerAccessUser,
				},
				AdminSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
				TLSSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
			},
		},
	}
	return repositoryServer
}

func GetFileStoreLocationSecretData(claimName string) map[string][]byte {
	return map[string][]byte{
		reposerver.TypeKey:      []byte(reposerver.LocTypeFilestore),
		reposerver.ClaimNameKey: []byte(claimName),
	}
}
