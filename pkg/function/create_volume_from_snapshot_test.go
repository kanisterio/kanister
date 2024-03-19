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

package function

import (
	"context"
	"encoding/json"
	"fmt"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	"github.com/kanisterio/kanister/pkg/blockstorage/gcepd"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil/mockblockstorage"
)

type CreateVolumeFromSnapshotTestSuite struct{}

var _ = Suite(&CreateVolumeFromSnapshotTestSuite{})

func (s *CreateVolumeFromSnapshotTestSuite) TestCreateVolumeFromSnapshot(c *C) {
	ctx := context.Background()
	ns := "ns"
	mockGetter := mockblockstorage.NewGetter()
	profile := &param.Profile{
		Location: crv1alpha1.Location{
			Type:   crv1alpha1.LocationTypeS3Compliant,
			Region: "us-west-2",
		},
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     "foo",
				Secret: "bar",
			},
		},
	}
	cli := fake.NewSimpleClientset()
	// fake doesn't handle generated names for PVs, so ...
	var i int
	pvl := &corev1.PersistentVolumeList{}
	// kube.CreatePV() calls create() and list() which is to be handled for fake client
	cli.PrependReactor("create", "persistentvolumes",
		func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			ca := action.(testing.CreateAction)
			pv := ca.GetObject().(*corev1.PersistentVolume)
			pvl.Items = append(pvl.Items, *pv)
			if pv.ObjectMeta.Name == "" && pv.ObjectMeta.GenerateName != "" {
				pv.ObjectMeta.Name = fmt.Sprintf("%s%d", pv.ObjectMeta.GenerateName, i)
				i++
				return true, pv, nil
			}
			return false, nil, nil
		})
	cli.PrependReactor("list", "persistentvolumes",
		func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, pvl, nil
		})
	tags := []*blockstorage.KeyValue{
		{Key: "testkey", Value: "testval"},
	}
	volInfo1 := VolumeSnapshotInfo{SnapshotID: "snap-1", Type: blockstorage.TypeEBS, Region: "us-west-2", PVCName: "pvc-1", Az: "us-west-2a", Tags: tags, VolumeType: "ssd"}
	volInfo2 := VolumeSnapshotInfo{SnapshotID: "snap-2", Type: blockstorage.TypeEBS, Region: "us-west-2", PVCName: "pvc-2", Az: "us-west-2a", Tags: tags, VolumeType: "ssd"}
	var PVCData1 []VolumeSnapshotInfo
	PVCData1 = append(PVCData1, volInfo1)
	PVCData1 = append(PVCData1, volInfo2)
	info, err := json.Marshal(PVCData1)
	c.Assert(err, IsNil)
	snapinfo := string(info)
	for _, tc := range []struct {
		snapshotinfo string
		check        Checker
		newPVCs      []string
	}{
		{
			snapshotinfo: snapinfo,
			check:        IsNil,
			newPVCs:      nil,
		},
		{
			snapshotinfo: snapinfo,
			check:        IsNil,
			newPVCs:      []string{"newpvc-1", "newpvc-2"},
		},
	} {
		providerList, err := createVolumeFromSnapshot(ctx, cli, ns, tc.snapshotinfo, tc.newPVCs, profile, mockGetter)
		c.Assert(providerList, Not(Equals), tc.check)
		c.Assert(err, tc.check)
		if err != nil {
			continue
		}
		c.Assert(len(providerList) == 2, Equals, true)
		provider, ok := providerList["pvc-1"]
		c.Assert(ok, Equals, true)
		c.Assert(len(provider.(*mockblockstorage.Provider).SnapIDList) == 1, Equals, true)
		c.Assert(mockblockstorage.CheckID("snap-1", provider.(*mockblockstorage.Provider).SnapIDList), Equals, true)
		c.Assert(len(provider.(*mockblockstorage.Provider).VolIDList) == 1, Equals, true)

		provider, ok = providerList["pvc-2"]
		c.Assert(ok, Equals, true)
		c.Assert(len(provider.(*mockblockstorage.Provider).SnapIDList) == 1, Equals, true)
		c.Assert(mockblockstorage.CheckID("snap-2", provider.(*mockblockstorage.Provider).SnapIDList), Equals, true)
		c.Assert(len(provider.(*mockblockstorage.Provider).VolIDList) == 1, Equals, true)

		if tc.newPVCs != nil {
			_, err = cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "newpvc-1", metav1.GetOptions{})
			c.Assert(err, IsNil)
			_, err = cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "newpvc-2", metav1.GetOptions{})
			c.Assert(err, IsNil)
		} else {
			_, err = cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "pvc-1", metav1.GetOptions{})
			c.Assert(err, IsNil)
			_, err = cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "pvc-2", metav1.GetOptions{})
			c.Assert(err, IsNil)
		}
	}
}

func (s *CreateVolumeFromSnapshotTestSuite) TestAddPVProvisionedByAnnotation(c *C) {
	for _, tc := range []struct {
		st                  blockstorage.Provider
		annotations         map[string]string
		expectedAnnotations map[string]string
	}{
		{
			st:          &gcepd.GpdStorage{},
			annotations: nil,
			expectedAnnotations: map[string]string{
				consts.PVProvisionedByAnnotation: consts.GCEPDProvisionerInTree,
			},
		},
		{
			st: &gcepd.GpdStorage{},
			annotations: map[string]string{
				"key": "value",
			},
			expectedAnnotations: map[string]string{
				"key":                            "value",
				consts.PVProvisionedByAnnotation: consts.GCEPDProvisionerInTree,
			},
		},
		{
			st:          &gcepd.GpdStorage{},
			annotations: map[string]string{},
			expectedAnnotations: map[string]string{
				consts.PVProvisionedByAnnotation: consts.GCEPDProvisionerInTree,
			},
		},
		{
			st: &awsebs.EbsStorage{},
			annotations: map[string]string{
				"keyone": "valueone",
				"keytwo": "valuetwo",
			},
			expectedAnnotations: map[string]string{
				"keyone":                         "valueone",
				"keytwo":                         "valuetwo",
				consts.PVProvisionedByAnnotation: consts.AWSEBSProvisionerInTree,
			},
		},
	} {
		op := addPVProvisionedByAnnotation(tc.annotations, tc.st)
		c.Assert(op, DeepEquals, tc.expectedAnnotations)
	}
}
