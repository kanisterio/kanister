// Copyright 2019 The Kanister Authors.
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

package zone

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
	"github.com/kanisterio/kanister/pkg/log"
)

type (
	// Mapper interface indicates provider that supports FromRegion mapping to list of zones
	Mapper interface {
		FromRegion(ctx context.Context, region string) ([]string, error)
	}
)

// FromSourceRegionZone gets the zones from the given region and sourceZones
// It will return a minimum of 0 and a maximum of zones equal to the length zones available from kubernetes.
// Depending on the length of the slice returned, the blockstorage providers will decide if
// a regional volume or a zonal volume should be created.
func FromSourceRegionZone(ctx context.Context, m Mapper, kubeCli kubernetes.Interface, sourceRegion string, sourceZones ...string) ([]string, error) {
	newZones := make(map[string]struct{})
	validZoneNames, err := m.FromRegion(ctx, sourceRegion)
	if err != nil || len(validZoneNames) == 0 {
		return nil, errors.Wrapf(err, "No provider zones for region (%s)", sourceRegion)
	}
	if !isNil(kubeCli) {
		getAvailableZones(ctx, newZones, kubeCli, validZoneNames, sourceZones, sourceRegion)
	}
	// If Kubernetes provided zones are invalid use valid sourceZones
	if len(newZones) == 0 {
		log.Info().Print("Validating source zones")
		for _, zone := range sourceZones {
			if isZoneValid(zone, validZoneNames) {
				newZones[zone] = struct{}{}
			}
		}
	}
	if len(newZones) == 0 {
		return nil, errors.Errorf("Unable to find valid availabilty zones for region (%s)", sourceRegion)
	}
	var zones []string
	for z := range newZones {
		zones = append(zones, z)
	}
	return zones, nil
}

func isNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}

func getAvailableZones(ctx context.Context, newZones map[string]struct{}, kubeCli kubernetes.Interface, validZoneNames []string, sourceZones []string, sourceRegion string) {
	availableZones, availableRegion, err := NodeZonesAndRegion(ctx, kubeCli)
	if err != nil {
		log.Info().Print("No available zones found", field.M{"error": err.Error()})
		return
	}
	if availableRegion != sourceRegion {
		log.Info().Print("Source region and available region mismatch", field.M{"sourceRegion": sourceRegion, "availableRegion": availableRegion})
	}
	if len(availableZones) <= 0 { // Will never occur, NodeZonesAndRegion returns error if empty
		log.Info().Print("No available zones found", field.M{"availableRegion": availableRegion})
		return
	}
	sanitizedAvailableZones := SanitizeAvailableZones(availableZones, validZoneNames)
	// Add all available valid source zones
	for _, zone := range sourceZones {
		z := getZoneFromKnownNodeZones(zone, sanitizedAvailableZones)
		if z != "" {
			newZones[z] = struct{}{}
		}
	}
	// If source zones aren't available and valid add all valid available zones
	if len(newZones) == 0 {
		for zone := range sanitizedAvailableZones {
			newZones[zone] = struct{}{}
		}
	}
}

func isZoneValid(zone string, validZones []string) bool {
	for _, z := range validZones {
		if zone == z {
			return true
		}
	}
	return false
}

// If the original zone is available, we return that one.
func getZoneFromKnownNodeZones(sourceZone string, availableZones map[string]struct{}) string {
	if _, ok := availableZones[sourceZone]; ok {
		return sourceZone
	}
	return ""
}

const (
	nodeZonesErr = `Failed to get Node availability zones.`
)

// NodeZonesAndRegion returns cloud provider failure-domain region and zones as reported by K8s
func NodeZonesAndRegion(ctx context.Context, cli kubernetes.Interface) (map[string]struct{}, string, error) {
	if cli == nil {
		return nil, "", errors.New(nodeZonesErr)
	}
	ns, err := cli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, "", errors.Wrap(err, nodeZonesErr)
	}
	zoneSet := make(map[string]struct{})
	regionSet := make(map[string]struct{})
	for _, n := range ns.Items {
		if v, ok := n.Labels[kubevolume.PVZoneLabelName]; ok {
			// make sure it is not a faultDomain
			if len(v) > 1 {
				zoneSet[v] = struct{}{}
			}
		}
		if v, ok := n.Labels[kubevolume.PVRegionLabelName]; ok {
			regionSet[v] = struct{}{}
		}
	}
	if len(regionSet) > 1 {
		return nil, "", errors.New("Multiple failure domain regions found")
	}
	if len(regionSet) == 0 {
		return nil, "", errors.New("No failure domain regions found")
	}
	var region []string
	for r := range regionSet {
		region = append(region, r)
	}
	return zoneSet, region[0], nil
}

// SanitizeAvailableZones validates and updates a map of zones against a list of valid zone names
func SanitizeAvailableZones(availableZones map[string]struct{}, validZoneNames []string) map[string]struct{} {
	sanitizedZones := map[string]struct{}{}
	for zone := range availableZones {
		if isZoneValid(zone, validZoneNames) {
			sanitizedZones[zone] = struct{}{}
		} else {
			closestMatch := levenshteinMatch(zone, validZoneNames)
			log.Debug().Print("Exact match not found for available zone, using closest match",
				field.M{"availableZone": zone, "closestMatch": closestMatch})
			sanitizedZones[closestMatch] = struct{}{}
		}
	}
	return sanitizedZones
}
