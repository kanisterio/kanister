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

package function

import (
	"context"
	"os"
	"strings"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const (
	skipTestErrorMsg = "Storage type not supported"
	AWSRegion        = "AWS_REGION"
)

type VolumeSnapshotTestSuite struct {
	cli       kubernetes.Interface
	crCli     versioned.Interface
	osCli     osversioned.Interface
	namespace string
	tp        *param.TemplateParams
}

var _ = Suite(&VolumeSnapshotTestSuite{})

func (s *VolumeSnapshotTestSuite) SetUpTest(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := versioned.NewForConfig(config)
	c.Assert(err, IsNil)
	osCli, err := osversioned.NewForConfig(config)
	c.Assert(err, IsNil)

	// Make sure the CRD's exist.
	err = resource.CreateCustomResources(context.Background(), config)
	c.Assert(err, IsNil)

	s.cli = cli
	s.crCli = crCli
	s.osCli = osCli

	ns := testutil.NewTestNamespace()

	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.GetName()

	ctx := context.Background()
	ss, err := s.cli.AppsV1().StatefulSets(s.namespace).Create(ctx, newStatefulSet(s.namespace), metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
	c.Assert(err, IsNil)

	pods, _, err := kube.FetchPods(s.cli, s.namespace, ss.UID)
	c.Assert(err, IsNil)
	volToPvc := kube.StatefulSetVolumes(s.cli, ss, &pods[0])
	pvc := volToPvc[pods[0].Spec.Containers[0].VolumeMounts[0].Name]
	c.Assert(len(pvc) > 0, Equals, true)
	id, secret, locationType, err := s.getCreds(c, ctx, s.cli, s.namespace, pvc)
	c.Assert(err, IsNil)
	if id == "" || secret == "" {
		c.Skip("Skipping the test since storage type not supported")
	}

	sec := NewTestProfileSecret(id, secret)
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, sec, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	p := NewTestProfile(s.namespace, sec.GetName(), locationType)
	_, err = s.crCli.CrV1alpha1().Profiles(s.namespace).Create(ctx, p, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      param.StatefulSetKind,
			Name:      ss.GetName(),
			Namespace: s.namespace,
		},
		Profile: &crv1alpha1.ObjectReference{
			Name:      p.GetName(),
			Namespace: s.namespace,
		},
	}

	tp, err := param.New(ctx, s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, ss), s.crCli, s.osCli, as)
	c.Assert(err, IsNil)
	s.tp = tp
}

// NewTestProfileSecret function returns a pointer to a new Secret test object.
func NewTestProfileSecret(id string, secret string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: map[string]string{
			"id":     id,
			"secret": secret,
		},
	}
}

// NewTestProfile function returns a pointer to a new Profile test object that
// passes validation.
func NewTestProfile(namespace string, secretName string, locationType crv1alpha1.LocationType) *crv1alpha1.Profile {
	region := ""
	if locationType == crv1alpha1.LocationTypeS3Compliant {
		region = os.Getenv(AWSRegion)
	}
	return &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testutil.TestProfileName,
			Namespace: namespace,
		},
		Location: crv1alpha1.Location{
			Type:   locationType,
			Region: region,
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

func (s *VolumeSnapshotTestSuite) TearDownTest(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func newVolumeSnapshotBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				Kind: param.StatefulSetKind,
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"backupInfo": {
						KeyValue: map[string]string{
							"manifest": "{{ .Phases.testBackupVolume.Output.volumeSnapshotInfo }}",
						},
					},
				},
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testBackupVolume",
						Func: CreateVolumeSnapshotFuncName,
						Args: map[string]interface{}{
							CreateVolumeSnapshotNamespaceArg: "{{ .StatefulSet.Namespace }}",
							CreateVolumeSnapshotSkipWaitArg:  true,
						},
					},
					{
						Name: "waitOnSnapshots",
						Func: WaitForSnapshotCompletionFuncName,
						Args: map[string]interface{}{
							WaitForSnapshotCompletionSnapshotsArg: "{{ .Phases.testBackupVolume.Output.volumeSnapshotInfo }}",
						},
					},
				},
			},
			"restore": {
				Kind: param.StatefulSetKind,
				InputArtifactNames: []string{
					"backupInfo",
				},
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testShutdownPod",
						Func: ScaleWorkloadFuncName,
						Args: map[string]interface{}{
							ScaleWorkloadNamespaceArg: "{{ .StatefulSet.Namespace }}",
							ScaleWorkloadReplicas:     0,
						},
					},
					{
						Name: "testRestoreVolume",
						Func: CreateVolumeFromSnapshotFuncName,
						Args: map[string]interface{}{
							CreateVolumeFromSnapshotNamespaceArg: "{{ .StatefulSet.Namespace }}",
							CreateVolumeFromSnapshotManifestArg:  "{{ .ArtifactsIn.backupInfo.KeyValue.manifest }}",
						},
					},
					{
						Name: "testBringupPod",
						Func: ScaleWorkloadFuncName,
						Args: map[string]interface{}{
							ScaleWorkloadNamespaceArg: "{{ .StatefulSet.Namespace }}",
							ScaleWorkloadReplicas:     1,
						},
					},
				},
			},
			"delete": {
				Kind: param.StatefulSetKind,
				InputArtifactNames: []string{
					"backupInfo",
				},
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "deleteVolumeSnapshot",
						Func: DeleteVolumeSnapshotFuncName,
						Args: map[string]interface{}{
							DeleteVolumeSnapshotNamespaceArg: "{{ .StatefulSet.Namespace }}",
							DeleteVolumeSnapshotManifestArg:  "{{ .ArtifactsIn.backupInfo.KeyValue.manifest }}",
						},
					},
				},
			},
		},
	}
}

func newStatefulSet(namespace string) *appsv1.StatefulSet {
	replicas := int32(1)
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.18.0",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kanister-test-pvc-snap-vol1",
									MountPath: "/var/lib/vol1",
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kanister-test-pvc-snap-vol1",
						Namespace: namespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceStorage): k8sresource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		},
	}
}

func (s *VolumeSnapshotTestSuite) TestVolumeSnapshot(c *C) {
	ctx := context.Background()
	actions := []string{"backup", "restore", "delete"}
	bp := newVolumeSnapshotBlueprint()
	for _, action := range actions {
		phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, *s.tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			c.Assert(param.InitPhaseParams(ctx, s.cli, s.tp, p.Name(), p.Objects()), IsNil)
			output, err := p.Exec(ctx, *bp, action, *s.tp)
			if err != nil && strings.Contains(err.Error(), skipTestErrorMsg) {
				c.Skip("Skipping the test since storage type not supported")
			}
			c.Assert(err, IsNil)
			param.UpdatePhaseParams(ctx, s.tp, p.Name(), output)
			if action == "backup" {
				arts, err := param.RenderArtifacts(bp.Actions[action].OutputArtifacts, *s.tp)
				c.Assert(err, IsNil)
				s.tp.ArtifactsIn = arts
			}
		}
	}
}

func (s *VolumeSnapshotTestSuite) getCreds(c *C, ctx context.Context, cli kubernetes.Interface, namespace string, pvcname string) (string, string, crv1alpha1.LocationType, error) {
	pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcname, metav1.GetOptions{})
	if err != nil {
		return "", "", "", err
	}
	pvName := pvc.Spec.VolumeName
	pv, err := cli.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return "", "", "", err
	}
	switch {
	case pv.Spec.AWSElasticBlockStore != nil:
		_ = GetEnvOrSkip(c, AWSRegion)
		return GetEnvOrSkip(c, awsconfig.AccessKeyID), GetEnvOrSkip(c, awsconfig.SecretAccessKey), crv1alpha1.LocationTypeS3Compliant, nil

	case pv.Spec.GCEPersistentDisk != nil:
		_ = GetEnvOrSkip(c, blockstorage.GoogleCloudCreds)
		creds, err := google.FindDefaultCredentials(ctx, compute.ComputeScope)
		if err != nil {
			return "", "", "", err
		}
		return creds.ProjectID, string(creds.JSON), crv1alpha1.LocationTypeGCS, nil
	}
	return "", "", "", nil
}

func GetEnvOrSkip(c *C, varName string) string {
	v := os.Getenv(varName)
	// Ensure the variable is set
	if v == "" {
		c.Skip("Required environment variable " + varName + " is not set")
	}
	return v
}
