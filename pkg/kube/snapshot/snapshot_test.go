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

package snapshot_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	scv1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/kube/volume"
	"github.com/kanisterio/kanister/pkg/poll"
)

func Test(t *testing.T) { check.TestingT(t) }

type SnapshotTestSuite struct {
	sourceNamespace string
	targetNamespace string
	snapshotter     snapshot.Snapshotter
	cli             kubernetes.Interface
	dynCli          dynamic.Interface
	snapshotClass   *string
	storageClassCSI string
}

var _ = check.Suite(&SnapshotTestSuite{})

var (
	defaultNamespace   = "default"
	fakeClass          = "fake-snapshotclass"
	fakeDriver         = "fake-driver"
	fakeSC             = "fake-storageclass"
	fakeSnapshotHandle = "snapshot/csi/handle1"

	testTimeout = 5 * time.Minute

	volNamePrefix      = "pvc-snapshot-test-"
	snapshotNamePrefix = "snap-snapshot-test-"
)

func (s *SnapshotTestSuite) SetUpSuite(c *check.C) {
	suffix := strconv.Itoa(int(time.Now().UnixNano() % 100000))
	s.sourceNamespace = "snapshot-test-source-" + suffix
	s.targetNamespace = "snapshot-test-target-" + suffix
	ctx := context.Background()
	cli, err := kube.NewClient()
	c.Assert(err, check.IsNil)
	s.cli = cli

	dynCli, err := kube.NewDynamicClient()
	c.Assert(err, check.IsNil)
	s.dynCli = dynCli

	s.snapshotter = snapshot.NewSnapshotter(cli, dynCli)

	snapClass, driver := findSnapshotClassName(c, ctx, s.dynCli, snapshot.VolSnapClassGVR, snapv1.VolumeSnapshotClass{})
	if snapClass != "" {
		s.snapshotClass = &snapClass
	}

	storageClasses, err := cli.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	c.Assert(err, check.IsNil)
	for _, class := range storageClasses.Items {
		if class.Provisioner == driver && *class.VolumeBindingMode == scv1.VolumeBindingImmediate {
			s.storageClassCSI = class.Name
		}
	}

	_, err = cli.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s.sourceNamespace}}, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	_, err = cli.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s.targetNamespace}}, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
}

func (s *SnapshotTestSuite) TearDownSuite(c *check.C) {
	s.cleanupNamespace(c, s.sourceNamespace)
	s.cleanupNamespace(c, s.targetNamespace)
}

func (s *SnapshotTestSuite) TestVolumeSnapshotFake(c *check.C) {
	snapshotName := "snap-1-fake"
	volName := "pvc-1-fake"
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotClassList"}, &unstructured.UnstructuredList{})

	fakeCli := fake.NewSimpleClientset()

	size, err := resource.ParseQuantity("1Gi")
	c.Assert(err, check.IsNil)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: volName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}
	_, err = fakeCli.CoreV1().PersistentVolumeClaims(defaultNamespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	fakeSs := snapshot.NewSnapshotter(fakeCli, dynfake.NewSimpleDynamicClient(scheme))
	snapshotMeta := snapshot.ObjectMeta{
		Name:      snapshotName,
		Namespace: defaultNamespace,
	}
	err = fakeSs.Create(context.Background(), volName, &fakeClass, false, snapshotMeta)
	c.Assert(err, check.IsNil)
	snap, err := fakeSs.Get(context.Background(), snapshotName, defaultNamespace)
	c.Assert(err, check.IsNil)
	c.Assert(snap.Name, check.Equals, snapshotName)

	err = fakeSs.Create(context.Background(), volName, &fakeClass, false, snapshotMeta)
	c.Assert(err, check.NotNil)
	deletedSnap, err := fakeSs.Delete(context.Background(), snap.Name, snap.Namespace)
	c.Assert(err, check.IsNil)
	c.Assert(deletedSnap.Name, check.Equals, snap.Name)
	_, err = fakeSs.Delete(context.Background(), snap.Name, snap.Namespace)
	c.Assert(err, check.IsNil)
}

func (s *SnapshotTestSuite) TestVolumeSnapshotClassCloneFake(c *check.C) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotClassList"}, &unstructured.UnstructuredList{})

	fakeCli := fake.NewSimpleClientset(
		&scv1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: fakeSC,
			},
			Provisioner: fakeDriver,
		},
	)
	dynCli := dynfake.NewSimpleDynamicClient(scheme)

	fakeParams := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}

	for _, tc := range []struct {
		sourceSnapClassSpec *unstructured.Unstructured
		snapClassGVR        schema.GroupVersionResource
		snapshotter         snapshot.Snapshotter
	}{
		{
			sourceSnapClassSpec: snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, fakeClass, fakeDriver, snapshot.DeletionPolicyDelete, fakeParams),
			snapClassGVR:        snapshot.VolSnapClassGVR,
			snapshotter:         snapshot.NewSnapshotter(fakeCli, dynCli),
		},
	} {
		annotationKeyToKeep := "keepme"
		annotationKeyToRemove := "removeme"
		tc.sourceSnapClassSpec.SetAnnotations(map[string]string{
			annotationKeyToKeep:   "true",
			annotationKeyToRemove: "true",
		})
		_, err := dynCli.Resource(tc.snapClassGVR).Create(ctx, tc.sourceSnapClassSpec, metav1.CreateOptions{})
		c.Assert(err, check.IsNil)
		err = tc.snapshotter.CloneVolumeSnapshotClass(ctx, tc.sourceSnapClassSpec.GetName(), "targetClass", snapshot.DeletionPolicyRetain, []string{annotationKeyToRemove})
		c.Assert(err, check.IsNil)

		// New VSC exists
		createdVSC, err := dynCli.Resource(tc.snapClassGVR).Get(ctx, "targetClass", metav1.GetOptions{})
		c.Assert(err, check.IsNil)

		// Annotations are set correctly
		c.Assert(createdVSC.GetAnnotations(), check.DeepEquals, map[string]string{annotationKeyToKeep: "true"})
		c.Assert(createdVSC.GetLabels(), check.DeepEquals, map[string]string{snapshot.CloneVolumeSnapshotClassLabelName: tc.sourceSnapClassSpec.GetName()})

		// Parameters are set correctly
		c.Assert(createdVSC.Object["parameters"], check.DeepEquals, snapshot.Mss2msi(fakeParams))

		// Lookup by old annotation correctly returns the source VSC
		scWithOldAnnotation, err := tc.snapshotter.GetVolumeSnapshotClass(ctx, annotationKeyToRemove, "true", fakeSC)
		c.Assert(err, check.IsNil)
		c.Assert(scWithOldAnnotation, check.Equals, tc.sourceSnapClassSpec.GetName())

		// Clone again succeeds
		err = tc.snapshotter.CloneVolumeSnapshotClass(ctx, tc.sourceSnapClassSpec.GetName(), "targetClass", snapshot.DeletionPolicyRetain, []string{annotationKeyToRemove})
		c.Assert(err, check.IsNil)
	}
}

func (s *SnapshotTestSuite) TestVolumeSnapshotCloneFake(c *check.C) {
	ctx := context.Background()
	fakeSnapshotName := "snap-1-fake"
	fakeContentName := "snapcontent-1-fake"
	deletionPolicy := "Delete"
	fakeContentAnnotation := map[string]string{
		"snapshot.storage.kubernetes.io/allow-volume-mode-change": "true",
	}

	fakeSnapshotAnnotation := map[string]string{
		"testAnnotation": "true",
	}

	scheme := runtime.NewScheme()
	dynCli := dynfake.NewSimpleDynamicClient(scheme)
	fakeTargetNamespace := "new-ns"
	fakeClone := "clone-1"

	for _, tc := range []struct {
		snapClassSpec     *unstructured.Unstructured
		snapClassGVR      schema.GroupVersionResource
		contentSpec       *unstructured.Unstructured
		contentGVR        schema.GroupVersionResource
		snapSpec          *unstructured.Unstructured
		snapGVR           schema.GroupVersionResource
		snapContentObject metav1.Object
		fakeSs            snapshot.Snapshotter
	}{
		{
			snapClassSpec: snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, fakeClass, fakeDriver, deletionPolicy, nil),
			snapClassGVR:  snapshot.VolSnapClassGVR,
			contentSpec: snapshot.UnstructuredVolumeSnapshotContent(snapshot.VolSnapContentGVR, fakeSnapshotName, defaultNamespace, deletionPolicy, fakeDriver, fakeSnapshotHandle, fakeClass, snapshot.ObjectMeta{
				Name:        fakeContentName,
				Annotations: fakeContentAnnotation,
			}),
			contentGVR: snapshot.VolSnapContentGVR,

			snapSpec: snapshot.UnstructuredVolumeSnapshot(snapshot.VolSnapGVR, "", fakeClass, snapshot.ObjectMeta{
				Name:        fakeSnapshotName,
				Namespace:   defaultNamespace,
				Annotations: fakeSnapshotAnnotation,
			}, snapshot.ObjectMeta{
				Name: fakeContentName,
			}),
			snapGVR:           snapshot.VolSnapGVR,
			snapContentObject: &snapv1.VolumeSnapshotContent{},
			fakeSs:            snapshot.NewSnapshotter(nil, dynCli),
		},
	} {
		tc.contentSpec.Object["status"] = map[string]interface{}{
			"snapshotHandle": fakeSnapshotHandle,
		}
		tc.snapSpec.Object["status"] = map[string]interface{}{
			"readyToUse":                     true,
			"boundVolumeSnapshotContentName": fakeContentName,
			"creationTime":                   metav1.Now().Format("2006-01-02T15:04:05Z"),
		}
		_, err := dynCli.Resource(tc.snapClassGVR).Create(ctx, tc.snapClassSpec, metav1.CreateOptions{})
		c.Assert(err, check.IsNil)
		_, err = dynCli.Resource(tc.snapGVR).Namespace(defaultNamespace).Create(ctx, tc.snapSpec, metav1.CreateOptions{})
		c.Assert(err, check.IsNil)
		_, err = dynCli.Resource(tc.contentGVR).Create(ctx, tc.contentSpec, metav1.CreateOptions{})
		c.Assert(err, check.IsNil)

		_, err = tc.fakeSs.Get(context.Background(), fakeSnapshotName, defaultNamespace)
		c.Assert(err, check.IsNil)

		err = tc.fakeSs.Clone(context.Background(), fakeSnapshotName, defaultNamespace, false, snapshot.ObjectMeta{
			Name:        fakeClone,
			Namespace:   fakeTargetNamespace,
			Labels:      map[string]string{"tag": "value"},
			Annotations: fakeSnapshotAnnotation,
		}, snapshot.ObjectMeta{
			Annotations: fakeContentAnnotation,
		})

		c.Assert(err, check.IsNil)

		clone, err := tc.fakeSs.Get(context.Background(), fakeClone, fakeTargetNamespace)
		c.Assert(err, check.IsNil)

		us, err := dynCli.Resource(tc.contentGVR).Get(ctx, *clone.Spec.Source.VolumeSnapshotContentName, metav1.GetOptions{})
		c.Assert(err, check.IsNil)
		err = snapshot.TransformUnstructured(us, tc.snapContentObject)
		c.Assert(err, check.IsNil)
	}
}

func (s *SnapshotTestSuite) TestWaitOnReadyToUse(c *check.C) {
	snapshotNameBase := "snap-1-fake"
	volName := "pvc-1-fake"
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotClassList"}, &unstructured.UnstructuredList{})

	fakeCli := fake.NewSimpleClientset()

	size, err := resource.ParseQuantity("1Gi")
	c.Assert(err, check.IsNil)

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: volName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}
	_, err = fakeCli.CoreV1().PersistentVolumeClaims(defaultNamespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	dynCli := dynfake.NewSimpleDynamicClient(scheme)

	fakeSs := snapshot.NewSnapshotter(fakeCli, dynCli)
	ctx := context.Background()

	var volumeSnapshotGVR schema.GroupVersionResource
	var snapshotName string
	volumeSnapshotGVR = snapshot.VolSnapGVR
	snapshotName = snapshotNameBase + "-snap"
	snapshotMeta := snapshot.ObjectMeta{
		Name:      snapshotName,
		Namespace: defaultNamespace,
	}
	err = fakeSs.Create(ctx, volName, &fakeClass, false, snapshotMeta)
	c.Assert(err, check.IsNil)

	// This function should timeout
	timeout := 500 * time.Millisecond
	bgTimeout := 5 * time.Second
	// We don't have readyToUse and no error, waiting indefinitely
	err = waitOnReadyToUseWithTimeout(ctx, fakeSs, snapshotName, timeout)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*context deadline exceeded*")

	reply := waitOnReadyToUseInBackground(ctx, fakeSs, snapshotName, bgTimeout)
	setReadyStatus(c, dynCli, volumeSnapshotGVR, snapshotName, defaultNamespace)
	select {
	case err = <-reply:
		c.Assert(err, check.IsNil)
	case <-time.After(2 * time.Second):
		c.Error("timeout waiting on ready to use")
	}

	setVolumeSnapshotStatus(c, dynCli, volumeSnapshotGVR, snapshotName, defaultNamespace, nil)

	// Set non-transient error
	message := "some error"
	setErrorStatus(c, dynCli, volumeSnapshotGVR, snapshotName, defaultNamespace, message)

	// If there is non-transient error, exit right away
	err = waitOnReadyToUseWithTimeout(ctx, fakeSs, snapshotName, timeout)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*some error.*")

	// Set transient error
	message = "the object has been modified; please apply your changes to the latest version and try again"
	setErrorStatus(c, dynCli, volumeSnapshotGVR, snapshotName, defaultNamespace, message)

	// If there is a transient error, wait with exp backoff which is long
	err = waitOnReadyToUseWithTimeout(ctx, fakeSs, snapshotName, timeout)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*context deadline exceeded*")

	reply = waitOnReadyToUseInBackground(ctx, fakeSs, snapshotName, bgTimeout)
	setReadyStatus(c, dynCli, volumeSnapshotGVR, snapshotName, defaultNamespace)
	select {
	case err = <-reply:
		c.Assert(err, check.IsNil)
	case <-time.After(2 * time.Second):
		c.Error("timeout waiting on ready to use")
	}
}

// Helpers to work with volume snapshot status used in TestWaitOnReadyToUse
// ----------------------------------------------------------------------------

func waitOnReadyToUseInBackground(
	ctx context.Context,
	fakeSs snapshot.Snapshotter,
	snapshotName string,
	timeout time.Duration,
) chan error {
	reply := make(chan error)
	go func() {
		err := waitOnReadyToUseWithTimeout(ctx, fakeSs, snapshotName, timeout)
		reply <- err
	}()
	return reply
}

func waitOnReadyToUseWithTimeout(
	ctx context.Context,
	fakeSs snapshot.Snapshotter,
	snapshotName string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	err := fakeSs.WaitOnReadyToUse(deadlineCtx, snapshotName, defaultNamespace)
	return err
}

func setReadyStatus(
	c *check.C,
	dynCli *dynfake.FakeDynamicClient,
	volumeSnapshotGVR schema.GroupVersionResource,
	snapshotName string,
	namespace string,
) {
	status := make(map[string]interface{})
	status["readyToUse"] = true
	status["creationTime"] = time.Now().Format(time.RFC3339)

	setVolumeSnapshotStatus(c, dynCli, volumeSnapshotGVR, snapshotName, namespace, status)
}

func setErrorStatus(
	c *check.C,
	dynCli *dynfake.FakeDynamicClient,
	volumeSnapshotGVR schema.GroupVersionResource,
	snapshotName string,
	namespace string,
	message string,
) {
	status := make(map[string]interface{})
	status["Error"] = map[string]interface{}{
		"Message": message,
	}
	setVolumeSnapshotStatus(c, dynCli, volumeSnapshotGVR, snapshotName, namespace, status)
}

func setVolumeSnapshotStatus(
	c *check.C,
	dynCli *dynfake.FakeDynamicClient,
	volumeSnapshotGVR schema.GroupVersionResource,
	snapshotName string,
	namespace string,
	status map[string]interface{},
) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	us, err := dynCli.Resource(volumeSnapshotGVR).Namespace(namespace).Get(ctx, snapshotName, metav1.GetOptions{})
	c.Assert(err, check.IsNil)
	us.Object["status"] = status
	_, err = dynCli.Resource(volumeSnapshotGVR).Namespace(namespace).UpdateStatus(ctx, us, metav1.UpdateOptions{})
	c.Assert(err, check.IsNil)
}

// ----------------------------------------------------------------------------

func (s *SnapshotTestSuite) TestVolumeSnapshot(c *check.C) {
	if s.snapshotClass == nil {
		c.Skip("No v1 Volumesnapshotclass in the cluster")
	}
	if s.storageClassCSI == "" {
		c.Skip("No Storageclass with CSI provisioner, install CSI and create a storageclass for it")
	}
	c.Logf("snapshotclass: %s, storageclass %s", *s.snapshotClass, s.storageClassCSI)
	c.Logf("VolumeSnapshot test - source namespace: %s - target namespace: %s", s.sourceNamespace, s.targetNamespace)
	s.testVolumeSnapshot(c, s.snapshotter, s.storageClassCSI, s.snapshotClass)
}

func (s *SnapshotTestSuite) testVolumeSnapshot(c *check.C, snapshotter snapshot.Snapshotter, storageClass string, snapshotClass *string) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	size, err := resource.ParseQuantity("1Gi")
	c.Assert(err, check.IsNil)

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: volNamePrefix,
			Namespace:    s.sourceNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): size,
				},
			},
			StorageClassName: &storageClass,
		},
	}
	pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.sourceNamespace).Create(ctx, pvc, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pvc.Status.Phase == corev1.ClaimBound, nil
	})
	c.Assert(err, check.IsNil)

	snapshotName := snapshotNamePrefix + strconv.Itoa(int(time.Now().UnixNano()))
	wait := true
	label := map[string]string{
		"snapshottest": "testlabel",
	}
	annotations := map[string]string{
		"snapshotannotation": "testannotation",
	}
	snapshotMeta := snapshot.ObjectMeta{
		Name:        snapshotName,
		Namespace:   s.sourceNamespace,
		Labels:      label,
		Annotations: annotations,
	}
	err = snapshotter.Create(ctx, pvc.Name, snapshotClass, wait, snapshotMeta)
	c.Assert(err, check.IsNil)

	snap, err := snapshotter.Get(ctx, snapshotName, s.sourceNamespace)
	c.Assert(err, check.IsNil)
	c.Assert(snap.Name, check.Equals, snapshotName)
	c.Assert(snap.Status.ReadyToUse, check.NotNil)
	c.Assert(*snap.Status.ReadyToUse, check.Equals, true)

	snapList, err := snapshotter.List(ctx, s.sourceNamespace, label)
	c.Assert(err, check.IsNil)
	c.Assert(len(snapList.Items), check.Equals, 1)
	c.Assert(snapList.Items[0].Labels, check.DeepEquals, label)
	snapshotMeta = snapshot.ObjectMeta{
		Name:        snapshotName,
		Namespace:   s.sourceNamespace,
		Annotations: annotations,
	}
	err = snapshotter.Create(ctx, pvc.Name, snapshotClass, wait, snapshotMeta)
	c.Assert(err, check.NotNil)

	snapshotCloneName := snapshotName + "-clone"
	volumeCloneName := pvc.Name + "-clone"
	sizeOriginal := "1Gi"
	label = map[string]string{
		"snapshottest": "testlabel2",
	}
	contentAnnotation := map[string]string{
		"test": "value",
	}
	snapshotMeta = snapshot.ObjectMeta{
		Name:        snapshotCloneName,
		Namespace:   s.targetNamespace,
		Labels:      label,
		Annotations: annotations,
	}
	snapshotContentMeta := snapshot.ObjectMeta{
		Annotations: contentAnnotation,
	}
	err = snapshotter.Clone(ctx, snapshotName, s.sourceNamespace, wait, snapshotMeta, snapshotContentMeta)
	c.Assert(err, check.IsNil)

	snapList, err = snapshotter.List(ctx, s.targetNamespace, label)
	c.Assert(err, check.IsNil)
	c.Assert(len(snapList.Items), check.Equals, 1)
	c.Assert(snapList.Items[0].Labels, check.DeepEquals, label)

	args := &volume.CreatePVCFromSnapshotArgs{
		KubeCli:          s.cli,
		DynCli:           s.dynCli,
		Namespace:        s.targetNamespace,
		VolumeName:       volumeCloneName,
		StorageClassName: storageClass,
		SnapshotName:     snapshotCloneName,
		RestoreSize:      sizeOriginal,
		Labels:           nil,
	}
	_, err = volume.CreatePVCFromSnapshot(ctx, args)
	c.Assert(err, check.IsNil)
	_ = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.targetNamespace).Get(ctx, volumeCloneName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pvc.Status.Phase == corev1.ClaimBound, nil
	})

	// Try with a greater restore size.
	sizeNew := "2Gi"
	volumeCloneName += "-2"
	args = &volume.CreatePVCFromSnapshotArgs{
		KubeCli:          s.cli,
		DynCli:           s.dynCli,
		Namespace:        s.targetNamespace,
		VolumeName:       volumeCloneName,
		StorageClassName: storageClass,
		SnapshotName:     snapshotCloneName,
		RestoreSize:      sizeNew,
		Labels: map[string]string{
			"label1": "testLabel",
		},
	}
	_, err = volume.CreatePVCFromSnapshot(ctx, args)
	c.Assert(err, check.IsNil)
	_ = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		pvc, err = s.cli.CoreV1().PersistentVolumeClaims(s.targetNamespace).Get(ctx, volumeCloneName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		c.Assert(pvc.Labels, check.DeepEquals, args.Labels)
		return pvc.Status.Phase == corev1.ClaimBound, nil
	})

	_, err = snapshotter.Delete(ctx, snap.Name, snap.Namespace)
	c.Assert(err, check.IsNil)

	_, err = snapshotter.Delete(ctx, snap.Name, snap.Namespace)
	c.Assert(err, check.IsNil)

	_, err = snapshotter.Delete(ctx, snapshotCloneName, s.targetNamespace)
	c.Assert(err, check.IsNil)
}

func (s *SnapshotTestSuite) cleanupNamespace(c *check.C, ns string) {
	ctx := context.Background()
	pvcs, erra := s.cli.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	if erra != nil {
		c.Logf("Failed to list PVCs, Namespace: %s, Error: %v", ns, erra)
	} else {
		for _, pvc := range pvcs.Items {
			if err := s.cli.CoreV1().PersistentVolumeClaims(ns).Delete(context.TODO(), pvc.Name, metav1.DeleteOptions{}); err != nil {
				erra = err
				c.Logf("Failed to delete PVC, PVC: %s, Namespace: %s, Error: %v", pvc.Name, ns, err)
			}
		}
	}

	vss, errb := s.dynCli.Resource(snapshot.VolSnapGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
	if errb != nil {
		c.Logf("Failed to list snapshots, Namespace: %s, Error: %v", ns, errb)
	} else {
		for _, vs := range vss.Items {
			if err := s.dynCli.Resource(snapshot.VolSnapGVR).Namespace(ns).Delete(context.TODO(), vs.GetName(), metav1.DeleteOptions{}); err != nil {
				errb = err
				c.Logf("Failed to delete snapshot, Volumesnapshot: %s, Namespace %s, Error: %v", vs.GetName(), vs.GetNamespace(), err)
			}
		}
	}

	if erra != nil || errb != nil {
		c.Logf("Skipping deleting the namespace, Namespace: %s", ns)
	}

	err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{})
	if err != nil {
		c.Logf("Failed to delete namespace, Namespace: %s, Error: %v", ns, err)
	}
}

type snapshotClassTC struct {
	name             string
	annotationKey    string
	annotationValue  string
	storageClassName string
	snapClass        *unstructured.Unstructured
	testKey          string
	testValue        string
	check            check.Checker
}

func (s *SnapshotTestSuite) TestGetVolumeSnapshotClassFake(c *check.C) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotClassList"}, &unstructured.UnstructuredList{})

	dynCli := dynfake.NewSimpleDynamicClient(scheme)
	kubeCli := fake.NewSimpleClientset(
		&scv1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: fakeSC,
			},
			Provisioner: fakeDriver,
		},
		&scv1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "wrongSC",
			},
			Provisioner: "wrongDriver",
		},
	)
	fakeSs := snapshot.NewSnapshotter(kubeCli, dynCli)
	_, err := fakeSs.GetVolumeSnapshotClass(ctx, "test-annotation", "value", fakeSC)
	c.Assert(err, check.NotNil)

	for _, tc := range []snapshotClassTC{
		{
			name:             "test-1",
			annotationKey:    "test-1",
			annotationValue:  "true",
			storageClassName: fakeSC,
			snapClass:        snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, "test-1", fakeDriver, "Delete", nil),
			testKey:          "test-1",
			testValue:        "true",
			check:            check.IsNil,
		},
		{
			name:             "test-2",
			annotationKey:    "",
			annotationValue:  "",
			storageClassName: fakeSC,
			snapClass:        snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, "test-2", fakeDriver, "Delete", nil),
			testKey:          "",
			testValue:        "",
			check:            check.IsNil,
		},
		{
			name:             "test-3",
			annotationKey:    "test-3",
			annotationValue:  "false",
			storageClassName: fakeSC,
			snapClass:        snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, "test-2", fakeDriver, "Delete", nil),
			testKey:          "invalid",
			testValue:        "false",
			check:            check.NotNil,
		},
		{
			name:             "test-4",
			annotationKey:    "test-4",
			annotationValue:  "false",
			storageClassName: fakeSC,
			snapClass:        snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, "test-4", fakeDriver, "Delete", nil),
			testKey:          "test-4",
			testValue:        "true",
			check:            check.NotNil,
		},
		{
			name:             "test-5",
			annotationKey:    "test-5",
			annotationValue:  "true",
			storageClassName: "badStorageClass",
			snapClass:        snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, "test-5", fakeDriver, "Delete", nil),
			testKey:          "test-5",
			testValue:        "true",
			check:            check.NotNil,
		},
		{
			name:             "test-6",
			annotationKey:    "test-6",
			annotationValue:  "true",
			storageClassName: fakeSC,
			snapClass:        snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, "test-6", "driverMismatch", "Delete", nil),
			testKey:          "test-6",
			testValue:        "true",
			check:            check.NotNil,
		},
		{ // missing driver/snapshotter in unsturctured object
			name:             "test-7",
			annotationKey:    "test-7",
			annotationValue:  "true",
			storageClassName: fakeSC,
			snapClass: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": fmt.Sprintf("%s/%s", snapshot.GroupName, snapshot.Version),
					"kind":       snapshot.VolSnapClassKind,
					"metadata": map[string]interface{}{
						"name": "test-7",
					},
					"deletionPolicy": "Delete",
				},
			},
			testKey:   "test-7",
			testValue: "true",
			check:     check.NotNil,
		},
		{ // bad kind
			name:             "test-8",
			annotationKey:    "test-8",
			annotationValue:  "true",
			storageClassName: fakeSC,
			snapClass: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": fmt.Sprintf("%s/%s", snapshot.GroupName, snapshot.Version),
					"kind":       "bad kind",
					"metadata": map[string]interface{}{
						"name": "test-8",
					},
					"deletionPolicy": "Delete",
				},
			},
			testKey:   "test-8",
			testValue: "true",
			check:     check.NotNil,
		},
		{ // not driver string
			name:             "test-9",
			annotationKey:    "test-9",
			annotationValue:  "true",
			storageClassName: fakeSC,
			snapClass: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": fmt.Sprintf("%s/%s", snapshot.GroupName, snapshot.Version),
					"kind":       snapshot.VolSnapClassKind,
					"metadata": map[string]interface{}{
						"name": "test-9",
					},
					"deletionPolicy": "Delete",
					"driver": map[string]interface{}{
						"not": "string",
					},
				},
			},
			testKey:   "test-9",
			testValue: "true",
			check:     check.NotNil,
		},
	} {
		tc.testGetSnapshotClass(c, dynCli, fakeSs, tc.snapClass, snapshot.VolSnapClassGVR)
	}
}

func (tc snapshotClassTC) testGetSnapshotClass(c *check.C, dynCli dynamic.Interface, fakeSs snapshot.Snapshotter, snapClass *unstructured.Unstructured, gvr schema.GroupVersionResource) {
	// Add annotations
	ctx := context.Background()
	snapClass.Object["metadata"] = map[string]interface{}{
		"name": tc.name,
		"annotations": map[string]interface{}{
			tc.annotationKey: tc.annotationValue,
		},
	}
	_, err := dynCli.Resource(gvr).Create(ctx, snapClass, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	defer func() {
		err := dynCli.Resource(gvr).Delete(context.TODO(), tc.name, metav1.DeleteOptions{})
		c.Assert(err, check.IsNil)
	}()
	name, err := fakeSs.GetVolumeSnapshotClass(ctx, tc.testKey, tc.testValue, tc.storageClassName)
	c.Assert(err, tc.check, check.Commentf("%s", tc.testKey))
	if err == nil {
		c.Assert(name, check.Equals, tc.name)
	}
}

func findSnapshotClassName(c *check.C, ctx context.Context, dynCli dynamic.Interface, gvr schema.GroupVersionResource, object interface{}) (string, string) {
	// Find alpha VolumeSnapshotClass name
	us, err := dynCli.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
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

func (s *SnapshotTestSuite) TestCreateFromSource(c *check.C) {
	ctx := context.Background()
	namespace := "namespace"
	existingSnapshotName := "existingSnapname"
	snapshotName := "snapname"
	snapshotClass := "volSnapClass"
	driver := "driver"
	labels := map[string]string{"Label": "1/"}
	annotations := map[string]string{"annotationtest": "true"}
	source := &snapshot.Source{
		Handle:                  namespace,
		Driver:                  driver,
		VolumeSnapshotClassName: snapshotClass,
	}
	scheme := runtime.NewScheme()
	snapshotMeta := snapshot.ObjectMeta{
		Name:        existingSnapshotName,
		Namespace:   namespace,
		Annotations: annotations,
	}
	snapshotContentMeta := snapshot.ObjectMeta{
		Name: "content",
	}
	volSnap := snapshot.UnstructuredVolumeSnapshot(
		snapshot.VolSnapGVR,
		"pvcName",
		snapshotClass,
		snapshotMeta, snapshotContentMeta)
	volSnapClass := snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, snapshotClass, "driver", "DELETE", nil)
	dynCli := dynfake.NewSimpleDynamicClient(scheme, volSnap, volSnapClass)
	kubeCli := fake.NewSimpleClientset()
	snapshotter := snapshot.NewSnapshotter(kubeCli, dynCli)
	for _, snapshotter := range []snapshot.Snapshotter{snapshotter} {
		snapshotMeta = snapshot.ObjectMeta{
			Name:        snapshotName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		}
		err := snapshotter.CreateFromSource(ctx, source, false, snapshotMeta, snapshot.ObjectMeta{})
		c.Assert(err, check.IsNil)
		foundSns, err := snapshotter.List(ctx, namespace, labels)
		c.Assert(err, check.IsNil)
		c.Assert(foundSns.Items, check.HasLen, 1)
		c.Assert(foundSns.Items[0].Name, check.Equals, snapshotName)
	}
}

func (s *SnapshotTestSuite) TestUpdateVolumeSnapshotStatus(c *check.C) {
	ctx := context.Background()
	namespace := "namespace"
	snapshotName := "snapname"
	snapshotClass := "volSnapClass"
	annotations := map[string]string{"annotationtest": "true"}
	snapshotMeta := snapshot.ObjectMeta{
		Name:        snapshotName,
		Namespace:   namespace,
		Annotations: annotations,
	}
	snapshotContentMeta := snapshot.ObjectMeta{
		Name: "content",
	}
	volSnap := snapshot.UnstructuredVolumeSnapshot(snapshot.VolSnapGVR, "pvcName", snapshotClass,
		snapshotMeta, snapshotContentMeta)
	volSnap.Object["status"] = map[string]interface{}{
		"readyToUse": false,
	}
	scheme := runtime.NewScheme()
	dynCli := dynfake.NewSimpleDynamicClient(scheme, volSnap)
	kubeCli := fake.NewSimpleClientset()

	snapshotter, ok := snapshot.NewSnapshotter(kubeCli, dynCli).(*snapshot.Snapshot)
	c.Assert(ok, check.Equals, true)

	// set true
	err := snapshotter.UpdateVolumeSnapshotStatus(ctx, namespace, snapshotName, true)
	c.Assert(err, check.IsNil)
	us, err := dynCli.Resource(snapshot.VolSnapGVR).Namespace(namespace).Get(ctx, snapshotName, metav1.GetOptions{})
	c.Assert(err, check.IsNil)
	status, ok := us.Object["status"].(map[string]interface{})
	c.Assert(ok, check.Equals, true)
	c.Assert(status["readyToUse"], check.Equals, true)

	// set false
	err = snapshotter.UpdateVolumeSnapshotStatus(ctx, namespace, snapshotName, false)
	c.Assert(err, check.IsNil)
	us, err = dynCli.Resource(snapshot.VolSnapGVR).Namespace(namespace).Get(ctx, snapshotName, metav1.GetOptions{})
	c.Assert(err, check.IsNil)
	status, ok = us.Object["status"].(map[string]interface{})
	c.Assert(ok, check.Equals, true)
	c.Assert(status["readyToUse"], check.Equals, false)

	// status not set
	snapshotMeta = snapshot.ObjectMeta{
		Name:        snapshotName,
		Namespace:   namespace,
		Annotations: annotations,
	}
	snapshotContentMeta = snapshot.ObjectMeta{
		Name: "content",
	}
	volSnap = snapshot.UnstructuredVolumeSnapshot(snapshot.VolSnapGVR, "pvcName", snapshotClass,
		snapshotMeta, snapshotContentMeta)
	dynCli = dynfake.NewSimpleDynamicClient(scheme, volSnap)
	snapshotter, ok = snapshot.NewSnapshotter(kubeCli, dynCli).(*snapshot.Snapshot)
	c.Assert(ok, check.Equals, true)
	err = snapshotter.UpdateVolumeSnapshotStatus(ctx, namespace, snapshotName, false)
	c.Assert(err, check.NotNil)
}

func (s *SnapshotTestSuite) TestGetSnapshotClassbyAnnotation(c *check.C) {
	ctx := context.Background()
	vsc1 := snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, "vsc1", "driver", snapshot.DeletionPolicyDelete, nil)
	vsc1.SetAnnotations(map[string]string{
		"key": "value",
	})
	vsc2 := snapshot.UnstructuredVolumeSnapshotClass(snapshot.VolSnapClassGVR, "vsc2", "driver", snapshot.DeletionPolicyDelete, nil)
	sc1 := &scv1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sc1",
		},
		Provisioner: "driver",
	}
	sc2 := &scv1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sc2",
			Annotations: map[string]string{
				"key": "vsc2",
			},
		},
		Provisioner: "driver",
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotClassList"}, &unstructured.UnstructuredList{})
	for _, tc := range []struct {
		dyncli     dynamic.Interface
		kubecli    kubernetes.Interface
		gvr        schema.GroupVersionResource
		key        string
		value      string
		sc         string
		errChecker check.Checker
		retVSC     string
	}{
		{
			dyncli:     dynfake.NewSimpleDynamicClient(scheme, vsc1),
			kubecli:    fake.NewSimpleClientset(sc1),
			gvr:        snapshot.VolSnapClassGVR,
			key:        "key",
			value:      "value",
			sc:         "sc1",
			errChecker: check.IsNil,
			retVSC:     "vsc1",
		},
		{ // no vsc available
			dyncli:     dynfake.NewSimpleDynamicClient(scheme),
			kubecli:    fake.NewSimpleClientset(sc1),
			gvr:        snapshot.VolSnapClassGVR,
			key:        "key",
			value:      "value",
			sc:         "sc1",
			errChecker: check.NotNil,
		},
		{ // annotation on sc
			dyncli:     dynfake.NewSimpleDynamicClient(scheme, vsc2),
			kubecli:    fake.NewSimpleClientset(sc2),
			gvr:        snapshot.VolSnapClassGVR,
			key:        "key",
			value:      "value",
			sc:         "sc2",
			errChecker: check.IsNil,
			retVSC:     "vsc2",
		},
		{ // missing vsc
			dyncli:     dynfake.NewSimpleDynamicClient(scheme),
			kubecli:    fake.NewSimpleClientset(sc2),
			gvr:        snapshot.VolSnapClassGVR,
			key:        "key",
			value:      "value",
			sc:         "sc2",
			errChecker: check.NotNil,
		},
	} {
		vsc, err := snapshot.GetSnapshotClassbyAnnotation(ctx, tc.dyncli, tc.kubecli, tc.gvr, tc.key, tc.value, tc.sc)
		c.Check(err, tc.errChecker)
		if tc.errChecker == check.IsNil {
			c.Assert(vsc, check.Equals, tc.retVSC)
		}
	}
}

type SnapshotLocalTestSuite struct{}

var _ = check.Suite(&SnapshotLocalTestSuite{})

func (s *SnapshotLocalTestSuite) TestLabels(c *check.C) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotList"}, &unstructured.UnstructuredList{})
	ns := "namespace"
	volName := "vol1"
	snapName := "snap1"
	snapClass := "snapClass"
	fakeCli := fake.NewSimpleClientset(fakePVC(volName, ns))
	for _, tc := range []struct {
		dynCli       dynamic.Interface
		createLabels map[string]string
		listLabel    map[string]string
		errChecker   check.Checker
		numResults   int
	}{
		{
			dynCli: dynfake.NewSimpleDynamicClient(scheme),
			createLabels: map[string]string{
				"label": "1/2/3",
			},
			listLabel: map[string]string{
				"label": "1/2/3",
			},
			errChecker: check.IsNil,
			numResults: 1,
		},
		{ // nothing that matches label
			dynCli:       dynfake.NewSimpleDynamicClient(scheme),
			createLabels: map[string]string{},
			listLabel: map[string]string{
				"label": "1",
			},
			errChecker: check.IsNil,
			numResults: 0,
		},
		{ // empty labels  list everytime
			dynCli: dynfake.NewSimpleDynamicClient(scheme),
			createLabels: map[string]string{
				"label": "1",
			},
			listLabel:  map[string]string{},
			errChecker: check.IsNil,
			numResults: 1,
		},
		{ // nil lists
			dynCli:     dynfake.NewSimpleDynamicClient(scheme),
			errChecker: check.IsNil,
			numResults: 1,
		},
	} {
		for _, fakeSs := range []snapshot.Snapshotter{
			snapshot.NewSnapshotter(fakeCli, tc.dynCli),
		} {
			var err error
			var list *snapv1.VolumeSnapshotList
			snapshotMeta := snapshot.ObjectMeta{
				Name:      snapName,
				Namespace: ns,
				Labels:    tc.createLabels,
			}
			err = fakeSs.Create(ctx, volName, &snapClass, false, snapshotMeta)
			if err == nil {
				list, err = fakeSs.List(ctx, ns, tc.listLabel)
				c.Assert(len(list.Items), check.Equals, tc.numResults)
			}
			c.Check(err, tc.errChecker)
		}
	}
}
func (s *SnapshotLocalTestSuite) TestAnnotations(c *check.C) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotList"}, &unstructured.UnstructuredList{})
	ns := "namespace"
	volName := "vol1"
	snapName := "snap1"
	snapClass := "snapClass"
	fakeCli := fake.NewSimpleClientset(fakePVC(volName, ns))
	for _, tc := range []struct {
		dynCli              dynamic.Interface
		snapshotAnnotations map[string]string
		errChecker          check.Checker
	}{
		{
			dynCli: dynfake.NewSimpleDynamicClient(scheme),
			snapshotAnnotations: map[string]string{
				"annotationtest": "true",
			},
			errChecker: check.IsNil,
		},
		{ // empty annotations list
			dynCli:              dynfake.NewSimpleDynamicClient(scheme),
			snapshotAnnotations: map[string]string{},
			errChecker:          check.IsNil,
		},
		{ // annotations list matches
			dynCli: dynfake.NewSimpleDynamicClient(scheme),
			snapshotAnnotations: map[string]string{
				"annotationtest":  "true",
				"annotationtest1": "false",
			},
			errChecker: check.IsNil,
		},
		{ // nil lists
			dynCli:     dynfake.NewSimpleDynamicClient(scheme),
			errChecker: check.IsNil,
		},
	} {
		for _, fakeSs := range []snapshot.Snapshotter{
			snapshot.NewSnapshotter(fakeCli, tc.dynCli),
		} {
			var err error
			var vs *snapv1.VolumeSnapshot
			snapshotMeta := snapshot.ObjectMeta{
				Name:        snapName,
				Namespace:   ns,
				Annotations: tc.snapshotAnnotations,
			}
			err = fakeSs.Create(ctx, volName, &snapClass, false, snapshotMeta)
			if err == nil {
				vs, err = fakeSs.Get(ctx, snapName, ns)
				annotation := vs.GetAnnotations()
				c.Assert(len(annotation), check.Equals, len(tc.snapshotAnnotations))
				c.Assert(annotation, check.DeepEquals, tc.snapshotAnnotations)
			}
			c.Check(err, tc.errChecker)
		}
	}
}
func fakePVC(name, namespace string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

type SnapshotTransformUnstructuredTestSuite struct{}

var _ = check.Suite(&SnapshotTransformUnstructuredTestSuite{})

func (s *SnapshotTransformUnstructuredTestSuite) TestNilUnstructured(c *check.C) {
	err := snapshot.TransformUnstructured(nil, nil)
	c.Check(err, check.ErrorMatches, "Cannot deserialize nil unstructured")
	u := &unstructured.Unstructured{}
	err = snapshot.TransformUnstructured(u, nil)
	c.Check(err, check.ErrorMatches, "Failed to Unmarshal unstructured object: json: Unmarshal\\(nil\\)")
}
