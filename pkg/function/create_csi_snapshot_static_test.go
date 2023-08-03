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
	"fmt"
	"strings"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/kube/snapshot"
)

type CreateCSISnapshotStaticTestSuite struct{}

var _ = Suite(&CreateCSISnapshotStaticTestSuite{})

func (testSuite *CreateCSISnapshotStaticTestSuite) SetUpSuite(c *C) {}

func (testSuite *CreateCSISnapshotStaticTestSuite) TestCreateCSISnapshotStatic(c *C) {
	const (
		snapshotName   = "test-snapshot"
		namespace      = "test-namespace"
		deletionPolicy = "Retain"
		driver         = "test-driver"
		snapshotClass  = "test-snapshot-class"
		snapshotHandle = "test-snapshot-handle"
	)

	for _, api := range []*metav1.APIResourceList{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       snapshot.VolSnapKind,
				APIVersion: "v1alpha1",
			},
			GroupVersion: fmt.Sprintf("%s/v1alpha1", snapshot.GroupName),
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       snapshot.VolSnapKind,
				APIVersion: "v1beta1",
			},
			GroupVersion: fmt.Sprintf("%s/v1beta1", snapshot.GroupName),
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       snapshot.VolSnapKind,
				APIVersion: "v1",
			},
			GroupVersion: fmt.Sprintf("%s/%s", snapshot.GroupName, snapshot.Version),
		},
	} {
		ctx := context.Background()
		fakeCli := fake.NewSimpleClientset()
		fakeCli.Resources = []*metav1.APIResourceList{api}

		scheme := runtime.NewScheme()
		dynCli := dynfake.NewSimpleDynamicClient(scheme)
		fakeSnapshotter, err := snapshot.NewSnapshotter(fakeCli, dynCli)
		c.Assert(err, IsNil)

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = fakeCli.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
		c.Assert(err, IsNil)

		gv := strings.Split(api.GroupVersion, "/")
		gvr := schema.GroupVersionResource{
			Group:    gv[0],
			Version:  gv[1],
			Resource: snapshot.VolumeSnapshotClassResourcePlural,
		}

		snapshotClass := snapshot.UnstructuredVolumeSnapshotClass(
			gvr,
			snapshotClass,
			driver,
			deletionPolicy,
			nil)
		_, err = dynCli.Resource(gvr).Create(ctx, snapshotClass, metav1.CreateOptions{})
		c.Assert(err, IsNil)

		_, err = createCSISnapshotStatic(
			ctx,
			fakeSnapshotter,
			snapshotName,
			namespace.GetName(),
			driver,
			snapshotHandle,
			snapshotClass.GetName(), false)
		c.Assert(err, IsNil)

		err = fakeCli.CoreV1().Namespaces().Delete(ctx, namespace.GetName(), metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}
