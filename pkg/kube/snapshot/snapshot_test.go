// Copyright 2019 Kasten Inc.
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
	"strconv"
	"strings"
	"testing"
	"time"

	snapshot "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	snapshotclient "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	snapshotfake "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned/fake"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/volume"
	"github.com/kanisterio/kanister/pkg/poll"
)

func Test(t *testing.T) { TestingT(t) }

type SnapshotTestSuite struct {
	sourceNamespace string
	targetNamespace string
	cli             kubernetes.Interface
	snapCli         snapshotclient.Interface
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

	sc, err := kube.NewSnapshotClient()
	c.Assert(err, IsNil)
	s.snapCli = sc

	vscs, err := sc.VolumesnapshotV1alpha1().VolumeSnapshotClasses().List(metav1.ListOptions{})
	if err != nil && !k8errors.IsNotFound(err) {
		c.Logf("Failed to query VolumeSnapshotClass, skipping test. Error: %v", err)
		c.Fail()
	}
	var snapshotterName string
	if len(vscs.Items) != 0 {
		vsClass, err := sc.VolumesnapshotV1alpha1().VolumeSnapshotClasses().Get(vscs.Items[0].Name, metav1.GetOptions{})
		if err != nil {
			c.Logf("Failed to get VolumeSnapshotClass, skipping test. Error: %v", err)
			c.Fail()
		}
		snapshotterName = vsClass.Snapshotter
		s.snapshotClass = &vsClass.Name
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
	fakeCli := fake.NewSimpleClientset()
	fakeSnapCli := snapshotfake.NewSimpleClientset()

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

	err = Create(context.Background(), fakeCli, fakeSnapCli, snapshotName, defaultNamespace, volName, &fakeClass, false)
	c.Assert(err, IsNil)
	snap, err := Get(context.Background(), fakeSnapCli, snapshotName, defaultNamespace)
	c.Assert(err, IsNil)
	c.Assert(snap.Name, Equals, snapshotName)

	err = Create(context.Background(), fakeCli, fakeSnapCli, snapshotName, defaultNamespace, volName, &fakeClass, false)
	c.Assert(err, NotNil)
	err = Delete(context.Background(), fakeSnapCli, snap.Name, snap.Namespace)
	c.Assert(err, IsNil)
	err = Delete(context.Background(), fakeSnapCli, snap.Name, snap.Namespace)
	c.Assert(err, NotNil)
}

func (s *SnapshotTestSuite) TestVolumeSnapshotCloneFake(c *C) {
	fakeSnapshotName := "snap-1-fake"
	fakeContentName := "snapcontent-1-fake"

	content := &snapshot.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name: fakeContentName,
		},
		Spec: snapshot.VolumeSnapshotContentSpec{
			VolumeSnapshotSource: snapshot.VolumeSnapshotSource{
				CSI: &snapshot.CSIVolumeSnapshotSource{
					Driver:         fakeDriver,
					SnapshotHandle: fakeSnapshotHandle,
				},
			},
			VolumeSnapshotClassName: &fakeClass,
			VolumeSnapshotRef: &corev1.ObjectReference{
				Name:      fakeSnapshotName,
				Namespace: defaultNamespace,
			},
		},
	}
	ctime := metav1.Now()
	snapshot := &snapshot.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeSnapshotName,
			Namespace: defaultNamespace,
		},
		Spec: snapshot.VolumeSnapshotSpec{
			SnapshotContentName:     fakeContentName,
			VolumeSnapshotClassName: &fakeClass,
		},
		Status: snapshot.VolumeSnapshotStatus{
			ReadyToUse:   true,
			CreationTime: &ctime,
		},
	}

	snapCli := snapshotfake.NewSimpleClientset()
	fakeTargetNamespace := "new-ns"
	fakeClone := "clone-1"

	_, err := snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(defaultNamespace).Create(snapshot)
	c.Assert(err, IsNil)
	_, err = snapCli.VolumesnapshotV1alpha1().VolumeSnapshotContents().Create(content)
	c.Assert(err, IsNil)

	_, err = Get(context.Background(), snapCli, fakeSnapshotName, defaultNamespace)
	c.Assert(err, IsNil)

	err = Clone(context.Background(), snapCli, fakeSnapshotName, defaultNamespace, fakeClone, fakeTargetNamespace, false)
	c.Assert(err, IsNil)

	clone, err := Get(context.Background(), snapCli, fakeClone, fakeTargetNamespace)
	c.Assert(err, IsNil)

	cloneContent, err := snapCli.VolumesnapshotV1alpha1().VolumeSnapshotContents().Get(clone.Spec.SnapshotContentName, metav1.GetOptions{})
	c.Assert(err, IsNil)
	c.Assert(strings.HasPrefix(cloneContent.Name, fakeClone), Equals, true)
}

func (s *SnapshotTestSuite) TestVolumeSnapshot(c *C) {
	if s.snapshotClass == nil {
		c.Skip("No Volumesnapshotclass in the cluster, create a volumesnapshotclass in the cluster")
	}
	if s.storageClassCSI == nil {
		c.Skip("No Storageclass with CSI provisioner, install CSI and create a storageclass for it")
	}
	c.Logf("VolumeSnapshot test - source namespace: %s - target namespace: %s", s.sourceNamespace, s.targetNamespace)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	size, err := resource.ParseQuantity("1Gi")
	c.Assert(err, IsNil)

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: volNamePrefix,
			Namespace:    s.sourceNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): size,
				},
			},
			StorageClassName: s.storageClassCSI,
		},
	}
	pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.sourceNamespace).Create(pvc)
	c.Assert(err, IsNil)
	poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pvc.Status.Phase == corev1.ClaimBound, nil
	})

	snapshotName := snapshotNamePrefix + strconv.Itoa(int(time.Now().UnixNano()))
	wait := true
	err = Create(ctx, s.cli, s.snapCli, snapshotName, s.sourceNamespace, pvc.Name, s.snapshotClass, wait)
	c.Assert(err, IsNil)

	snap, err := Get(ctx, s.snapCli, snapshotName, s.sourceNamespace)
	c.Assert(err, IsNil)
	c.Assert(snap.Name, Equals, snapshotName)
	c.Assert(snap.Status.ReadyToUse, Equals, true)

	err = Create(ctx, s.cli, s.snapCli, snapshotName, s.sourceNamespace, pvc.Name, s.snapshotClass, wait)
	c.Assert(err, NotNil)

	snapshotCloneName := snapshotName + "-clone"
	volumeCloneName := pvc.Name + "-clone"
	err = Clone(ctx, s.snapCli, snapshotName, s.sourceNamespace, snapshotCloneName, s.targetNamespace, wait)
	c.Assert(err, IsNil)

	_, err = volume.CreatePVCFromSnapshot(ctx, s.cli, s.snapCli, s.targetNamespace, volumeCloneName, "", snapshotCloneName, nil)
	c.Assert(err, IsNil)
	poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.targetNamespace).Get(volumeCloneName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pvc.Status.Phase == corev1.ClaimBound, nil
	})

	// Try with a greater restore size.
	sizeNew := 2
	volumeCloneName += "-2"
	_, err = volume.CreatePVCFromSnapshot(ctx, s.cli, s.snapCli, s.targetNamespace, volumeCloneName, "", snapshotCloneName, &sizeNew)
	c.Assert(err, IsNil)
	poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.targetNamespace).Get(volumeCloneName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pvc.Status.Phase == corev1.ClaimBound, nil
	})

	err = Delete(ctx, s.snapCli, snap.Name, snap.Namespace)
	c.Assert(err, IsNil)

	err = Delete(ctx, s.snapCli, snap.Name, snap.Namespace)
	c.Assert(err, NotNil)

}

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

	vss, errb := s.snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(ns).List(metav1.ListOptions{})
	if errb != nil {
		c.Logf("Failed to list snapshots, Namespace: %s, Error: %v", ns, errb)
	} else {
		for _, vs := range vss.Items {
			if err := s.snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(ns).Delete(vs.Name, &metav1.DeleteOptions{}); err != nil {
				errb = err
				c.Logf("Failed to delete snapshot, Volumesnapshot: %s, Namespace %s, Error: %v", vs.Name, vs.Namespace, err)
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
