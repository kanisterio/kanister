package function

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const (
	volumeSnapshotInfoKey = "volumeSnapshotInfo"
	manifestKey           = "manifest"
	backupInfoKey         = "backupInfo"
	skipTestErrorMsg      = "Storage type not supported"
	AWSRegion             = "AWS_REGION"
)

type VolumeSnapshotTestSuite struct {
	cli       kubernetes.Interface
	crCli     versioned.Interface
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

	// Make sure the CRD's exist.
	err = resource.CreateCustomResources(context.Background(), config)
	c.Assert(err, IsNil)

	s.cli = cli
	s.crCli = crCli

	ns := testutil.NewTestNamespace()

	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.GetName()

	ctx := context.Background()
	ss, err := s.cli.AppsV1().StatefulSets(s.namespace).Create(newStatefulSet(s.namespace))
	c.Assert(err, IsNil)
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
	c.Assert(err, IsNil)

	pods, _, err := kube.FetchPods(s.cli, s.namespace, ss.UID)
	c.Assert(err, IsNil)
	volToPvc := kube.StatefulSetVolumes(s.cli, ss, &pods[0])
	pvc, _ := volToPvc[pods[0].Spec.Containers[0].VolumeMounts[0].Name]
	c.Assert(len(pvc) > 0, Equals, true)
	id, secret, err := s.getCreds(c, s.cli, s.namespace, pvc)
	c.Assert(err, IsNil)
	if id == "" || secret == "" {
		c.Skip("Skipping the test since storage type not supported")
	}

	serviceKey, err := getServiceKey(c)
	c.Assert(err, IsNil)
	sec := NewTestProfileSecret(serviceKey, id, secret)
	sec, err = s.cli.Core().Secrets(s.namespace).Create(sec)
	c.Assert(err, IsNil)

	p := NewTestProfile(s.namespace, sec.GetName())
	_, err = s.crCli.CrV1alpha1().Profiles(s.namespace).Create(p)
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

	tp, err := param.New(ctx, s.cli, s.crCli, as)
	c.Assert(err, IsNil)
	s.tp = tp

}

// NewTestProfileSecret function returns a pointer to a new Secret test object.
func NewTestProfileSecret(serviceKey string, id string, secret string) *v1.Secret {
	return &v1.Secret{
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
func NewTestProfile(namespace string, secretName string) *crv1alpha1.Profile {
	return &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testutil.TestProfileName,
			Namespace: namespace,
		},
		Location: crv1alpha1.Location{
			Type:   crv1alpha1.LocationTypeS3Compliant,
			Region: os.Getenv(AWSRegion),
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
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
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
						Func: "CreateVolumeSnapshot",
						Args: map[string]interface{}{
							CreateVolumeSnapshotNamespaceArg: "{{ .StatefulSet.Namespace }}",
							CreateVolumeSnapshotSkipWaitArg:  true,
						},
					},
					{
						Name: "waitOnSnapshots",
						Func: "WaitForSnapshotCompletion",
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
						Func: "ScaleWorkload",
						Args: map[string]interface{}{
							ScaleWorkloadNamespaceArg: "{{ .StatefulSet.Namespace }}",
							ScaleWorkloadReplicas:     0,
						},
					},
					{
						Name: "testRestoreVolume",
						Func: "CreateVolumeFromSnapshot",
						Args: map[string]interface{}{
							CreateVolumeFromSnapshotNamespaceArg: "{{ .StatefulSet.Namespace }}",
							CreateVolumeFromSnapshotManifestArg:  "{{ .ArtifactsIn.backupInfo.KeyValue.manifest }}",
						},
					},
					{
						Name: "testBringupPod",
						Func: "ScaleWorkload",
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
						Func: "DeleteVolumeSnapshot",
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
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "kanister-test-pvc-snap-vol1",
									MountPath: "/var/lib/vol1",
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				v1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kanister-test-pvc-snap-vol1",
						Namespace: namespace,
					},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: *k8sresource.NewQuantity(1, k8sresource.BinarySI),
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
		phases, err := kanister.GetPhases(*bp, action, *s.tp)
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

func (s *VolumeSnapshotTestSuite) getCreds(c *C, cli kubernetes.Interface, namespace string, pvcname string) (string, string, error) {
	pvc, err := cli.Core().PersistentVolumeClaims(namespace).Get(pvcname, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	pvName := pvc.Spec.VolumeName
	pv, err := cli.Core().PersistentVolumes().Get(pvName, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	switch {
	case pv.Spec.AWSElasticBlockStore != nil:
		_ = GetEnvOrSkip(c, AWSRegion)
		return GetEnvOrSkip(c, awsebs.AccessKeyID), GetEnvOrSkip(c, awsebs.SecretAccessKey), nil

	case pv.Spec.GCEPersistentDisk != nil:
		serviceKey, err := getServiceKey(c)
		if err != nil {
			return "", "", err
		}
		return "test_project_id", serviceKey, nil
	}
	return "", "", nil
}

func getServiceKey(c *C) (string, error) {
	filename := GetEnvOrSkip(c, blockstorage.GoogleCloudCreds)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func GetEnvOrSkip(c *C, varName string) string {
	v := os.Getenv(varName)
	// Ensure the variable is set
	if v == "" {
		c.Skip("Required environment variable " + varName + " is not set")
	}
	return v
}
