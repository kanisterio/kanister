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

	"github.com/kanisterio/kanister/pkg/kube/snapshot"

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	// DeleteCSISnapshotTestNamespace is the namespace where testing is done
	DeleteCSISnapshotTestNamespace = "test-restore-csi-snapshot"
	// DeleteCSISnapshotOriginalPVCName is the name of the PVC that will be captured
	DeleteCSISnapshotOriginalPVCName = "test-pvc"
	// DeleteCSISnapshotPVCName is the name of the new PVC that will be restored
	DeleteCSISnapshotNewPVCName = "test-pvc-restored"
	// DeleteCSISnapshotSnapshotName is the name of the snapshot
	DeleteCSISnapshotSnapshotName = "test-snapshot"
	// DeleteCSISnapshotSnapshotClass is the fake snapshot class
	DeleteCSISnapshotSnapshotClass = "test-snapshot-class"
	// DeleteCSISnapshotStorageClass is the fake storage class
	DeleteCSISnapshotStorageClass = "test-storage-class"
)

type DeleteCSISnapshotTestSuite struct {
	snapName            string
	pvcName             string
	newPVCName          string
	namespace           string
	volumeSnapshotClass string
	storageClass        string
}

var _ = Suite(&DeleteCSISnapshotTestSuite{})

func (testSuite *DeleteCSISnapshotTestSuite) SetUpSuite(c *C) {
	testSuite.volumeSnapshotClass = DeleteCSISnapshotSnapshotClass
	testSuite.storageClass = DeleteCSISnapshotStorageClass
	testSuite.pvcName = DeleteCSISnapshotOriginalPVCName
	testSuite.newPVCName = DeleteCSISnapshotNewPVCName
	testSuite.snapName = DeleteCSISnapshotSnapshotName
	testSuite.namespace = DeleteCSISnapshotTestNamespace
}

func (testSuite *DeleteCSISnapshotTestSuite) TestDeleteCSISnapshot(c *C) {
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
		fakeCli := fake.NewSimpleClientset()
		fakeCli.Resources = []*metav1.APIResourceList{apiResourceList}

		_, err := fakeCli.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testSuite.namespace}}, metav1.CreateOptions{})
		c.Assert(err, IsNil)

		scheme := runtime.NewScheme()
		fakeSnapshotter, err := snapshot.NewSnapshotter(fakeCli, dynfake.NewSimpleDynamicClient(scheme))
		c.Assert(err, IsNil)

		originalPVC := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: testSuite.pvcName,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				StorageClassName: &testSuite.storageClass,
				AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		}
		_, err = fakeCli.CoreV1().PersistentVolumeClaims(testSuite.namespace).Create(context.TODO(), originalPVC, metav1.CreateOptions{})
		c.Assert(err, IsNil)

		err = fakeSnapshotter.Create(context.Background(), testSuite.snapName, testSuite.namespace, testSuite.pvcName, &testSuite.volumeSnapshotClass, false, nil)
		c.Assert(err, IsNil)

		vs, err := fakeSnapshotter.Get(context.Background(), testSuite.snapName, testSuite.namespace)
		c.Assert(err, IsNil)
		c.Assert(vs.Name, Equals, testSuite.snapName)

		restoreArgs := restoreCSISnapshotArgs{
			Name:         testSuite.snapName,
			PVC:          testSuite.newPVCName,
			Namespace:    testSuite.namespace,
			StorageClass: testSuite.storageClass,
			RestoreSize:  originalPVC.Spec.Resources.Requests.Storage(),
			AccessModes:  originalPVC.Spec.AccessModes,
			Labels:       nil,
		}
		newPVC := newPVCManifest(restoreArgs)
		_, err = fakeCli.CoreV1().PersistentVolumeClaims(restoreArgs.Namespace).Create(context.TODO(), newPVC, metav1.CreateOptions{})
		c.Assert(err, IsNil)
		c.Assert(newPVC.Name, Equals, testSuite.newPVCName)

		vs, err = fakeSnapshotter.Delete(context.Background(), testSuite.snapName, testSuite.namespace)
		c.Assert(err, IsNil)
		vs, err = fakeSnapshotter.Get(context.Background(), testSuite.snapName, testSuite.namespace)
		c.Assert(err, NotNil)


		err = fakeCli.CoreV1().Namespaces().Delete(context.Background(), testSuite.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}
