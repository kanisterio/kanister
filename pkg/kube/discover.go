package kube

import (
	"context"
	"fmt"

	"k8s.io/client-go/discovery"
)

const (
	osAppsGroupName  = `apps.openshift.io`
	osRouteGroupName = `route.openshift.io`

	groupVersionFormat = "%s/%s"
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

// IsResAvailableInGroupVersion takes a resource and checks if that exists in the passed group and version
func IsResAvailableInGroupVersion(ctx context.Context, cli discovery.DiscoveryInterface, groupName, version, resource string) (bool, error) {
	// This call is going to fail with error type `*discovery.ErrGroupDiscoveryFailed` if there are
	// some api-resources that are served by aggregated API server and the aggregated API server is not ready.
	// So if this utility is being called for those api-resources, `false` would be returned
	resList, err := cli.ServerPreferredResources()
	if err != nil {
		if _, ok := err.(*discovery.ErrGroupDiscoveryFailed); !ok {
			return false, err
		}
	}

	gv := fmt.Sprintf(groupVersionFormat, groupName, version)
	for _, res := range resList {
		for _, r := range res.APIResources {
			if r.Name == resource && gv == res.GroupVersion {
				return true, nil
			}
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
			if fmt.Sprintf(groupVersionFormat, groupName, version) == v.GroupVersion {
				return true, nil
			}
		}
	}
	return false, nil
}
