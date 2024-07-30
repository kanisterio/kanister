// Copyright 2022 The Kanister Authors.
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

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/kube/snapshot"
)

const (
	// testRestoreNamespace is the namespace where testing is done
	testRestoreNamespace = "test-restore-csi-snapshot"
	// originalPVCName is the name of the PVC that will be captured
	originalPVCName = "test-pvc"
	// newPVCName is the name of the new PVC that will be restored
	newPVCName = "test-pvc-restored"
)

type RestoreCSISnapshotTestSuite struct {
	snapName            string
	pvcName             string
	newPVCName          string
	namespace           string
	volumeSnapshotClass string
	storageClass        string
}

var _ = Suite(&RestoreCSISnapshotTestSuite{})

func (testSuite *RestoreCSISnapshotTestSuite) SetUpSuite(c *C) {
	testSuite.volumeSnapshotClass = snapshotClass
	testSuite.storageClass = storageClass
	testSuite.pvcName = originalPVCName
	testSuite.newPVCName = newPVCName
	testSuite.snapName = snapshotName
	testSuite.namespace = testRestoreNamespace
}

func (testSuite *RestoreCSISnapshotTestSuite) TestRestoreCSISnapshot(c *C) {
	for _, apiResourceList := range []*metav1.APIResourceList{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VolumeSnapshot",
				APIVersion: "v1alpha1",
			},
			GroupVersion: "snapshot.storage.k8s.io/v1alpha1",
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VolumeSnapshot",
				APIVersion: "v1beta1",
			},
			GroupVersion: "snapshot.storage.k8s.io/v1beta1",
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VolumeSnapshot",
				APIVersion: "v1",
			},
			GroupVersion: "snapshot.storage.k8s.io/v1",
		},
	} {
		ctx := context.Background()
		fakeCli := fake.NewSimpleClientset()
		fakeCli.Resources = []*metav1.APIResourceList{apiResourceList}

		_, err := fakeCli.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testSuite.namespace}}, metav1.CreateOptions{})
		c.Assert(err, IsNil)

		scheme := runtime.NewScheme()
		fakeSnapshotter, err := snapshot.NewSnapshotter(fakeCli, dynfake.NewSimpleDynamicClient(scheme))
		c.Assert(err, IsNil)

		originalPVC := getOriginalPVCManifest(testSuite.pvcName, testSuite.storageClass)
		createPVC(c, testSuite.namespace, originalPVC, fakeCli)
		fakeSnapshotMeta := snapshot.ObjectMeta{
			Name:      testSuite.snapName,
			Namespace: testSuite.namespace,
		}
		err = fakeSnapshotter.Create(ctx, testSuite.pvcName, &testSuite.volumeSnapshotClass, false, fakeSnapshotMeta)
		c.Assert(err, IsNil)

		vs, err := fakeSnapshotter.Get(ctx, testSuite.snapName, testSuite.namespace)
		c.Assert(err, IsNil)
		c.Assert(vs.Name, Equals, testSuite.snapName)

		restoreArgs := restoreCSISnapshotArgs{
			Name:         testSuite.snapName,
			PVC:          testSuite.newPVCName,
			Namespace:    testSuite.namespace,
			StorageClass: testSuite.storageClass,
			RestoreSize:  originalPVC.Spec.Resources.Requests.Storage(),
			VolumeMode:   *originalPVC.Spec.VolumeMode,
			AccessModes:  originalPVC.Spec.AccessModes,
			Labels:       nil,
		}
		pvc, err := restoreCSISnapshot(ctx, fakeCli, restoreArgs)
		c.Assert(err, IsNil)
		c.Assert(pvc.Name, Equals, testSuite.newPVCName)

		err = fakeCli.CoreV1().Namespaces().Delete(ctx, testSuite.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}

func (testSuite *RestoreCSISnapshotTestSuite) TestValidateVolumeModeArg(c *C) {
	for _, scenario := range []struct {
		Arg         corev1.PersistentVolumeMode
		ExpectedErr Checker
	}{
		{
			Arg:         "test",
			ExpectedErr: NotNil,
		},
		{
			Arg:         corev1.PersistentVolumeFilesystem,
			ExpectedErr: IsNil,
		},
	} {
		err := validateVolumeModeArg(scenario.Arg)
		c.Assert(err, scenario.ExpectedErr)
	}
}

func (testSuite *RestoreCSISnapshotTestSuite) TestValidateAccessModeArg(c *C) {
	for _, scenario := range []struct {
		Arg         []corev1.PersistentVolumeAccessMode
		ExpectedErr Checker
	}{
		{
			Arg:         []corev1.PersistentVolumeAccessMode{"test"},
			ExpectedErr: NotNil,
		},
		{
			Arg:         []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			ExpectedErr: IsNil,
		},
	} {
		err := validateVolumeAccessModesArg(scenario.Arg)
		c.Assert(err, scenario.ExpectedErr)
	}
}

func createPVC(c *C, namespace string, pvc *corev1.PersistentVolumeClaim, fakeCli *fake.Clientset) {
	_, err := fakeCli.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	c.Assert(err, IsNil)
}

func getOriginalPVCManifest(pvcName, storageClassName string) *corev1.PersistentVolumeClaim {
	volumeMode := corev1.PersistentVolumeFilesystem
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			VolumeMode:       &volumeMode,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}
