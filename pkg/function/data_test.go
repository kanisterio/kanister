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
	"encoding/json"
	"fmt"
	"os"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"

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
	cli                  kubernetes.Interface
	crCli                versioned.Interface
	osCli                osversioned.Interface
	namespace            string
	profile              *param.Profile
	profileLocalEndpoint *param.Profile
	providerType         objectstore.ProviderType
}

const (
	testBucketName = "kio-store-tests"
)

var _ = check.Suite(&DataSuite{providerType: objectstore.ProviderTypeS3})
var _ = check.Suite(&DataSuite{providerType: objectstore.ProviderTypeGCS})

func (s *DataSuite) SetUpSuite(c *check.C) {
	config, err := kube.LoadConfig()
	c.Assert(err, check.IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, check.IsNil)
	crCli, err := versioned.NewForConfig(config)
	c.Assert(err, check.IsNil)
	osCli, err := osversioned.NewForConfig(config)
	c.Assert(err, check.IsNil)

	// Make sure the CRD's exist.
	err = resource.CreateCustomResources(context.Background(), config)
	c.Assert(err, check.IsNil)

	s.cli = cli
	s.crCli = crCli
	s.osCli = osCli

	ctx := context.Background()
	ns := testutil.NewTestNamespace()
	ns.GenerateName = "kanister-datatest-"

	cns, err := s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = cns.GetName()

	s.profile = s.createNewTestProfile(c, testutil.TestProfileName, false)
	if useMinio, ok := os.LookupEnv("USE_MINIO"); ok && useMinio == "true" {
		s.profileLocalEndpoint = s.createNewTestProfile(c, "test-profile-loc", true)
	}

	err = os.Setenv("POD_NAMESPACE", s.namespace)
	c.Assert(err, check.IsNil)
	err = os.Setenv("POD_SERVICE_ACCOUNT", "default")
	c.Assert(err, check.IsNil)
}

func (s *DataSuite) TearDownSuite(c *check.C) {
	ctx := context.Background()
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(ctx, s.namespace, metav1.DeleteOptions{})
	}
}

func newRestoreDataBlueprint(pvc, identifierArg, identifierVal string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"restore": {
				Kind: param.StatefulSetKind,
				SecretNames: []string{
					"backupKey",
				},
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testRestore",
						Func: RestoreDataFuncName,
						Args: map[string]interface{}{
							RestoreDataNamespaceArg:            "{{ .StatefulSet.Namespace }}",
							RestoreDataImageArg:                "ghcr.io/kanisterio/kanister-tools:0.114.0",
							RestoreDataBackupArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}",
							RestoreDataRestorePathArg:          "/mnt/data",
							RestoreDataEncryptionKeyArg:        "{{ .Secrets.backupKey.Data.password | toString }}",
							RestoreDataVolsArg: map[string]string{
								pvc: "/mnt/data",
							},
							identifierArg: fmt.Sprintf("{{ .Options.%s }}", identifierVal),
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
			"backup": {
				Kind: param.StatefulSetKind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testBackup",
						Func: BackupDataFuncName,
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

func newCheckRepositoryBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"checkRepository": {
				Kind: param.StatefulSetKind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testCheckRepository",
						Func: CheckRepositoryFuncName,
						Args: map[string]interface{}{
							CheckRepositoryArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}",
							CheckRepositoryEncryptionKeyArg:  "{{ .Secrets.backupKey.Data.password | toString }}",
						},
					},
				},
			},
		},
	}
}

func newLocationDeleteBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"delete": {
				Kind: param.StatefulSetKind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testLocationDelete",
						Func: LocationDeleteFuncName,
						Args: map[string]interface{}{
							LocationDeleteArtifactArg: "{{ .Profile.Location.Bucket }}",
						},
					},
				},
			},
		},
	}
}

func newBackupDataAllBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				Kind: param.StatefulSetKind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testBackupDataAll",
						Func: BackupDataAllFuncName,
						Args: map[string]interface{}{
							BackupDataAllNamespaceArg:            "{{ .StatefulSet.Namespace }}",
							BackupDataAllContainerArg:            "{{ index .StatefulSet.Containers 0 0 }}",
							BackupDataAllIncludePathArg:          "/etc",
							BackupDataAllBackupArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}",
						},
					},
				},
			},
		},
	}
}

func newRestoreDataAllBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"restore": {
				Kind: param.StatefulSetKind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testRestoreDataAll",
						Func: RestoreDataAllFuncName,
						Args: map[string]interface{}{
							RestoreDataAllNamespaceArg:            "{{ .StatefulSet.Namespace }}",
							RestoreDataAllImageArg:                "ghcr.io/kanisterio/kanister-tools:0.114.0",
							RestoreDataAllBackupArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}",
							RestoreDataAllBackupInfo:              fmt.Sprintf("{{ .Options.%s }}", BackupDataAllOutput),
							RestoreDataAllRestorePathArg:          "/mnt/data",
						},
					},
				},
			},
		},
	}
}

func newDeleteDataAllBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"delete": {
				Kind: param.StatefulSetKind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testDeleteDataAll",
						Func: DeleteDataAllFuncName,
						Args: map[string]interface{}{
							DeleteDataAllNamespaceArg:            "{{ .StatefulSet.Namespace }}",
							DeleteDataAllBackupArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}",
							DeleteDataAllBackupInfo:              fmt.Sprintf("{{ .Options.%s }}", BackupDataAllOutput),
							DeleteDataAllReclaimSpace:            true,
						},
					},
				},
			},
		},
	}
}

func (s *DataSuite) getTemplateParamsAndPVCName(c *check.C, replicas int32) (*param.TemplateParams, []string) {
	ctx := context.Background()
	ss, err := s.cli.AppsV1().StatefulSets(s.namespace).Create(context.TODO(), testutil.NewTestStatefulSet(replicas), metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
	c.Assert(err, check.IsNil)
	pvcs := []string{}
	var i int32
	for i = 0; i < replicas; i++ {
		pvc := testutil.NewTestPVC()
		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
		c.Assert(err, check.IsNil)
		pvcs = append(pvcs, pvc.GetName())
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "secret-datatest-",
			Namespace:    s.namespace,
		},
		Type: "Opaque",
		StringData: map[string]string{
			"password": "myPassword",
		},
	}
	secret, err = s.cli.CoreV1().Secrets(s.namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

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
			"backupKey": {
				Kind:      "Secret",
				Name:      secret.GetName(),
				Namespace: s.namespace,
			},
		},
	}

	tp, err := param.New(ctx, s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, ss), s.crCli, s.osCli, as)
	c.Assert(err, check.IsNil)
	tp.Profile = s.profile

	return tp, pvcs
}

func (s *DataSuite) TestBackupRestoreDeleteData(c *check.C) {
	tp, pvcs := s.getTemplateParamsAndPVCName(c, 1)

	for _, pvc := range pvcs {
		// Test backup
		bp := *newBackupDataBlueprint()
		out := runAction(c, bp, "backup", tp)
		c.Assert(out[BackupDataOutputBackupID].(string), check.Not(check.Equals), "")
		c.Assert(out[BackupDataOutputBackupTag].(string), check.Not(check.Equals), "")
		c.Check(out[BackupDataStatsOutputFileCount].(string), check.Not(check.Equals), "")
		c.Check(out[BackupDataStatsOutputSize].(string), check.Not(check.Equals), "")
		c.Assert(out[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)

		options := map[string]string{
			BackupDataOutputBackupID:  out[BackupDataOutputBackupID].(string),
			BackupDataOutputBackupTag: out[BackupDataOutputBackupTag].(string),
		}
		tp.Options = options

		// Test restore
		bp = *newRestoreDataBlueprint(pvc, RestoreDataBackupTagArg, BackupDataOutputBackupTag)
		_ = runAction(c, bp, "restore", tp)

		if useMinio, ok := os.LookupEnv("USE_MINIO"); ok && useMinio == "true" {
			tp.Profile = s.profileLocalEndpoint
		}
		bp = *newLocationDeleteBlueprint()
		_ = runAction(c, bp, "delete", tp)
	}
}

func (s *DataSuite) TestBackupRestoreDataWithSnapshotID(c *check.C) {
	tp, pvcs := s.getTemplateParamsAndPVCName(c, 1)
	for _, pvc := range pvcs {
		// Test backup
		bp := *newBackupDataBlueprint()
		out := runAction(c, bp, "backup", tp)
		c.Assert(out[BackupDataOutputBackupID].(string), check.Not(check.Equals), "")
		c.Assert(out[BackupDataOutputBackupTag].(string), check.Not(check.Equals), "")
		c.Check(out[BackupDataStatsOutputFileCount].(string), check.Not(check.Equals), "")
		c.Check(out[BackupDataStatsOutputSize].(string), check.Not(check.Equals), "")
		c.Assert(out[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)

		options := map[string]string{
			BackupDataOutputBackupID:  out[BackupDataOutputBackupID].(string),
			BackupDataOutputBackupTag: out[BackupDataOutputBackupTag].(string),
		}
		tp.Options = options

		// Test restore with ID
		bp = *newRestoreDataBlueprint(pvc, RestoreDataBackupIdentifierArg, BackupDataOutputBackupID)
		_ = runAction(c, bp, "restore", tp)
	}
}

func (s *DataSuite) TestBackupRestoreDeleteDataAll(c *check.C) {
	replicas := int32(2)
	tp, pvcs := s.getTemplateParamsAndPVCName(c, replicas)

	// Test backup
	bp := *newBackupDataAllBlueprint()
	out := runAction(c, bp, "backup", tp)
	c.Assert(out[BackupDataAllOutput].(string), check.Not(check.Equals), "")
	c.Assert(out[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)

	output := make(map[string]BackupInfo)
	c.Assert(json.Unmarshal([]byte(out[BackupDataAllOutput].(string)), &output), check.IsNil)
	c.Assert(int32(len(output)), check.Equals, replicas)
	for k := range output {
		c.Assert(k, check.Equals, output[k].PodName)
	}
	options := map[string]string{BackupDataAllOutput: out[BackupDataAllOutput].(string)}
	tp.Options = options

	for i, pod := range tp.StatefulSet.Pods {
		tp.StatefulSet.PersistentVolumeClaims[pod] = map[string]string{pvcs[i]: "/mnt/data"}
	}
	// Test restore
	bp = *newRestoreDataAllBlueprint()
	_ = runAction(c, bp, "restore", tp)

	// Test delete
	bp = *newDeleteDataAllBlueprint()
	_ = runAction(c, bp, "delete", tp)
}

func newCopyDataTestBlueprint() crv1alpha1.Blueprint {
	return crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"addfile": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "test1",
						Func: PrepareDataFuncName,
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
			"copy": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testCopy",
						Func: CopyVolumeDataFuncName,
						Args: map[string]interface{}{
							CopyVolumeDataNamespaceArg:      "{{ .PVC.Namespace }}",
							CopyVolumeDataVolumeArg:         "{{ .PVC.Name }}",
							CopyVolumeDataArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}/{{ .PVC.Namespace }}/{{ .PVC.Name }}",
						},
					},
				},
			},
			"restore": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testRestore",
						Func: RestoreDataFuncName,
						Args: map[string]interface{}{
							RestoreDataNamespaceArg:            "{{ .PVC.Namespace }}",
							RestoreDataImageArg:                "ghcr.io/kanisterio/kanister-tools:0.114.0",
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
						Func: PrepareDataFuncName,
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: "{{ .PVC.Namespace }}",
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"sh", "-c",
								fmt.Sprintf("ls -la {{ .Options.%s }} && cat {{ .Options.%s }}/foo.txt", CopyVolumeDataOutputBackupRoot, CopyVolumeDataOutputBackupRoot),
							},
							PrepareDataVolumes:        map[string]string{"{{ .PVC.Name }}": fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupRoot)},
							PrepareDataFailOnErrorArg: true,
						},
					},
				},
			},
			"delete": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testDelete",
						Func: DeleteDataFuncName,
						Args: map[string]interface{}{
							DeleteDataNamespaceArg:            "{{ .PVC.Namespace }}",
							DeleteDataBackupArtifactPrefixArg: fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupArtifactLocation),
							DeleteDataBackupIdentifierArg:     fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupID),
						},
					},
				},
			},
		},
	}
}
func (s *DataSuite) TestCopyData(c *check.C) {
	pvcSpec := testutil.NewTestPVC()
	pvc, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvcSpec, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	tp := s.initPVCTemplateParams(c, pvc, nil)
	bp := newCopyDataTestBlueprint()

	// Add a file on the source PVC
	_ = runAction(c, bp, "addfile", tp)
	// Copy PVC data
	out := runAction(c, bp, "copy", tp)

	// Validate outputs and setup as inputs for restore
	c.Assert(out[CopyVolumeDataOutputBackupID].(string), check.Not(check.Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupRoot].(string), check.Not(check.Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupArtifactLocation].(string), check.Not(check.Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupTag].(string), check.Not(check.Equals), "")
	c.Assert(out[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)
	options := map[string]string{
		CopyVolumeDataOutputBackupID:               out[CopyVolumeDataOutputBackupID].(string),
		CopyVolumeDataOutputBackupRoot:             out[CopyVolumeDataOutputBackupRoot].(string),
		CopyVolumeDataOutputBackupArtifactLocation: out[CopyVolumeDataOutputBackupArtifactLocation].(string),
		CopyVolumeDataOutputBackupTag:              out[CopyVolumeDataOutputBackupTag].(string),
	}

	// Create a new PVC
	pvc2, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvcSpec, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	tp = s.initPVCTemplateParams(c, pvc2, options)
	// Restore data from copy
	_ = runAction(c, bp, "restore", tp)
	// Validate file exists on this new PVC
	_ = runAction(c, bp, "checkfile", tp)
	// Delete data from copy
	_ = runAction(c, bp, "delete", tp)
}

func newCopyDataDifferentPathsTestBlueprint() crv1alpha1.Blueprint {
	return crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"addfile": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "test1",
						Func: PrepareDataFuncName,
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: "{{ .PVC.Namespace }}",
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"sh", "-c",
								"mkdir -p /mnt/source_data/subdir && echo 'test content' > /mnt/source_data/subdir/test.txt",
							},
							PrepareDataVolumes: map[string]string{"{{ .PVC.Name }}": "/mnt/source_data"},
						},
					},
				},
			},
			"copy": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testCopy",
						Func: CopyVolumeDataFuncName,
						Args: map[string]interface{}{
							CopyVolumeDataNamespaceArg:      "{{ .PVC.Namespace }}",
							CopyVolumeDataVolumeArg:         "{{ .PVC.Name }}",
							CopyVolumeDataArtifactPrefixArg: "{{ .Profile.Location.Bucket }}/{{ .Profile.Location.Prefix }}/{{ .PVC.Namespace }}/{{ .PVC.Name }}",
							CopyVolumeDataMountPathArg:      "/mnt/source_data",
						},
					},
				},
			},
			"restore": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testRestore",
						Func: RestoreDataFuncName,
						Args: map[string]interface{}{
							RestoreDataNamespaceArg:            "{{ .PVC.Namespace }}",
							RestoreDataImageArg:                "ghcr.io/kanisterio/kanister-tools:0.113.0",
							RestoreDataBackupArtifactPrefixArg: fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupArtifactLocation),
							RestoreDataBackupTagArg:            fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupTag),
							RestoreDataBackupPathArg:           fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupRoot),
							RestoreDataVolsArg: map[string]string{
								"{{ .PVC.Name }}": "/mnt/target_data",
							},
							RestoreDataRestorePathArg: "/mnt/target_data",
						},
					},
				},
			},
			"checkfile": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Func: PrepareDataFuncName,
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: "{{ .PVC.Namespace }}",
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"sh", "-c",
								"ls -la /mnt/target_data/ && cat /mnt/target_data/subdir/test.txt",
							},
							PrepareDataVolumes: map[string]string{"{{ .PVC.Name }}": "/mnt/target_data"},
						},
					},
				},
			},
			"delete": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testDelete",
						Func: DeleteDataFuncName,
						Args: map[string]interface{}{
							DeleteDataNamespaceArg:            "{{ .PVC.Namespace }}",
							DeleteDataBackupArtifactPrefixArg: fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupArtifactLocation),
							DeleteDataBackupIdentifierArg:     fmt.Sprintf("{{ .Options.%s }}", CopyVolumeDataOutputBackupID),
						},
					},
				},
			},
		},
	}
}

func (s *DataSuite) TestCopyDataDifferentPaths(c *check.C) {
	pvcSpec := testutil.NewTestPVC()
	pvc, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvcSpec, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	tp := s.initPVCTemplateParams(c, pvc, nil)
	bp := newCopyDataDifferentPathsTestBlueprint()

	// Add a file with subdirectory on the source PVC at /mnt/source_data
	_ = runAction(c, bp, "addfile", tp)
	// Copy PVC data
	out := runAction(c, bp, "copy", tp)

	// Validate outputs and setup as inputs for restore
	c.Assert(out[CopyVolumeDataOutputBackupID].(string), check.Not(check.Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupRoot].(string), check.Not(check.Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupArtifactLocation].(string), check.Not(check.Equals), "")
	c.Assert(out[CopyVolumeDataOutputBackupTag].(string), check.Not(check.Equals), "")
	c.Assert(out[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)
	options := map[string]string{
		CopyVolumeDataOutputBackupID:               out[CopyVolumeDataOutputBackupID].(string),
		CopyVolumeDataOutputBackupRoot:             out[CopyVolumeDataOutputBackupRoot].(string),
		CopyVolumeDataOutputBackupArtifactLocation: out[CopyVolumeDataOutputBackupArtifactLocation].(string),
		CopyVolumeDataOutputBackupTag:              out[CopyVolumeDataOutputBackupTag].(string),
	}

	// Create a new PVC for restoration
	pvc2, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(context.TODO(), pvcSpec, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	tp = s.initPVCTemplateParams(c, pvc2, options)
	// Restore data from copy to different path /mnt/target_data
	_ = runAction(c, bp, "restore", tp)
	// Validate file exists at the correct location in target path
	// This proves the fix works - files restore to /mnt/target_data/subdir/test.txt
	// instead of /mnt/target_data/mnt/vol_data/pvcname/subdir/test.txt (which would be wrong)
	_ = runAction(c, bp, "checkfile", tp)
	// Delete data from copy
	_ = runAction(c, bp, "delete", tp)
}

func runAction(c *check.C, bp crv1alpha1.Blueprint, action string, tp *param.TemplateParams) map[string]interface{} {
	phases, err := kanister.GetPhases(bp, action, kanister.DefaultVersion, *tp)
	c.Assert(err, check.IsNil)
	out := make(map[string]interface{})
	for _, p := range phases {
		o, err := p.Exec(context.Background(), bp, action, *tp)
		c.Assert(err, check.IsNil)
		for k, v := range o {
			out[k] = v
		}
	}
	return out
}

func (s *DataSuite) initPVCTemplateParams(c *check.C, pvc *corev1.PersistentVolumeClaim, options map[string]string) *param.TemplateParams {
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
	tp, err := param.New(context.Background(), s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, pvc), s.crCli, s.osCli, as)
	c.Assert(err, check.IsNil)
	tp.Profile = s.profile
	return tp
}

func (s *DataSuite) TestCheckRepository(c *check.C) {
	tp, _ := s.getTemplateParamsAndPVCName(c, 1)

	// Test backup
	bp := *newBackupDataBlueprint()
	out := runAction(c, bp, "backup", tp)
	c.Assert(out[BackupDataOutputBackupID].(string), check.Not(check.Equals), "")
	c.Assert(out[BackupDataOutputBackupTag].(string), check.Not(check.Equals), "")
	c.Assert(out[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)

	// Test CheckRepository
	bp2 := *newCheckRepositoryBlueprint()
	out2 := runAction(c, bp2, "checkRepository", tp)
	c.Assert(out2[CheckRepositoryPasswordIncorrect].(string), check.Equals, "false")
	c.Assert(out2[CheckRepositoryRepoDoesNotExist].(string), check.Equals, "false")
	c.Assert(out2[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)
}

func (s *DataSuite) TestCheckRepositoryWrongPassword(c *check.C) {
	tp, _ := s.getTemplateParamsAndPVCName(c, 1)

	// Test backup
	bp := *newBackupDataBlueprint()
	bp.Actions["backup"].Phases[0].Args[BackupDataBackupArtifactPrefixArg] = fmt.Sprintf("%s/%s", bp.Actions["backup"].Phases[0].Args[BackupDataBackupArtifactPrefixArg], "abcdef")
	bp.Actions["backup"].Phases[0].Args[BackupDataEncryptionKeyArg] = "foobar"
	out := runAction(c, bp, "backup", tp)
	c.Assert(out[BackupDataOutputBackupID].(string), check.Not(check.Equals), "")
	c.Assert(out[BackupDataOutputBackupTag].(string), check.Not(check.Equals), "")
	c.Assert(out[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)

	// Test CheckRepository
	bp2 := *newCheckRepositoryBlueprint()
	bp2.Actions["checkRepository"].Phases[0].Args[CheckRepositoryArtifactPrefixArg] = fmt.Sprintf("%s/%s", bp2.Actions["checkRepository"].Phases[0].Args[CheckRepositoryArtifactPrefixArg], "abcdef")
	out2 := runAction(c, bp2, "checkRepository", tp)
	c.Assert(out2[CheckRepositoryPasswordIncorrect].(string), check.Equals, "true")
}

func (s *DataSuite) TestCheckRepositoryRepoNotAvailable(c *check.C) {
	tp, _ := s.getTemplateParamsAndPVCName(c, 1)

	// Test backup
	bp := *newBackupDataBlueprint()
	out := runAction(c, bp, "backup", tp)
	c.Assert(out[BackupDataOutputBackupID].(string), check.Not(check.Equals), "")
	c.Assert(out[BackupDataOutputBackupTag].(string), check.Not(check.Equals), "")
	c.Assert(out[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)

	// Test CheckRepository
	bp2 := *newCheckRepositoryBlueprint()
	bp2.Actions["checkRepository"].Phases[0].Args[CheckRepositoryArtifactPrefixArg] = fmt.Sprintf("%s/%s", bp2.Actions["checkRepository"].Phases[0].Args[CheckRepositoryArtifactPrefixArg], c.TestName())
	out2 := runAction(c, bp2, "checkRepository", tp)
	c.Assert(out2[CheckRepositoryRepoDoesNotExist].(string), check.Equals, "true")
	c.Assert(out2[FunctionOutputVersion].(string), check.Equals, kanister.DefaultVersion)
}

func (s *DataSuite) createNewTestProfile(c *check.C, profileName string, localEndpoint bool) *param.Profile {
	var err error
	ctx := context.Background()

	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, sec, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	p.Name = profileName
	_, err = s.crCli.CrV1alpha1().Profiles(s.namespace).Create(ctx, p, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	var location crv1alpha1.Location
	switch s.providerType {
	case objectstore.ProviderTypeS3:
		location = crv1alpha1.Location{
			Type: crv1alpha1.LocationTypeS3Compliant,
		}
	case objectstore.ProviderTypeGCS:
		location = crv1alpha1.Location{
			Type: crv1alpha1.LocationTypeGCS,
		}
	default:
		c.Fatalf("Unrecognized objectstore '%s'", s.providerType)
	}
	location.Prefix = "testBackupRestoreLocDelete"
	location.Bucket = testBucketName
	if endpoint, ok := os.LookupEnv("LOCATION_CLUSTER_ENDPOINT"); ok && !localEndpoint {
		location.Endpoint = endpoint
	}
	profile := testutil.ObjectStoreProfileOrSkip(c, s.providerType, location)
	return profile
}
