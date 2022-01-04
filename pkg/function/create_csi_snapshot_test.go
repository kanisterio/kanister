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

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	scv1 "k8s.io/api/storage/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	// CreateCSISnapshotTestNamespace is the namespace where testing is done
	CreateCSISnapshotTestNamespace = "test-create-csi-snapshot"
	// CreateCSISnapshotPVCName is the name of the PVC that will be captured
	CreateCSISnapshotPVCName = "test-pvc"
	// CreateCSISnapshotSnapshotName is the name of the snapshot
	CreateCSISnapshotSnapshotName = "test-snapshot"
)

type CreateCSISnapshotTestSuite struct {
	fakeCli             *fake.Clientset
	fakeSnapshotter     snapshot.Snapshotter
	snapName            string
	pvcName             string
	namespace           string
	volumeSnapshotClass string
	storageClass        string
}

var _ = Suite(&CreateCSISnapshotTestSuite{})

func (testSuite *CreateCSISnapshotTestSuite) SetUpSuite(c *C) {
	ctx := context.Background()

	kubeCli, err := kube.NewClient()
	c.Assert(err, IsNil)
	dynCli, err := kube.NewDynamicClient()
	c.Assert(err, IsNil)

	volumeSnapshotClass, driver := FindSnapshotClassName(c, ctx, dynCli, snapshot.VolSnapClassGVR, snapv1.VolumeSnapshotClass{})
	testSuite.volumeSnapshotClass = volumeSnapshotClass

	storageClasses, err := kubeCli.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	c.Assert(err, IsNil)
	for _, class := range storageClasses.Items {
		if class.Provisioner == driver && *class.VolumeBindingMode == scv1.VolumeBindingImmediate {
			testSuite.storageClass = class.Name
			break
		}
	}
	testSuite.pvcName = CreateCSISnapshotPVCName
	testSuite.snapName = CreateCSISnapshotSnapshotName

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1alpha1", Kind: "VolumeSnapshotClassList"}, &unstructured.UnstructuredList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1beta1", Kind: "VolumeSnapshotClassList"}, &unstructured.UnstructuredList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotClassList"}, &unstructured.UnstructuredList{})

	fakeCli := fake.NewSimpleClientset()
	fakeCli.Resources = []*metav1.APIResourceList{
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
	}
	testSuite.fakeCli = fakeCli
	testSuite.fakeSnapshotter, err = snapshot.NewSnapshotter(fakeCli, dynfake.NewSimpleDynamicClient(scheme))
	c.Assert(err, IsNil)

	testSuite.namespace = CreateCSISnapshotTestNamespace
	_, err = testSuite.fakeCli.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testSuite.namespace}}, metav1.CreateOptions{})
	c.Assert(err, IsNil)
}

func (testSuite *CreateCSISnapshotTestSuite) TestCreateCSISnapshot(c *C) {
	testSuite.createPVC(c)
	err := testSuite.fakeSnapshotter.Create(context.Background(), testSuite.snapName, testSuite.namespace, testSuite.pvcName, &testSuite.volumeSnapshotClass, false, nil)
	c.Assert(err, IsNil)
	vs, err := testSuite.fakeSnapshotter.Get(context.Background(), testSuite.snapName, testSuite.namespace)
	c.Assert(err, IsNil)
	c.Assert(vs.Name, Equals, testSuite.snapName)
}

func (testSuite *CreateCSISnapshotTestSuite) TearDownSuite(c *C) {
	ctx := context.Background()
	err := testSuite.fakeCli.CoreV1().Namespaces().Delete(ctx, testSuite.namespace, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func (testSuite *CreateCSISnapshotTestSuite) createPVC(c *C) {
	_, err := testSuite.fakeCli.CoreV1().PersistentVolumeClaims(testSuite.namespace).Create(context.TODO(), getPVCManifest(testSuite.pvcName, testSuite.storageClass), metav1.CreateOptions{})
	c.Assert(err, IsNil)
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

func FindSnapshotClassName(c *C, ctx context.Context, dynCli dynamic.Interface, gvr schema.GroupVersionResource, object interface{}) (string, string) {
	us, err := dynCli.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil && !k8errors.IsNotFound(err) {
		c.Logf("Failed to query VolumeSnapshotClass, skipping test. Error: %v", err)
		c.Fail()
	}
	var snapshotterName, snapshotClass string
	if (us != nil) && len(us.Items) != 0 {
		usClass, err := dynCli.Resource(gvr).Get(ctx, us.Items[0].GetName(), metav1.GetOptions{})
		if err != nil {
			c.Logf("Failed to get VolumeSnapshotClass, skipping test. Error: %v", err)
			c.Fail()
		}
		snapshotClass = usClass.GetName()
		if vsc, ok := object.(snapv1.VolumeSnapshotClass); ok {
			err := snapshot.TransformUnstructured(usClass, &vsc)
			if err != nil {
				c.Logf("Failed to query VolumeSnapshotClass, skipping test. Error: %v", err)
				c.Fail()
			}
			snapshotterName = vsc.Driver
		}
	}
	return snapshotClass, snapshotterName
}
