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
	// testDeleteNamespace specifies the namespace where testing is done
	testDeleteNamespace = "test-delete-csi-snapshot"
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
	testSuite.volumeSnapshotClass = snapshotClass
	testSuite.storageClass = storageClass
	testSuite.pvcName = originalPVCName
	testSuite.newPVCName = newPVCName
	testSuite.snapName = snapshotName
	testSuite.namespace = testDeleteNamespace
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
		ctx := context.Background()
		fakeCli := fake.NewSimpleClientset()
		fakeCli.Resources = []*metav1.APIResourceList{apiResourceList}

		_, err := fakeCli.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testSuite.namespace}}, metav1.CreateOptions{})
		c.Assert(err, IsNil)

		scheme := runtime.NewScheme()
		fakeSnapshotter, err := snapshot.NewSnapshotter(fakeCli, dynfake.NewSimpleDynamicClient(scheme))
		c.Assert(err, IsNil)

		originalPVC := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: testSuite.pvcName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &testSuite.storageClass,
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		}
		_, err = fakeCli.CoreV1().PersistentVolumeClaims(testSuite.namespace).Create(ctx, originalPVC, metav1.CreateOptions{})
		c.Assert(err, IsNil)
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
			AccessModes:  originalPVC.Spec.AccessModes,
			Labels:       nil,
		}
		newPVC := newPVCManifest(restoreArgs)
		_, err = fakeCli.CoreV1().PersistentVolumeClaims(restoreArgs.Namespace).Create(ctx, newPVC, metav1.CreateOptions{})
		c.Assert(err, IsNil)
		c.Assert(newPVC.Name, Equals, testSuite.newPVCName)

		_, err = deleteCSISnapshot(ctx, fakeSnapshotter, testSuite.snapName, testSuite.namespace)
		c.Assert(err, IsNil)
		_, err = fakeSnapshotter.Get(ctx, testSuite.snapName, testSuite.namespace)
		c.Assert(err, NotNil)

		err = fakeCli.CoreV1().Namespaces().Delete(ctx, testSuite.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}
