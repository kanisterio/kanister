// Copyright 2022 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
)

type KubeUtilsSuite struct{}

var _ = check.Suite(&KubeUtilsSuite{})

func (s *KubeUtilsSuite) TestCreateSecretMountSpec(c *check.C) {
	existingVolMount := v1.VolumeMount{
		Name:      "volName",
		MountPath: "mntpath",
		ReadOnly:  false,
	}
	volMounts := []v1.VolumeMount{existingVolMount}
	existingVol := v1.Volume{
		Name: "vol",
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: "abc",
				ReadOnly:  false,
			},
		},
	}
	vols := []v1.Volume{existingVol}
	secretName := "test-location-secret"
	secretMount := map[string]string{locationSecretNameKey: secretName}
	volMounts, vols, err := createSecretMountSpec(secretMount, volMounts, vols)
	c.Assert(err, check.IsNil)
	expectedVolMounts := []v1.VolumeMount{
		existingVolMount,
		{
			Name:      LocationSecretVolumeMountName,
			MountPath: LocationSecretMountPath,
		},
	}
	expectedVols := []v1.Volume{
		existingVol,
		{
			Name: LocationSecretVolumeMountName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		},
	}
	c.Assert(volMounts, check.DeepEquals, expectedVolMounts)
	c.Assert(vols, check.DeepEquals, expectedVols)
}
