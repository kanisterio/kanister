package function

import (
	"context"

	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
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

const actionName = "backupAndRestore"

func newRestoreDataBlueprint(pvc string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			actionName: &crv1alpha1.BlueprintAction{
				Kind: param.StatefulSetKind,
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testRestore",
						Func: "RestoreData",
						Args: map[string]interface{}{
							RestoreDataNamespaceArg:            "{{ .StatefulSet.Namespace }}",
							RestoreDataImageArg:                "kanisterio/kanister-tools:0.12.0",
							RestoreDataBackupArtifactPrefixArg: "{{ .Profile.Location.S3Compliant.Bucket }}/{{ .Profile.Location.S3Compliant.Prefix }}",
							RestoreDataRestorePathArg:          "/",
							RestoreDataBackupIdentifierArg:     "{{ .Time }}",
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
			actionName: &crv1alpha1.BlueprintAction{
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
							BackupDataBackupArtifactPrefixArg: "{{ .Profile.Location.S3Compliant.Bucket }}/{{ .Profile.Location.S3Compliant.Prefix }}",
							BackupDataBackupIdentifierArg:     "{{ .Time }}",
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

	tp.Profile = testutil.ObjectStoreProfileOrSkip(c)

	for _, bp := range []crv1alpha1.Blueprint{
		*newBackupDataBlueprint(),
		*newRestoreDataBlueprint(pvc.GetName()),
	} {
		phases, err := kanister.GetPhases(bp, actionName, *tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			err := p.Exec(context.Background(), *tp)
			c.Assert(err, IsNil)
		}
	}
}
