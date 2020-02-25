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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
	"github.com/kanisterio/kanister/pkg/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
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
func FromSourceRegionZone(ctx context.Context, m Mapper, region string, sourceZones ...string) ([]string, error) {
	newZones := make(map[string]struct{})
	cli, err := kube.NewClient()
	if err == nil {
		nzs, rs, errzr := NodeZonesAndRegion(ctx, cli)
		if errzr != nil {
			log.WithError(errzr).Print("Ignoring error getting Node availability zones.")
		}
		if len(nzs) != 0 {
			for _, zone := range sourceZones {
				var z string
				// getZoneFromKnownNodeZones() func makes sure that all zones
				// added to newZones[] are unique and not repeated.
				z, err = getZoneFromKnownNodeZones(ctx, zone, nzs, newZones)
				if err == nil && isZoneValid(ctx, m, z, rs) {
					newZones[z] = struct{}{}
				}
				if err != nil {
					log.WithError(err).Print("Ignoring error getting Zone from KnownNodeZones.")
				}
			}
		}
	}
	if len(newZones) == 0 {
		for _, zone := range sourceZones {
			// WithUnknownNodeZones() func makes sure that all zones
			// added to newZones[] are unique and not repeated.
			newZone := WithUnknownNodeZones(ctx, m, region, zone, newZones)
			if newZone != "" {
				newZones[newZone] = struct{}{}
			}
		}
	}

	if len(newZones) == 0 {
		return nil, errors.Errorf("Could not get zone for region %s and sourceZones %s", region, sourceZones)
	}

	var zones []string
	for z := range newZones {
		zones = append(zones, z)
	}
	return zones, nil
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
func WithUnknownNodeZones(ctx context.Context, m Mapper, region string, sourceZone string, newZones map[string]struct{}) string {
	// We could not get the zones of the nodes, so we return an arbitrary one.
	zs, err := m.FromRegion(ctx, region)
	if err != nil || len(zs) == 0 {
		// If all else fails, we return the original AZ.
		log.WithError(err).Print("Using original AZ.", field.M{"region": region})
		return sourceZone
	}

	// We look for a zone with the same suffix.
	for _, z := range zs {
		if getZoneSuffixesMatch(z, sourceZone) {
			// check if zone z is already added to newZones
			if _, ok := newZones[z]; ok {
				continue
			}
			return z
		}
	}

	// We return an arbitrary zone in the region.
	for i := range zs {
		// check if zone zs[i] is already added to newZones
		if _, ok := newZones[zs[i]]; ok {
			continue
		}
		return zs[i]
	}

	return ""
}

func getZoneFromKnownNodeZones(ctx context.Context, sourceZone string, nzs map[string]struct{}, newZones map[string]struct{}) (string, error) {
	// If the original zone is available, we return that one.
	if _, ok := nzs[sourceZone]; ok {
		return sourceZone, nil
	}

	// If there's an available zone with the zone suffix, we use that one.
	for nz := range nzs {
		if getZoneSuffixesMatch(nz, sourceZone) {
			// check if zone nz is already added to newZones
			if _, ok := newZones[nz]; ok {
				continue
			}
			return nz, nil
		}
	}
	// If any nodes are available, return an arbitrary one.
	return consistentZone(sourceZone, nzs, newZones)
}

func consistentZone(sourceZone string, nzs map[string]struct{}, newZones map[string]struct{}) (string, error) {
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

	// check if zone s[i] is already added to newZones
	if _, ok := newZones[s[i]]; ok {
		return "", nil
	}
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
	for _, n := range ns.Items {
		// For kubernetes 1.17 onwards failureDomain annotations are being deprecated
		// and will need to use topology.kubernetes.io/zone=us-east-1c and
		// topology.kubernetes.io/region=us-east-1
		// https://kubernetes.io/docs/reference/kubernetes-api/labels-annotations-taints/#topologykubernetesioregion
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
