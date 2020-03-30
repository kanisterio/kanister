// Copyright 2020 The Kanister Authors.
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

package transforms

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

// GetTransformedZones returns list of transformed zones if applicable
func GetTransformedZones(ctx context.Context, kubeCli kubernetes.Interface, currentZone string, pvc *v1.PersistentVolumeClaim) ([]string, error) {
	// fetch the storageClass

	// If storage class has a topology specification
	// get list of zones

	return []string{currentZone}, nil
}

// GetTransformedVolumeType returns transformed volume type if able. Otherwise returns orginal type
func GetTransformedVolumeType(ctx context.Context, currentVol *blockstorage.Volume, pvc *v1.PersistentVolumeClaim) (string, error) {
	// validate if provider can change vol Type

	return currentVol.VolumeType, nil
}
