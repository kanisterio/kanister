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
	"hash/fnv"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
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
// It will return a minimum of 1 and a maximum of zones equal to the length of sourceZones
// If no zones are found it will return an error.
// Depending on the length of the slice returned, the blockstorage providers will decide if
// a regional volume or a zonal volume should be created.
func FromSourceRegionZone(ctx context.Context, m Mapper, kubeCli kubernetes.Interface, sourceRegion string, sourceZones ...string) ([]string, error) {
	validZoneNames, err := m.FromRegion(ctx, sourceRegion)
	if err != nil || len(validZoneNames) == 0 {
		return nil, errors.Wrapf(err, "No provider zones for region (%s)", sourceRegion)
	}
	newZones := getAvailableZones(ctx, kubeCli, validZoneNames, sourceZones, sourceRegion)
	// If Kubernetes provided zones are invalid use valid sourceZones
	if len(newZones) == 0 {
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

func getAvailableZones(ctx context.Context, kubeCli kubernetes.Interface, validZoneNames []string, sourceZones []string, sourceRegion string) map[string]struct{} {
	if kubeCli == nil {
		return map[string]struct{}{}
	}
	newZones := make(map[string]struct{})
	availableZones, availableRegion, err := NodeZonesAndRegion(ctx, kubeCli)
	if err != nil {
		log.WithError(err).Print("No available zones found")
		return map[string]struct{}{}
	}
	if availableRegion != sourceRegion {
		log.Info().Print("Source region and available region mismatch", field.M{"sourceRegion": sourceRegion, "availableRegion": availableRegion})
	}
	if len(availableZones) <= 0 { // Will never occur, NodeZonesAndRegion returns error if empty
		log.Info().Print("No available zones found", field.M{"availableRegion": availableRegion})
		return map[string]struct{}{}
	}
	sanitizedAvailableZones := SanitizeAvailableZones(availableZones, validZoneNames)
	// Add all available valid source zones
	for _, zone := range sourceZones {
		if z := getZoneFromAvailableZones(zone, sanitizedAvailableZones); z != "" {
			newZones[z] = struct{}{}
		}
	}
	// If source zones aren't available get consistent zone from available zones
	if len(newZones) == 0 {
		for _, zone := range sourceZones {
			if z := consistentZone(zone, sanitizedAvailableZones); z != "" {
				newZones[z] = struct{}{}
			}
		}
	}
	return newZones
}

// consistentZone will return the same zone given a source and a list of zones.
// This output however can change if the list changes, which would impact
// multi volume restores.
func consistentZone(sourceZone string, availableZones map[string]struct{}) string {
	// shouldn't hit this case as we catch it in the caller
	if len(availableZones) == 0 {
		return ""
	}
	s := make([]string, 0, len(availableZones))
	for zone := range availableZones {
		s = append(s, zone)
	}
	sort.Slice(s, func(i, j int) bool {
		return strings.Compare(s[i], s[j]) < 0
	})
	h := fnv.New32()
	if _, err := h.Write([]byte(sourceZone)); err != nil {
		log.Info().Print("failed to hash source zone", field.M{"sourceZone": sourceZone})
		return ""
	}
	i := int(h.Sum32()) % len(availableZones)
	return s[i]
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
func getZoneFromAvailableZones(sourceZone string, availableZones map[string]struct{}) string {
	if _, ok := availableZones[sourceZone]; ok {
		return sourceZone
	}
	return ""
}

const (
	nodeZonesErr = `Failed to get Node availability zones`
)

// NodeZonesAndRegion returns cloud provider failure-domain region and zones as reported by K8s
func NodeZonesAndRegion(ctx context.Context, cli kubernetes.Interface) (map[string]struct{}, string, error) {
	if cli == nil {
		return nil, "", errors.New(nodeZonesErr)
	}
	ns, err := GetReadySchedulableNodes(cli)
	if err != nil {
		return nil, "", errors.Wrap(err, nodeZonesErr)
	}
	zoneSet := make(map[string]struct{})
	regionSet := make(map[string]struct{})
	for _, n := range ns {
		zone := kubevolume.GetZoneFromNode(n)
		if zone != "" {
			zoneSet[zone] = struct{}{}
		}
		region := kubevolume.GetRegionFromNode(n)
		if region != "" {
			regionSet[region] = struct{}{}
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

// GetReadySchedulableNodes addresses the common use case of getting nodes you can do work on.
// 1) Needs to be schedulable.
// 2) Needs to be ready.
// Derived from "k8s.io/kubernetes/test/e2e/framework/node"
// TODO: check for taints as well
func GetReadySchedulableNodes(cli kubernetes.Interface) ([]v1.Node, error) {
	ns, err := cli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	total := len(ns.Items)
	var unschedulable, notReady int
	var l []v1.Node
	for _, node := range ns.Items {
		switch {
		case !isNodeReady(&node):
			notReady++
		case !isNodeSchedulable(&node):
			unschedulable++
		default:
			l = append(l, node)
		}
	}
	log.Info().Print("Available nodes status", field.M{"total": total, "unschedulable": unschedulable, "notReady": notReady})
	if len(l) == 0 {
		return nil, errors.New("There are currently no ready, schedulable nodes in the cluster")
	}
	return l, nil
}

// isNodeSchedulable returns true if:
// 1) doesn't have "unschedulable" field set
// 2) it also returns true from IsNodeReady
// Derived from "k8s.io/kubernetes/test/e2e/framework/node"
func isNodeSchedulable(node *v1.Node) bool {
	if node == nil {
		return false
	}
	return !node.Spec.Unschedulable
}

// isNodeReady returns true if:
// 1) it's Ready condition is set to true
// Derived from "k8s.io/kubernetes/test/e2e/framework/node"
func isNodeReady(node *v1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == v1.NodeReady {
			if cond.Status == v1.ConditionTrue {
				return true
			}
		}
	}
	return false
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
