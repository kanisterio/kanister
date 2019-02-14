package awsebs

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
)

func zoneForVolumeCreateFromSnapshot(ctx context.Context, region string, sourceZone string) (string, error) {
	nzs, err := nodeZones(ctx)
	if err != nil {
		log.Errorf("Ignoring error getting Node availability zones. Error: %+v", err)
	}
	if len(nzs) != 0 {
		var z string
		if z, err = zoneFromKnownNodeZones(ctx, region, sourceZone, nzs); err == nil && isZoneValid(z, region) {
			return z, nil
		}
	}
	return zoneWithUnknownNodeZones(ctx, region, sourceZone)
}

func isZoneValid(zone, region string) bool {
	if validZones, err := staticRegionToZones(region); err == nil {
		for _, z := range validZones {
			if zone == z {
				return true
			}
		}
	}
	return false
}

func zoneFromKnownNodeZones(ctx context.Context, region string, sourceZone string, nzs map[string]struct{}) (string, error) {
	// If the original zone is available, we return that one.
	if _, ok := nzs[sourceZone]; ok {
		return sourceZone, nil
	}
	// If there's an available zone with the zone suffix, we use that one.
	for nz := range nzs {
		if zoneSuffixesMatch(nz, sourceZone) {
			return nz, nil
		}
	}
	// If any nodes are available, return an arbitrary one.
	// This is relatively random based on go's map iteration.
	for nz := range nzs {
		return nz, nil
	}
	// Unreachable
	return "", nil
}

func zoneWithUnknownNodeZones(ctx context.Context, region string, sourceZone string) (string, error) {
	// We could not the zones of the nodes, so we return an arbitrary one.
	zs, err := regionToZones(ctx, region)
	if err != nil || len(zs) == 0 {
		// If all else fails, we return the original AZ.
		log.Errorf("Using original AZ. region: %s, Error: %+v", region, err)
		return sourceZone, nil
	}
	// We look for a zone with the same suffix.
	for _, z := range zs {
		if zoneSuffixesMatch(z, sourceZone) {
			return z, nil
		}
	}
	// We return an arbitrary zone in the region.
	return zs[0], nil
}

func zoneSuffixesMatch(zone1, zone2 string) bool {
	return zone1[len(zone1)-1] == zone2[len(zone2)-1]
}

const (
	nodeZonesErr = `Failed to get Node availability zones.`
)

func nodeZones(ctx context.Context) (map[string]struct{}, error) {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return nil, errors.Wrap(err, nodeZonesErr)
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, nodeZonesErr)
	}
	ns, err := cli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, nodeZonesErr)
	}
	zoneSet := make(map[string]struct{}, len(ns.Items))
	for _, n := range ns.Items {
		if v, ok := n.Labels[kube.PVZoneLabelName]; ok {
			zoneSet[v] = struct{}{}
		}
	}
	return zoneSet, nil
}
