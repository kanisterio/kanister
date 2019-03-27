package function

import (
	"context"
	"fmt"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type DataSuite struct {
	cli       kubernetes.Interface
	crCli     versioned.Interface
	namespace string
}

var _ = Suite(&DataSuite{})

func (s *DataSuite) SetUpSuite(c *C) {
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
	ns.GenerateName = "kanister-datatest-"

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

func (s *DataSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func newRestoreDataBlueprint(pvc string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"restore": &crv1alpha1.BlueprintAction{
				Kind: param.StatefulSetKind,
				SecretNames: []string{
					"backupKey",
				},
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testRestore",
						Func: "RestoreData",
						Args: map[string]interface{}{
							RestoreDataNamespaceArg:            "{{ .StatefulSet.Namespace }}",
							RestoreDataImageArg:                "kanisterio/kanister-tools:0.18.0",
							RestoreDataBackupArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}",
							RestoreDataRestorePathArg:          "/mnt/data",
							RestoreDataBackupTagArg:            fmt.Sprintf("{{ .Options.%s }}", BackupDataOutputBackupTag),
							RestoreDataEncryptionKeyArg:        "{{ .Secrets.backupKey.Data.password | toString }}",
							RestoreDataVolsArg: map[string]string{
								pvc: "/mnt/data",
							},
						},
					},
				},
			},
		},
	}
}

func newBackupDataBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": &crv1alpha1.BlueprintAction{
				Kind: param.StatefulSetKind,
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testBackup",
						Func: "BackupData",
						Args: map[string]interface{}{
							BackupDataNamespaceArg:            "{{ .StatefulSet.Namespace }}",
							BackupDataPodArg:                  "{{ index .StatefulSet.Pods 0 }}",
							BackupDataContainerArg:            "{{ index .StatefulSet.Containers 0 0 }}",
							BackupDataIncludePathArg:          "/etc",
							BackupDataBackupArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}",
							BackupDataEncryptionKeyArg:        "{{ .Secrets.backupKey.Data.password | toString }}",
						},
					},
				},
			},
		},
	}
}

func (s *DataSuite) TestBackupRestoreData(c *C) {
	ctx := context.Background()
	ss, err := s.cli.AppsV1().StatefulSets(s.namespace).Create(testutil.NewTestStatefulSet())
	c.Assert(err, IsNil)
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
	c.Assert(err, IsNil)

	pvc := testutil.NewTestPVC()
	pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(pvc)
	c.Assert(err, IsNil)

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-datatest",
			Namespace: s.namespace,
		},
		Type: "Opaque",
		StringData: map[string]string{
			"password": "myPassword",
		},
	}
	secret, err = s.cli.CoreV1().Secrets(s.namespace).Create(secret)
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
		Secrets: map[string]crv1alpha1.ObjectReference{
			"backupKey": crv1alpha1.ObjectReference{
				Kind:      "Secret",
				Name:      secret.GetName(),
				Namespace: s.namespace,
			},
		},
	}

	tp, err := param.New(ctx, s.cli, s.crCli, as)
	c.Assert(err, IsNil)

	location := crv1alpha1.Location{
		Type:   crv1alpha1.LocationTypeS3Compliant,
		Bucket: testutil.GetEnvOrSkip(c, testutil.TestS3BucketName),
	}
	tp.Profile = testutil.ObjectStoreProfileOrSkip(c, objectstore.ProviderTypeS3, location)

	// Test backup
	bp := *newBackupDataBlueprint()
	out := runAction(c, bp, "backup", tp)
	c.Assert(out[BackupDataOutputBackupID].(string), Not(Equals), "")
	c.Assert(out[BackupDataOutputBackupTag].(string), Not(Equals), "")

	options := map[string]string{
		BackupDataOutputBackupID:  out[BackupDataOutputBackupID].(string),
		BackupDataOutputBackupTag: out[BackupDataOutputBackupTag].(string),
	}
	tp.Options = options

	// Test restore
	bp = *newRestoreDataBlueprint(pvc.GetName())
	_ = runAction(c, bp, "restore", tp)
}

func newCopyDataTestBlueprint() crv1alpha1.Blueprint {
	return crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"addfile": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "test1",
						Func: "PrepareData",
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: "{{ .PVC.Namespace }}",
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"touch",
								"/mnt/data1/foo.txt",
							},
							PrepareDataVolumes: map[string]string{"{{ .PVC.Name }}": "/mnt/data1"},
						},
					},
				},
			},
			"copy": &crv1alpha1.BlueprintAction{
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testCopy",
						Func: "CopyVolumeData",
						Args: map[string]interface{}{
							CopyVolumeDataNamespaceArg:      "{{ .PVC.Namespace }}",
							CopyVolumeDataVolumeArg:         "{{ .PVC.Name }}",
							CopyVolumeDataArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}/{{ .PVC.Namespace }}/{{ .PVC.Name }}",
						},
					},
				},
			},
			"restore": &crv1alpha1.BlueprintAction{
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testRestore",
						Func: "RestoreData",
						Args: map[string]interface{}{
							RestoreDataNamespaceArg:            "{{ .PVC.Namespace }}",
							RestoreDataImageArg:                "kanisterio/kanister-tools:0.18.0",
							RestoreDataBackupArtifactPrefixArg: fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupArtifactLocation),
							RestoreDataBackupTagArg:            fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupTag),
							RestoreDataVolsArg: map[string]string{
								"{{ .PVC.Name }}": fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupRoot),
							},
						},
					},
				},
			},
			"checkfile": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Func: "PrepareData",
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: "{{ .PVC.Namespace }}",
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"ls",
								"-l",
								"/mnt/datadir/foo.txt",
							},
							PrepareDataVolumes: map[string]string{"{{ .PVC.Name }}": "/mnt/datadir"},
						},
					},
				},
			},
		},
	}
}
func (s *DataSuite) TestCopyData(c *C) {
	pvcSpec := testutil.NewTestPVC()
	pvc, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(pvcSpec)
	c.Assert(err, IsNil)
	tp := s.initPVCTemplateParams(c, pvc, nil)
	bp := newCopyDataTestBlueprint()

	// Add a file on the source PVC
	_ = runAction(c, bp, "addfile", tp)
	// Copy PVC data
	out := runAction(c, bp, "copy", tp)

	// Validate outputs and setup as inputs for restore
	c.Assert(out[CopyVolumeDataOutputBackupID].(string), Not(Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupRoot].(string), Not(Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupArtifactLocation].(string), Not(Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupTag].(string), Not(Equals), "")
	options := map[string]string{
		CopyVolumeDataOutputBackupID:               out[CopyVolumeDataOutputBackupID].(string),
		CopyVolumeDataOutputBackupRoot:             out[CopyVolumeDataOutputBackupRoot].(string),
		CopyVolumeDataOutputBackupArtifactLocation: out[CopyVolumeDataOutputBackupArtifactLocation].(string),
		CopyVolumeDataOutputBackupTag:              out[CopyVolumeDataOutputBackupTag].(string),
	}

	// Create a new PVC
	pvc2, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(pvcSpec)
	c.Assert(err, IsNil)
	tp = s.initPVCTemplateParams(c, pvc2, options)
	// Restore data from copy
	_ = runAction(c, bp, "restore", tp)
	// Validate file exists on this new PVC
	_ = runAction(c, bp, "checkfile", tp)
}

func runAction(c *C, bp crv1alpha1.Blueprint, action string, tp *param.TemplateParams) map[string]interface{} {
	phases, err := kanister.GetPhases(bp, action, *tp)
	c.Assert(err, IsNil)
	out := make(map[string]interface{})
	for _, p := range phases {
		o, err := p.Exec(context.Background(), bp, action, *tp)
		c.Assert(err, IsNil)
		for k, v := range o {
			out[k] = v
		}
	}
	return out
}

func (s *DataSuite) initPVCTemplateParams(c *C, pvc *v1.PersistentVolumeClaim, options map[string]string) *param.TemplateParams {
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      param.PVCKind,
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
		},
		Profile: &crv1alpha1.ObjectReference{
			Name:      testutil.TestProfileName,
			Namespace: s.namespace,
		},
		Options: options,
	}
	tp, err := param.New(context.Background(), s.cli, s.crCli, as)
	c.Assert(err, IsNil)

	location := crv1alpha1.Location{
		Type:   crv1alpha1.LocationTypeS3Compliant,
		Bucket: testutil.GetEnvOrSkip(c, testutil.TestS3BucketName),
	}
	tp.Profile = testutil.ObjectStoreProfileOrSkip(c, objectstore.ProviderTypeS3, location)
	return tp
}
