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

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/testutil/mockblockstorage"
)

type WaitForSnapshotCompletionTestSuite struct{}

var _ = Suite(&WaitForSnapshotCompletionTestSuite{})

func (s *WaitForSnapshotCompletionTestSuite) TestWaitwithRole(c *C) {
	ctx := context.Background()
	mockGetter := mockblockstorage.NewGetter()
	pvcData1 := []VolumeSnapshotInfo{
		{SnapshotID: "snap-1", Type: blockstorage.TypeEBS, Region: "us-west-2", PVCName: "pvc-1", Az: "us-west-2a", VolumeType: "ssd"},
		{SnapshotID: "snap-2", Type: blockstorage.TypeEBS, Region: "us-west-2", PVCName: "pvc-2", Az: "us-west-2a", VolumeType: "ssd"},
	}
	info, err := json.Marshal(pvcData1)
	c.Assert(err, IsNil)
	snapinfo := string(info)
	for _, tc := range []struct {
		snapshotinfo string
		check        Checker
		profile      *param.Profile
	}{
		{
			snapshotinfo: snapinfo,
			check:        IsNil,
			profile: &param.Profile{
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
		},
		{
			snapshotinfo: snapinfo,
			check:        IsNil,
			profile: &param.Profile{
				Location: crv1alpha1.Location{
					Type:   crv1alpha1.LocationTypeS3Compliant,
					Region: "us-west-2",
				},
				Credential: param.Credential{
					Type: param.CredentialTypeSecret,
					Secret: &corev1.Secret{
						Type: corev1.SecretType(secrets.AWSSecretType),
						Data: map[string][]byte{
							secrets.AWSAccessKeyID:     []byte("key-id"),
							secrets.AWSSecretAccessKey: []byte("access-key"),
							secrets.ConfigRole:         []byte("role"),
						},
					},
				},
			},
		},
	} {
		err := waitForSnapshotsCompletion(ctx, tc.snapshotinfo, tc.profile, mockGetter)
		c.Assert(err, tc.check)
	}
}
