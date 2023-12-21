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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
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
	newZones := getAvailableZones(ctx, m, kubeCli, sourceZones, sourceRegion)
	// If Kubernetes provided zones are invalid use valid sourceZones from sourceRegion
	if len(newZones) == 0 {
		validZoneNames, err := m.FromRegion(ctx, sourceRegion)
		if err != nil || len(validZoneNames) == 0 {
			return nil, errors.Wrapf(err, "No provider zones for region (%s)", sourceRegion)
		}
		for _, zone := range sourceZones {
			if isZoneValid(zone, validZoneNames) {
				newZones[zone] = struct{}{}
			}
		}
	}
	if len(newZones) == 0 {
		return nil, errors.Errorf("Unable to find valid availability zones for region (%s)", sourceRegion)
	}
	var zones []string
	for z := range newZones {
		zones = append(zones, z)
	}
	return zones, nil
}

func getAvailableZones(ctx context.Context, m Mapper, kubeCli kubernetes.Interface, sourceZones []string, sourceRegion string) map[string]struct{} {
	if kubeCli == nil {
		return map[string]struct{}{}
	}
	availableZones, availableRegion, err := NodeZonesAndRegion(ctx, kubeCli)
	if err != nil {
		log.WithError(err).Print("No available zones found")
		return map[string]struct{}{}
	}
	if availableRegion != sourceRegion {
		log.Info().Print("Source region and available region mismatch", field.M{"sourceRegion": sourceRegion, "availableRegion": availableRegion})
	}
	// TODO: validate availableRegion
	// if we fail to get zones from the available region,
	// we will defer to the source region for the zones.
	validZoneNames, err := m.FromRegion(ctx, availableRegion)
	if err != nil || len(validZoneNames) == 0 {
		return map[string]struct{}{}
	}
	sanitizedAvailableZones := SanitizeAvailableZones(availableZones, validZoneNames)
	// Add all available valid source zones
	newZones := make(map[string]struct{})
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
// There are 2 main purposes of this method. One is to avoid returning the same
// zone from a list of zones (the first one) if the sourceZone is differnet.
// The second is to have it return the same zone everytime if the sourceZone doesn't change.
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
		zone := kube.GetZoneFromNode(n)
		// make sure it is not a faultDomain
		// For Example: all non-zonal cluster nodes in azure get assigned a faultDomain(0/1)
		// for "failure-domain.beta.kubernetes.io/zone" label
		if len(zone) > 1 {
			zoneSet[zone] = struct{}{}
		}
		region := kube.GetRegionFromNode(n)
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
func GetReadySchedulableNodes(cli kubernetes.Interface) ([]corev1.Node, error) {
	ns, err := cli.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	total := len(ns.Items)
	var unschedulable, notReady int
	var l []corev1.Node
	for _, node := range ns.Items {
		switch {
		case !kube.IsNodeReady(&node):
			notReady++
		case !kube.IsNodeSchedulable(&node):
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
