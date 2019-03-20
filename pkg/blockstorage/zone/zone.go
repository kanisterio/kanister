package zone

import (
	"context"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
)

type (
	Mapper interface {
		FromRegion(ctx context.Context, region string) ([]string, error)
	}
)

// FromSourceRegionZone gets the zones from the given region and sourceZome
func FromSourceRegionZone(ctx context.Context, m Mapper, region string, sourceZone string) (string, error) {
	nzs, err := nodeZones(ctx)
	if err != nil {
		log.Errorf("Ignoring error getting Node availability zones. Error: %+v", err)
	}
	if len(nzs) != 0 {
		var z string
		if z, err = getZoneFromKnownNodeZones(ctx, sourceZone, nzs); err == nil && isZoneValid(ctx, m, z, region) {
			return z, nil
		}
	}
	return WithUnknownNodeZones(ctx, m, region, sourceZone)
}

func isZoneValid(ctx context.Context, m Mapper, zone, region string) bool {
	if validZones, err := m.FromRegion(ctx, region); err == nil {
		for _, z := range validZones {
			if zone == z {
				return true
			}
		}
	}
	return false
}

// WithUnknownNodeZones get the zone list  for the region
func WithUnknownNodeZones(ctx context.Context, m Mapper, region string, sourceZone string) (string, error) {
	// We could not the zones of the nodes, so we return an arbitrary one.
	zs, err := m.FromRegion(ctx, region)
	if err != nil || len(zs) == 0 {
		// If all else fails, we return the original AZ.
		log.Errorf("Using original AZ. region: %s, Error: %+v", region, err)
		return sourceZone, nil
	}
	// We look for a zone with the same suffix.
	for _, z := range zs {
		if getZoneSuffixesMatch(z, sourceZone) {
			return z, nil
		}
	}
	// We return an arbitrary zone in the region.
	return zs[0], nil
}

func getZoneFromKnownNodeZones(ctx context.Context, sourceZone string, nzs map[string]struct{}) (string, error) {
	// If the original zone is available, we return that one.
	if _, ok := nzs[sourceZone]; ok {
		return sourceZone, nil
	}
	// If there's an available zone with the zone suffix, we use that one.
	for nz := range nzs {
		if getZoneSuffixesMatch(nz, sourceZone) {
			return nz, nil
		}
	}
	// If any nodes are available, return an arbitrary one.
	return consistentZone(sourceZone, nzs)
}

func consistentZone(sourceZone string, nzs map[string]struct{}) (string, error) {
	if len(nzs) == 0 {
		return "", errors.New("could not restore volume: no zone found")
	}
	s := make([]string, 0, len(nzs))
	for nz := range nzs {
		s = append(s, nz)
	}
	sort.Slice(s, func(i, j int) bool {
		return strings.Compare(s[i], s[j]) < 0
	})
	h := fnv.New32()
	if _, err := h.Write([]byte(sourceZone)); err != nil {
		return "", errors.Errorf("failed to hash source zone %s: %s", sourceZone, err.Error())
	}
	i := int(h.Sum32()) % len(nzs)
	return s[i], nil
}

func getZoneSuffixesMatch(zone1, zone2 string) bool {
	if zone1 == "" || zone2 == "" {
		return zone1 == zone2
	}
	a1 := zone1[len(zone1)-1]
	a2 := zone2[len(zone2)-1]
	return a1 == a2
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
		if v, ok := n.Labels[kubevolume.PVZoneLabelName]; ok {
			zoneSet[v] = struct{}{}
		}
	}
	return zoneSet, nil
}
