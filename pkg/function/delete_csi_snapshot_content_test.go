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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/kube/snapshot"
)

type DeleteCSISnapshotContentTestSuite struct{}

var _ = Suite(&DeleteCSISnapshotContentTestSuite{})

func (testSuite *DeleteCSISnapshotContentTestSuite) SetUpSuite(c *C) {}

func (testSuite *DeleteCSISnapshotContentTestSuite) TestDeleteCSISnapshotContent(c *C) {
	const (
		snapshotContentName = "test-delete-snapshot-content"
		snapshotName        = "test-delete-snapshot-name"
		snapshotNamespace   = "test-delete-snapshot-namespace"
		snapshotClassName   = "test-delete-snapshot-class"
		deletionPolicy      = "Retain"
		driver              = "test-delete-driver"
		handle              = "test-delete-handle"
	)
	for _, api := range []*metav1.APIResourceList{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       snapshot.VolSnapContentKind,
				APIVersion: "v1alpha1",
			},
			GroupVersion: fmt.Sprintf("%s/v1alpha1", snapshot.GroupName),
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       snapshot.VolSnapContentKind,
				APIVersion: "v1beta1",
			},
			GroupVersion: fmt.Sprintf("%s/v1beta1", snapshot.GroupName),
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       snapshot.VolSnapContentKind,
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

		source := &snapshot.Source{
			Handle:                  snapshotNamespace,
			Driver:                  driver,
			VolumeSnapshotClassName: snapshotClassName,
		}
		fakeSnapshotContentMeta := snapshot.ObjectMeta{
			Name: snapshotContentName,
		}
		err = fakeSnapshotter.CreateContentFromSource(ctx, source, snapshotName,
			snapshotNamespace, deletionPolicy, fakeSnapshotContentMeta)
		c.Assert(err, IsNil)

		gv := strings.Split(api.GroupVersion, "/")
		gvr := schema.GroupVersionResource{
			Group:    gv[0],
			Version:  gv[1],
			Resource: snapshot.VolumeSnapshotContentResourcePlural,
		}

		_, err = dynCli.Resource(gvr).Get(ctx, snapshotContentName, metav1.GetOptions{})
		c.Assert(err, IsNil)

		err = deleteCSISnapshotContent(ctx, fakeSnapshotter, snapshotContentName)
		c.Assert(err, IsNil)

		_, err = dynCli.Resource(gvr).Get(ctx, snapshotContentName, metav1.GetOptions{})
		c.Assert(err, NotNil)
	}
}
