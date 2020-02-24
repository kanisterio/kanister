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
	"sort"

	"github.com/agnivade/levenshtein"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/pkg/errors"
	//	"github.com/schollz/closestmatch"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type (
	// Mapper interface indicates provider that supports FromRegion mapping to list of zones
	Mapper interface {
		FromRegion(ctx context.Context, region string) ([]string, error)
	}
)

// FromSourceRegionZone gets the zones from the given region and sourceZones
// It will return a minimum of 0 and a maximum of zones equal to the length of sourceZones.
// Depending on the length of the slice returned, the blockstorage providers will decide if
// a regional volume or a zonal volume should be created.
func FromSourceRegionZone(ctx context.Context, m Mapper, sourceRegion string, sourceZones ...string) ([]string, error) {
	newZones := make(map[string]struct{})
	validZoneNames, err := m.FromRegion(ctx, sourceRegion)
	if err != nil || len(validZoneNames) == 0 {
		// maybe error out right here.
		log.WithError(err).Print("Using original AZ.", field.M{"region": sourceRegion})
		return sourceZones, nil
	}
	cli, err := kube.NewClient()
	if err == nil {
		availableZones, availableRegion, err := NodeZonesAndRegion(ctx, cli)
		if err != nil {
			// log
			goto Validate
		}
		if availableRegion != sourceRegion {
			log.Error().Print("Source region and available region mismatch", field.M{"sourceRegion": sourceRegion, "availableRegion": availableRegion})
		}
		if len(availableZones) <= 0 {
			// log return nil, errors.Errorf("Could not get zone for region %s and sourceZones %s", availableRegion, sourceZones)
			goto Validate
		}

		sanitizedAvailableZones := sanitizeAvailableZones(availableZones, validZoneNames)

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
Validate:
	if len(newZones) == 0 {
		return nil, errors.Errorf("Unable to find valid availabilty zones for region (%s)", sourceRegion)
	}
	var zones []string
	for z := range newZones {
		zones = append(zones, z)
	}
	return zones, nil
}

func sanitizeAvailableZones(availableZones map[string]struct{}, validZoneNames []string) map[string]struct{} {
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

func levenshteinMatch(input string, options []string) string {
	sort.Slice(options, func(i, j int) bool {
		return levenshtein.ComputeDistance(input, options[i]) < levenshtein.ComputeDistance(input, options[j])
	})
	return options[0]
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
	ns, err := GetReadySchedulableNodes(cli)
	if err != nil {
		return nil, "", errors.Wrap(err, nodeZonesErr)
	}
	zoneSet := make(map[string]struct{}, len(ns.Items))
	regionSet := make(map[string]struct{})
	for _, n := range ns.Items {
		// For kubernetes 1.17 onwards failureDomain annotations are being deprecated
		// and will need to use topology.kubernetes.io/zone=us-east-1c and
		// topology.kubernetes.io/region=us-east-1
		// https://kubernetes.io/docs/reference/kubernetes-api/labels-annotations-taints/#topologykubernetesioregion
		if v, ok := n.Labels[kubevolume.PVZoneLabelName]; ok {
			zoneSet[v] = struct{}{}
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

// GetReadySchedulableNodes addresses the common use case of getting nodes you can do work on.
// 1) Needs to be schedulable.
// 2) Needs to be ready.
// Derived from "k8s.io/kubernetes/test/e2e/framework/node"
// TODO: check for taints as well
func GetReadySchedulableNodes(cli kubernetes.Interface) (*v1.NodeList, error) {
	ns, err := cli.CoreV1().Nodes().List(metav1.ListOptions{FieldSelector: fields.Set{
		"spec.unschedulable": "false",
	}.AsSelector().String()})
	if err != nil {
		return nil, err
	}
	Filter(ns, func(node v1.Node) bool {
		return IsNodeSchedulable(&node)
	})
	if len(ns.Items) == 0 {
		return nil, errors.New("There are currently no ready, schedulable nodes in the cluster")
	}
	return ns, nil
}

// Filter filters nodes in NodeList in place, removing nodes that do not
// satisfy the given condition
// Derived from "k8s.io/kubernetes/test/e2e/framework/node"
func Filter(nodeList *v1.NodeList, fn func(node v1.Node) bool) {
	var l []v1.Node
	for _, node := range nodeList.Items {
		if fn(node) {
			l = append(l, node)
		}
	}
	nodeList.Items = l
}

// IsNodeSchedulable returns true if:
// 1) doesn't have "unschedulable" field set
// 2) it also returns true from IsNodeReady
// Derived from "k8s.io/kubernetes/test/e2e/framework/node"
func IsNodeSchedulable(node *v1.Node) bool {
	if node == nil {
		return false
	}
	return !node.Spec.Unschedulable && IsNodeReady(node)
}

// IsNodeReady returns true if:
// 1) it's Ready condition is set to true
// Derived from "k8s.io/kubernetes/test/e2e/framework/node"
func IsNodeReady(node *v1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == v1.NodeReady {
			if cond.Status == v1.ConditionTrue {
				return true
			}
		}
	}
	return false
}
