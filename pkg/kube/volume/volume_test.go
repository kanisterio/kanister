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

package volume

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func Test(t *testing.T) { TestingT(t) }

type TestVolSuite struct{}

var _ = Suite(&TestVolSuite{})

func (s *TestVolSuite) TestCreatePVC(c *C) {
	// Create PVC
	ctx := context.Background()
	pvcSize := int64(1024)
	ns := "kanister-pvc-test"
	targetVolID := "testVolID"
	annotations := map[string]string{"a1": "foo"}
	cli := fake.NewSimpleClientset()
	pvcName, err := CreatePVC(ctx, cli, ns, NoPVCNameSpecified, pvcSize, targetVolID, annotations, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, nil)
	c.Assert(err, IsNil)
	pvc, err := cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvcName, metav1.GetOptions{})
	c.Assert(err, IsNil)

	c.Assert(len(pvc.Spec.AccessModes) >= 1, Equals, true)
	accessMode := pvc.Spec.AccessModes[0]
	c.Assert(accessMode, Equals, corev1.ReadWriteOnce)
	capacity, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	c.Assert(ok, Equals, true)
	c.Assert(capacity.Value() >= int64(pvcSize), Equals, true)
	eq := reflect.DeepEqual(annotations, pvc.ObjectMeta.Annotations)
	c.Assert(eq, Equals, true)
	c.Assert(len(pvc.Spec.Selector.MatchLabels) >= 1, Equals, true)
	label := pvc.Spec.Selector.MatchLabels[pvMatchLabelName]
	c.Assert(label, Equals, filepath.Base(targetVolID))

	volumeMode := corev1.PersistentVolumeBlock
	_, err = CreatePVC(ctx, cli, ns, "pvc2", pvcSize, targetVolID, annotations, nil, &volumeMode)
	c.Assert(err, IsNil)
	pvc2, err := cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "pvc2", metav1.GetOptions{})
	c.Assert(err, IsNil)
	c.Assert(len(pvc2.Spec.AccessModes) >= 1, Equals, true)
	c.Assert(*pvc2.Spec.VolumeMode, Equals, corev1.PersistentVolumeBlock)
	c.Assert(pvc2.GetAnnotations(), NotNil)
	c.Assert(pvc2.GetAnnotations()["a1"], Equals, "foo")
}

func (s *TestVolSuite) TestGetPVCRestoreSize(c *C) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotList"}, &unstructured.UnstructuredList{})
	fakeCli := fake.NewSimpleClientset()
	fakeCli.Resources = []*metav1.APIResourceList{{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VolumeSnapshot",
			APIVersion: "v1",
		},
		GroupVersion: "snapshot.storage.k8s.io/v1",
	}}
	for _, tc := range []struct {
		args       *CreatePVCFromSnapshotArgs
		sizeValue  int64
		errChecker Checker
	}{
		{ // only snapshot restore size
			args: &CreatePVCFromSnapshotArgs{
				KubeCli: fakeCli,
				DynCli: dynfake.NewSimpleDynamicClient(scheme,
					s.fakeUnstructuredSnasphotWSize("vsName", "vsNamespace", "10Gi")),
				SnapshotName: "vsName",
				Namespace:    "vsNamespace",
			},
			sizeValue:  10737418240,
			errChecker: IsNil,
		},
		{ // only args restore size
			args: &CreatePVCFromSnapshotArgs{
				KubeCli: fakeCli,
				DynCli: dynfake.NewSimpleDynamicClient(scheme,
					s.fakeUnstructuredSnasphotWSize("vsName", "vsNamespace", "")),
				SnapshotName: "vsName",
				Namespace:    "vsNamespace",
				RestoreSize:  "10Gi",
			},
			sizeValue:  10737418240,
			errChecker: IsNil,
		},
		{ // neither
			args: &CreatePVCFromSnapshotArgs{
				KubeCli: fakeCli,
				DynCli: dynfake.NewSimpleDynamicClient(scheme,
					s.fakeUnstructuredSnasphotWSize("vsName", "vsNamespace", "")),
				SnapshotName: "vsName",
				Namespace:    "vsNamespace",
			},
			errChecker: NotNil,
		},
		{ // both, snapshot size is bigger
			args: &CreatePVCFromSnapshotArgs{
				KubeCli: fakeCli,
				DynCli: dynfake.NewSimpleDynamicClient(scheme,
					s.fakeUnstructuredSnasphotWSize("vsName", "vsNamespace", "10Gi")),
				SnapshotName: "vsName",
				Namespace:    "vsNamespace",
				RestoreSize:  "9Gi",
			},
			sizeValue:  10737418240,
			errChecker: IsNil,
		},
		{ // both, args size is bigger
			args: &CreatePVCFromSnapshotArgs{
				KubeCli: fakeCli,
				DynCli: dynfake.NewSimpleDynamicClient(scheme,
					s.fakeUnstructuredSnasphotWSize("vsName1", "vsNamespace1", "9Gi")),
				SnapshotName: "vsName1",
				Namespace:    "vsNamespace1",
				RestoreSize:  "10Gi",
			},
			sizeValue:  10737418240,
			errChecker: IsNil,
		},
		{ // Failed to find snapshot
			args: &CreatePVCFromSnapshotArgs{
				KubeCli:      fakeCli,
				DynCli:       dynfake.NewSimpleDynamicClient(scheme),
				SnapshotName: "vsName",
				Namespace:    "vsNamespace",
			},
			errChecker: NotNil,
		},
		{ // Failed to create snapshotter
			args: &CreatePVCFromSnapshotArgs{
				KubeCli:      fake.NewSimpleClientset(), // fails to find dynamic api
				DynCli:       dynfake.NewSimpleDynamicClient(scheme),
				SnapshotName: "vsName",
				Namespace:    "vsNamespace",
			},
			errChecker: NotNil,
		},
		{ // bad args restore size
			args: &CreatePVCFromSnapshotArgs{
				SnapshotName: "vsName",
				Namespace:    "vsNamespace",
				RestoreSize:  "10wut",
			},
			errChecker: NotNil,
		},
	} {
		q, err := getPVCRestoreSize(ctx, tc.args)
		c.Assert(err, tc.errChecker)
		if tc.errChecker == IsNil {
			c.Assert(q.Value(), Equals, tc.sizeValue)
		}
	}
}

func (s *TestVolSuite) fakeUnstructuredSnasphotWSize(vsName, namespace, size string) *unstructured.Unstructured {
	gvr := schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots"}
	Object := map[string]interface{}{
		"apiVersion": fmt.Sprintf("%s/%s", gvr.Group, gvr.Version),
		"kind":       "VolumeSnapshot",
		"metadata": map[string]interface{}{
			"name":      vsName,
			"namespace": namespace,
		},
	}
	if size != "" {
		q := resource.MustParse(size)
		Object["status"] = map[string]interface{}{
			"restoreSize": q.ToUnstructured(),
		}
	}
	return &unstructured.Unstructured{Object: Object}
}

func (s *TestVolSuite) TestZoneToRegion(c *C) {
	for idx, tc := range []struct {
		zone           string
		expectedRegion []string
	}{
		{
			zone:           "us-west1-b",
			expectedRegion: []string{"us-west1"},
		},
		{
			zone:           "us-west1-a",
			expectedRegion: []string{"us-west1"},
		},
		{
			zone:           "us-west2-c",
			expectedRegion: []string{"us-west2"},
		},
		{
			zone:           "us-west1-a__us-west2-b",
			expectedRegion: []string{"us-west1", "us-west2"},
		},
		{
			zone:           "us-west1-a__us-west2-b__us-west2-c",
			expectedRegion: []string{"us-west1", "us-west2"},
		},
		{
			zone:           "us-west1-a__us-west1-b__us-west2-b__us-west2-c",
			expectedRegion: []string{"us-west1", "us-west2"},
		},
		{
			zone:           "us-west1-a__us-west1-b__us-west2-b__us-west2-c__us-west2-d",
			expectedRegion: []string{"us-west1", "us-west2"},
		},
	} {
		reg := zonesToRegions(tc.zone)
		c.Assert(reg, DeepEquals, tc.expectedRegion, Commentf("Case #%d", idx))
	}
}
