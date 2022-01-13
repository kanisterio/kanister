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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/kube/snapshot"
)

const (
	// CreateCSISnapshotTestNamespace is the namespace where testing is done
	CreateCSISnapshotTestNamespace = "test-create-csi-snapshot"
	// CreateCSISnapshotPVCName is the name of the PVC that will be captured
	CreateCSISnapshotPVCName = "test-pvc"
	// CreateCSISnapshotSnapshotName is the name of the snapshot
	CreateCSISnapshotSnapshotName = "test-snapshot"
	// CreateCSISnapshotSnapshotClass is the fake snapshot class
	CreateCSISnapshotSnapshotClass = "test-snapshot-class"
	// CreateCSISnapshotStorageClass is the fake storage class
	CreateCSISnapshotStorageClass = "test-storage-class"
)

type CreateCSISnapshotTestSuite struct {
	snapName            string
	pvcName             string
	namespace           string
	volumeSnapshotClass string
	storageClass        string
}

var _ = Suite(&CreateCSISnapshotTestSuite{})

func (testSuite *CreateCSISnapshotTestSuite) SetUpSuite(c *C) {
	testSuite.volumeSnapshotClass = CreateCSISnapshotSnapshotClass
	testSuite.storageClass = CreateCSISnapshotStorageClass
	testSuite.pvcName = CreateCSISnapshotPVCName
	testSuite.snapName = CreateCSISnapshotSnapshotName
	testSuite.namespace = CreateCSISnapshotTestNamespace
}

func (testSuite *CreateCSISnapshotTestSuite) TestCreateCSISnapshot(c *C) {
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

		_, err := fakeCli.CoreV1().Namespaces().Create(ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testSuite.namespace}}, metav1.CreateOptions{})
		c.Assert(err, IsNil)

		scheme := runtime.NewScheme()
		fakeSnapshotter, err := snapshot.NewSnapshotter(fakeCli, dynfake.NewSimpleDynamicClient(scheme))
		c.Assert(err, IsNil)

		_, err = fakeCli.CoreV1().PersistentVolumeClaims(testSuite.namespace).Create(ctx, getPVCManifest(testSuite.pvcName, testSuite.storageClass), metav1.CreateOptions{})
		c.Assert(err, IsNil)

		_, err = createCSISnapshot(ctx, fakeSnapshotter, nil, testSuite.snapName, testSuite.namespace, testSuite.pvcName, testSuite.volumeSnapshotClass)
		c.Assert(err, IsNil)

		err = fakeCli.CoreV1().Namespaces().Delete(ctx, testSuite.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}

func getPVCManifest(pvcName, storageClassName string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}
