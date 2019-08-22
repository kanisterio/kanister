// Copyright 2019 Kasten Inc.
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

package ibmutils

// IBM Cloud utils

import (
	"context"

	ibmprov "github.com/IBM/ibmcloud-storage-volume-lib/lib/provider"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kanisterio/kanister/pkg/kube"
)

// AuthorizeSoftLayerFileHosts some of volumes are required post creation authorization to be mounted
func AuthorizeSoftLayerFileHosts(ctx context.Context, vol *ibmprov.Volume, slCli ibmprov.Session) error {
	k8scli, err := kube.NewClient()
	if err != nil {
		return errors.Wrap(err, "Failed to created k8s client.")
	}

	nodes, err := k8scli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to list nodes")
	}

	nodeips := []string{}

	for _, node := range nodes.Items {
		for _, ip := range node.Status.Addresses {
			if ip.Type == v1.NodeInternalIP {
				nodeips = append(nodeips, ip.Address)
			}
		}
	}

	return slCli.AuthorizeVolume(ibmprov.VolumeAuthorization{
		Volume:  *vol,
		HostIPs: nodeips,
	})
}
