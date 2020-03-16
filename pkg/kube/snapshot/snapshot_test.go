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

package snapshot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	snapshotfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/kube"
)

func Test(t *testing.T) { TestingT(t) }

type SnapshotTestSuite struct {
	sourceNamespace string
	targetNamespace string
	snap            Snapshotter
	cli             kubernetes.Interface
	dynCli          dynamic.Interface
	snapshotClass   *string
	storageClassCSI *string
}

var _ = Suite(&SnapshotTestSuite{})

var (
	defaultNamespace   = "default"
	fakeClass          = "fake-snapshotclass"
	fakeDriver         = "fake-driver"
	fakeSnapshotHandle = "snapshot/csi/handle1"

	testTimeout = 5 * time.Minute

	volNamePrefix      = "pvc-snapshot-test-"
	snapshotNamePrefix = "snap-snapshot-test-"
)

func (s *SnapshotTestSuite) SetUpSuite(c *C) {
	suffix := strconv.Itoa(int(time.Now().UnixNano() % 100000))
	s.sourceNamespace = "snapshot-test-source-" + suffix
	s.targetNamespace = "snapshot-test-target-" + suffix

	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	dynCli, err := kube.NewDynamicClient()
	c.Assert(err, IsNil)
	s.dynCli = dynCli

	s.snap = NewSnapshotAlpha(dynCli, cli)

	us, err := dynCli.Resource(VolSnapClassGVR).Namespace("").List(metav1.ListOptions{})
	if err != nil && !k8errors.IsNotFound(err) {
		c.Logf("Failed to query VolumeSnapshotClass, skipping test. Error: %v", err)
		c.Fail()
	}
	var snapshotterName string
	if len(us.Items) != 0 {
		usClass, err := dynCli.Resource(VolSnapClassGVR).Namespace("").Get(us.Items[0].GetName(), metav1.GetOptions{})
		if err != nil {
			c.Logf("Failed to get VolumeSnapshotClass, skipping test. Error: %v", err)
			c.Fail()
		}
		vsc := VolumeSnapshotClass{}
		err = transformUnstructured(usClass, &vsc)
		if err != nil {
			c.Logf("Failed to query VolumeSnapshotClass, skipping test. Error: %v", err)
			c.Fail()
		}
		vsName := usClass.GetName()
		snapshotterName = vsc.Snapshotter
		s.snapshotClass = &vsName
	}

	storageClasses, err := cli.StorageV1().StorageClasses().List(metav1.ListOptions{})
	c.Assert(err, IsNil)
	for _, class := range storageClasses.Items {
		if class.Provisioner == snapshotterName {
			s.storageClassCSI = &class.Name
			break
		}
	}

	_, err = cli.CoreV1().Namespaces().Create(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s.sourceNamespace}})
	c.Assert(err, IsNil)

	_, err = cli.CoreV1().Namespaces().Create(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s.targetNamespace}})
	c.Assert(err, IsNil)
}

func (s *SnapshotTestSuite) TearDownSuite(c *C) {
	s.cleanupNamespace(c, s.sourceNamespace)
	s.cleanupNamespace(c, s.targetNamespace)
}

func (s *SnapshotTestSuite) TestVolumeSnapshotFake(c *C) {
	snapshotName := "snap-1-fake"
	volName := "pvc-1-fake"
	scheme := runtime.NewScheme()
	fakeCli := fake.NewSimpleClientset()
	fakeSs := NewSnapshotAlpha(snapshotfake.NewSimpleDynamicClient(scheme), fakeCli)

	size, err := resource.ParseQuantity("1Gi")
	c.Assert(err, IsNil)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: volName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}
	_, err = fakeCli.CoreV1().PersistentVolumeClaims(defaultNamespace).Create(pvc)
	c.Assert(err, IsNil)

	err = fakeSs.Create(context.Background(), snapshotName, defaultNamespace, volName, &fakeClass, false)
	c.Assert(err, IsNil)
	snap, err := fakeSs.Get(context.Background(), snapshotName, defaultNamespace)
	c.Assert(err, IsNil)
	c.Assert(snap.Name, Equals, snapshotName)

	err = fakeSs.Create(context.Background(), snapshotName, defaultNamespace, volName, &fakeClass, false)
	c.Assert(err, NotNil)
	err = fakeSs.Delete(context.Background(), snap.Name, snap.Namespace)
	c.Assert(err, IsNil)
	err = fakeSs.Delete(context.Background(), snap.Name, snap.Namespace)
	c.Assert(err, IsNil)
}

func (s *SnapshotTestSuite) TestVolumeSnapshotCloneFake(c *C) {
	fakeSnapshotName := "snap-1-fake"
	fakeContentName := "snapcontent-1-fake"

	vsc := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", snapshotGroup, snapshotVersionAlpha),
			"kind":       volSnapClassKind,
			"metadata": map[string]interface{}{
				"name": fakeClass,
			},
			"snapshotter":    fakeDriver,
			"deletionPolicy": "Delete",
		},
	}

	content := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", snapshotGroup, snapshotVersionAlpha),
			"kind":       volSnapContentKind,
			"metadata": map[string]interface{}{
				"name": fakeContentName,
			},
			"spec": map[string]interface{}{
				"volumeSnapshotSource": map[string]interface{}{
					"csiVolumeSnapshotSource": map[string]interface{}{
						"driver":         fakeDriver,
						"snapshotHandle": fakeSnapshotHandle,
					},
				},
				"snapshotClassName": fakeClass,
				"volumeSnapshotRef": map[string]interface{}{
					"name":      fakeSnapshotName,
					"namespace": defaultNamespace,
				},
			},
		},
	}

	snap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", snapshotGroup, snapshotVersionAlpha),
			"kind":       volSnapKind,
			"metadata": map[string]interface{}{
				"name":      fakeSnapshotName,
				"namespace": defaultNamespace,
			},
			"spec": map[string]interface{}{
				"snapshotContentName": fakeContentName,
				"snapshotClassName":   fakeClass,
			},
			"status": map[string]interface{}{
				"readyToUse":   true,
				"creationTime": metav1.Now().Format("2006-01-02T15:04:05Z"),
			},
		},
	}

	scheme := runtime.NewScheme()
	dynCli := snapshotfake.NewSimpleDynamicClient(scheme)
	fakeTargetNamespace := "new-ns"
	fakeClone := "clone-1"

	_, err := dynCli.Resource(VolSnapClassGVR).Namespace("").Create(vsc, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	_, err = dynCli.Resource(VolSnapGVR).Namespace(defaultNamespace).Create(snap, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	_, err = dynCli.Resource(VolSnapContentGVR).Namespace("").Create(content, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	fakeSs := NewSnapshotAlpha(dynCli, nil)
	_, err = fakeSs.Get(context.Background(), fakeSnapshotName, defaultNamespace)
	c.Assert(err, IsNil)

	err = fakeSs.Clone(context.Background(), fakeSnapshotName, defaultNamespace, fakeClone, fakeTargetNamespace, false)
	c.Assert(err, IsNil)

	clone, err := fakeSs.Get(context.Background(), fakeClone, fakeTargetNamespace)
	c.Assert(err, IsNil)

	us, err := dynCli.Resource(VolSnapContentGVR).Namespace("").Get(clone.Spec.SnapshotContentName, metav1.GetOptions{})
	c.Assert(err, IsNil)
	cloneContent := VolumeSnapshotContent{}
	err = transformUnstructured(us, &cloneContent)
	c.Assert(err, IsNil)
	c.Assert(strings.HasPrefix(cloneContent.Name, fakeClone), Equals, true)
	c.Assert(cloneContent.Spec.DeletionPolicy, Equals, vsc.Object["deletionPolicy"])
}

//func (s *SnapshotTestSuite) TestVolumeSnapshot(c *C) {
//	if s.snapshotClass == nil {
//		c.Skip("No Volumesnapshotclass in the cluster, create a volumesnapshotclass in the cluster")
//	}
//	if s.storageClassCSI == nil {
//		c.Skip("No Storageclass with CSI provisioner, install CSI and create a storageclass for it")
//	}
//	c.Logf("VolumeSnapshot test - source namespace: %s - target namespace: %s", s.sourceNamespace, s.targetNamespace)
//	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
//	defer cancel()
//
//	size, err := resource.ParseQuantity("1Gi")
//	c.Assert(err, IsNil)
//
//	pvc := &corev1.PersistentVolumeClaim{
//		ObjectMeta: metav1.ObjectMeta{
//			GenerateName: volNamePrefix,
//			Namespace:    s.sourceNamespace,
//		},
//		Spec: corev1.PersistentVolumeClaimSpec{
//			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
//			Resources: corev1.ResourceRequirements{
//				Requests: corev1.ResourceList{
//					corev1.ResourceName(corev1.ResourceStorage): size,
//				},
//			},
//			StorageClassName: s.storageClassCSI,
//		},
//	}
//	pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.sourceNamespace).Create(pvc)
//	c.Assert(err, IsNil)
//	_ = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
//		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
//		if err != nil {
//			return false, err
//		}
//		return pvc.Status.Phase == corev1.ClaimBound, nil
//	})
//
//	snapshotName := snapshotNamePrefix + strconv.Itoa(int(time.Now().UnixNano()))
//	wait := true
//	err = Create(ctx, s.cli, s.dynCli, snapshotName, s.sourceNamespace, pvc.Name, s.snapshotClass, wait)
//	c.Assert(err, IsNil)
//
//	snap, err := Get(ctx, s.dynCli, snapshotName, s.sourceNamespace)
//	c.Assert(err, IsNil)
//	c.Assert(snap.Name, Equals, snapshotName)
//	c.Assert(snap.Status.ReadyToUse, Equals, true)
//
//	err = Create(ctx, s.cli, s.dynCli, snapshotName, s.sourceNamespace, pvc.Name, s.snapshotClass, wait)
//	c.Assert(err, NotNil)
//
//	snapshotCloneName := snapshotName + "-clone"
//	volumeCloneName := pvc.Name + "-clone"
//	err = Clone(ctx, s.dynCli, snapshotName, s.sourceNamespace, snapshotCloneName, s.targetNamespace, wait)
//	c.Assert(err, IsNil)
//
//	_, err = volume.CreatePVCFromSnapshot(ctx, s.cli, s.dynCli, s.targetNamespace, volumeCloneName, "", snapshotCloneName, nil)
//	c.Assert(err, IsNil)
//	_ = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
//		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.targetNamespace).Get(volumeCloneName, metav1.GetOptions{})
//		if err != nil {
//			return false, err
//		}
//		return pvc.Status.Phase == corev1.ClaimBound, nil
//	})
//
//	// Try with a greater restore size.
//	sizeNew := 2
//	volumeCloneName += "-2"
//	_, err = volume.CreatePVCFromSnapshot(ctx, s.cli, s.dynCli, s.targetNamespace, volumeCloneName, "", snapshotCloneName, &sizeNew)
//	c.Assert(err, IsNil)
//	_ = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
//		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.targetNamespace).Get(volumeCloneName, metav1.GetOptions{})
//		if err != nil {
//			return false, err
//		}
//		return pvc.Status.Phase == corev1.ClaimBound, nil
//	})
//
//	err = Delete(ctx, s.dynCli, snap.Name, snap.Namespace)
//	c.Assert(err, IsNil)
//
//	err = Delete(ctx, s.dynCli, snap.Name, snap.Namespace)
//	c.Assert(err, NotNil)
//}

func (s *SnapshotTestSuite) cleanupNamespace(c *C, ns string) {
	pvcs, erra := s.cli.CoreV1().PersistentVolumeClaims(ns).List(metav1.ListOptions{})
	if erra != nil {
		c.Logf("Failed to list PVCs, Namespace: %s, Error: %v", ns, erra)
	} else {
		for _, pvc := range pvcs.Items {
			if err := s.cli.CoreV1().PersistentVolumeClaims(ns).Delete(pvc.Name, &metav1.DeleteOptions{}); err != nil {
				erra = err
				c.Logf("Failed to delete PVC, PVC: %s, Namespace: %s, Error: %v", pvc.Name, ns, err)
			}
		}
	}

	vss, errb := s.dynCli.Resource(VolSnapGVR).Namespace(ns).List(metav1.ListOptions{})
	if errb != nil {
		c.Logf("Failed to list snapshots, Namespace: %s, Error: %v", ns, errb)
	} else {
		for _, vs := range vss.Items {
			if err := s.dynCli.Resource(VolSnapGVR).Namespace(ns).Delete(vs.GetName(), &metav1.DeleteOptions{}); err != nil {

				errb = err
				c.Logf("Failed to delete snapshot, Volumesnapshot: %s, Namespace %s, Error: %v", vs.GetName(), vs.GetNamespace(), err)
			}
		}
	}

	if erra != nil || errb != nil {
		c.Logf("Skipping deleting the namespace, Namespace: %s", ns)
	}

	err := s.cli.CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{})
	if err != nil {
		c.Logf("Failed to delete namespace, Namespace: %s, Error: %v", ns, err)
	}
}
