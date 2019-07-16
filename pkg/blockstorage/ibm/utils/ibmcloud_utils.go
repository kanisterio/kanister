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
