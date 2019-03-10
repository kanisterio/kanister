package awsebs

import (
	"context"

	"github.com/kanisterio/kanister/pkg/blockstorage/zone"
	log "github.com/sirupsen/logrus"
)

func zoneForVolumeCreateFromSnapshot(ctx context.Context, region string, sourceZone string) (string, error) {
	nzs, err := zone.NodeZones(ctx)
	if err != nil {
		log.Errorf("Ignoring error getting Node availability zones. Error: %+v", err)
	}
	if len(nzs) != 0 {
		var z string
		if z, err = zone.GetZoneFromKnownNodeZones(ctx, sourceZone, nzs); err == nil && isZoneValid(z, region) {
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
		if zone.GetZoneSuffixesMatch(z, sourceZone) {
			return z, nil
		}
	}
	// We return an arbitrary zone in the region.
	return zs[0], nil
}
