package zone

import (
	"context"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetZoneFromKnownNodeZones get the zone from known node zones
func GetZoneFromKnownNodeZones(ctx context.Context, sourceZone string, nzs map[string]struct{}) (string, error) {
	// If the original zone is available, we return that one.
	if _, ok := nzs[sourceZone]; ok {
		return sourceZone, nil
	}
	// If there's an available zone with the zone suffix, we use that one.
	for nz := range nzs {
		if GetZoneSuffixesMatch(nz, sourceZone) {
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

// GetZoneSuffixesMatch check if the given zones have a matching suffix
func GetZoneSuffixesMatch(zone1, zone2 string) bool {
	a1 := zone1[len(zone1)-1]
	a2 := zone2[len(zone2)-1]
	return a1 == a2
}

const (
	nodeZonesErr = `Failed to get Node availability zones.`
)

// NodeZones get the zones available for the nodes
func NodeZones(ctx context.Context) (map[string]struct{}, error) {
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
