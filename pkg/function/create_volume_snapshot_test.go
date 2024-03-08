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

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	kube "github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil/mockblockstorage"
)

type CreateVolumeSnapshotTestSuite struct{}

var _ = Suite(&CreateVolumeSnapshotTestSuite{})

func (s *CreateVolumeSnapshotTestSuite) TestGetPVCInfo(c *C) {
	ctx := context.Background()
	ns := "ns"
	mockGetter := mockblockstorage.NewGetter()
	tp := param.TemplateParams{
		Profile: &param.Profile{
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
		},
	}
	cli := fake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-test-1",
				Namespace: ns,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "pv-test-1",
			},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-test-1",
				Labels: map[string]string{
					kube.TopologyZoneLabelName:   "us-west-2a",
					kube.TopologyRegionLabelName: "us-west-2",
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): k8sresource.MustParse("1Gi"),
				},
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					AWSElasticBlockStore: &corev1.AWSElasticBlockStoreVolumeSource{
						VolumeID: "vol-abc123",
					},
				},
			},
		},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-test-2",
				Namespace: ns,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "pv-test-2",
			},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-test-2",
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): k8sresource.MustParse("1Gi"),
				},
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					AWSElasticBlockStore: &corev1.AWSElasticBlockStoreVolumeSource{
						VolumeID: "vol-abc123",
					},
				},
			},
		},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-test-3",
				Namespace: ns,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "pv-test-3",
			},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-test-3",
				Labels: map[string]string{
					kube.TopologyZoneLabelName: "us-west-2a",
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): k8sresource.MustParse("1Gi"),
				},
			},
		},
	)
	_, err := cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "pvc-test-1", metav1.GetOptions{})
	c.Assert(err, IsNil)
	_, err = cli.CoreV1().PersistentVolumes().Get(ctx, "pv-test-1", metav1.GetOptions{})
	c.Assert(err, IsNil)

	for _, tc := range []struct {
		pvc          string
		wantVolumeID string
		wantType     blockstorage.Type
		wantVolZone  string
		wantPVC      string
		wantSize     int64
		wantRegion   string
		check        Checker
	}{
		{
			pvc:          "pvc-test-1",
			wantVolumeID: "vol-abc123",
			wantType:     blockstorage.TypeEBS,
			wantVolZone:  "us-west-2a",
			wantPVC:      "pvc-test-1",
			wantSize:     int64(1073741824),
			wantRegion:   "us-west-2",
			check:        IsNil,
		},
		{
			pvc:   "pvc-test-2",
			check: NotNil,
		},
		{
			pvc:   "pvc-test-3",
			check: NotNil,
		},
	} {
		volInfo, err := getPVCInfo(ctx, cli, ns, tc.pvc, tp, mockGetter)
		c.Assert(err, tc.check)
		c.Assert(volInfo, Not(Equals), tc.check)
		if err != nil {
			continue
		}
		c.Assert(volInfo.volumeID, Equals, tc.wantVolumeID)
		c.Assert(volInfo.sType, Equals, tc.wantType)
		c.Assert(volInfo.volZone, Equals, tc.wantVolZone)
		c.Assert(volInfo.pvc, Equals, tc.wantPVC)
		c.Assert(volInfo.size, Equals, tc.wantSize)
		c.Assert(volInfo.region, Equals, tc.wantRegion)
	}
}
