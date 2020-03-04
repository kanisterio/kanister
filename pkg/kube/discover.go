package kube

import (
	"context"

	"k8s.io/client-go/discovery"
)

const osAppsGroupName = `apps.openshift.io`

// IsOSAppsGroupAvailable returns true if the openshift apps group is registered in service discovery.
func IsOSAppsGroupAvailable(ctx context.Context, cli discovery.DiscoveryInterface) (bool, error) {
	sgs, err := cli.ServerGroups()
	if err != nil {
		return false, err
	}
	for _, g := range sgs.Groups {
		if g.Name == osAppsGroupName {
			return true, nil
		}
	}
	return false, nil
}
