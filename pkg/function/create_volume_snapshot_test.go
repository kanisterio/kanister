package function

import (
	"context"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
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
		&v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-test-1",
				Namespace: ns,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				VolumeName: "pv-test-1",
			},
		},
		&v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-test-1",
				Labels: map[string]string{
					kubevolume.PVZoneLabelName:   "us-west-2a",
					kubevolume.PVRegionLabelName: "us-west-2",
				},
			},
			Spec: v1.PersistentVolumeSpec{
				Capacity: v1.ResourceList{
					v1.ResourceStorage: *k8sresource.NewQuantity(1, k8sresource.BinarySI),
				},
				PersistentVolumeSource: v1.PersistentVolumeSource{
					AWSElasticBlockStore: &v1.AWSElasticBlockStoreVolumeSource{
						VolumeID: "vol-abc123",
					},
				},
			},
		},
		&v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-test-2",
				Namespace: ns,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				VolumeName: "pv-test-2",
			},
		},
		&v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-test-2",
			},
			Spec: v1.PersistentVolumeSpec{
				Capacity: v1.ResourceList{
					v1.ResourceStorage: *k8sresource.NewQuantity(1, k8sresource.BinarySI),
				},
				PersistentVolumeSource: v1.PersistentVolumeSource{
					AWSElasticBlockStore: &v1.AWSElasticBlockStoreVolumeSource{
						VolumeID: "vol-abc123",
					},
				},
			},
		},
		&v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-test-3",
				Namespace: ns,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				VolumeName: "pv-test-3",
			},
		},
		&v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-test-3",
				Labels: map[string]string{
					kubevolume.PVZoneLabelName: "us-west-2a",
				},
			},
			Spec: v1.PersistentVolumeSpec{
				Capacity: v1.ResourceList{
					v1.ResourceStorage: *k8sresource.NewQuantity(1, k8sresource.BinarySI),
				},
			},
		},
	)
	_, err := cli.Core().PersistentVolumeClaims(ns).Get("pvc-test-1", metav1.GetOptions{})
	c.Assert(err, IsNil)
	_, err = cli.Core().PersistentVolumes().Get("pv-test-1", metav1.GetOptions{})
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
			wantSize:     int64(1),
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
