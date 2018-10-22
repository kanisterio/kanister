package function

import (
	"context"

	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const volSnapshotInfoKey = "volumeSnapshotInfo"

type VolumeSnapshotTestSuite struct {
	cli       kubernetes.Interface
	crCli     versioned.Interface
	namespace string
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

	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.Core().Secrets(s.namespace).Create(sec)
	c.Assert(err, IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	_, err = s.crCli.CrV1alpha1().Profiles(s.namespace).Create(p)
	c.Assert(err, IsNil)
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

	ss, err := s.cli.AppsV1().StatefulSets(s.namespace).Create(newStatefulSet(s.namespace))
	c.Assert(err, IsNil)
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
	c.Assert(err, IsNil)

	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      param.StatefulSetKind,
			Name:      ss.GetName(),
			Namespace: s.namespace,
		},
		Profile: &crv1alpha1.ObjectReference{
			Name:      testutil.TestProfileName,
			Namespace: s.namespace,
		},
	}

	tp, err := param.New(ctx, s.cli, s.crCli, as)
	c.Assert(err, IsNil)

	bp := newVolumeSnapshotBlueprint()
	action := "backup"
	phases, err := kanister.GetPhases(*bp, action, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		output, err := p.Exec(ctx, *bp, action, *tp)
		c.Assert(err, IsNil)
		c.Assert(output, NotNil)
		c.Assert(output[volSnapshotInfoKey], NotNil)
		keyval := make(map[string]string)
		keyval["manifest"] = output[volSnapshotInfoKey].(string)
		artifact := crv1alpha1.Artifact{
			KeyValue: keyval,
		}
		tp.ArtifactsIn = make(map[string]crv1alpha1.Artifact)
		tp.ArtifactsIn["backupInfo"] = artifact
	}

	action = "restore"
	phases, err = kanister.GetPhases(*bp, action, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		_, err = p.Exec(ctx, *bp, action, *tp)
		c.Assert(err, IsNil)
	}

	action = "delete"
	phases, err = kanister.GetPhases(*bp, action, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		_, err = p.Exec(ctx, *bp, action, *tp)
		c.Assert(err, IsNil)
	}
}
