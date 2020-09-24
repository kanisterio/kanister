package kube

import (
	"context"
	"fmt"

	"k8s.io/client-go/discovery"
)

const (
	osAppsGroupName  = `apps.openshift.io`
	osRouteGroupName = `route.openshift.io`
)

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

// IsOSRouteGroupAvailable returns true is the openshift route group is registered in service discovery
func IsOSRouteGroupAvailable(ctx context.Context, cli discovery.DiscoveryInterface) (bool, error) {
	sgs, err := cli.ServerGroups()
	if err != nil {
		return false, err
	}
	for _, g := range sgs.Groups {
		if g.Name == osRouteGroupName {
			return true, nil
		}
	}
	return false, nil
}

// IsGroupVersionAvailable returns true if given group/version is registered.
func IsGroupVersionAvailable(ctx context.Context, cli discovery.DiscoveryInterface, groupName, version string) (bool, error) {
	sgs, err := cli.ServerGroups()
	if err != nil {
		return false, err
	}

	for _, g := range sgs.Groups {
		for _, v := range g.Versions {
			if fmt.Sprintf("%s/%s", groupName, version) == v.GroupVersion {
				return true, nil
			}
		}
	}
	return false, nil
}
