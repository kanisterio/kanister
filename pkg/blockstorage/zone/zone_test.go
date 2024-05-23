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
	"fmt"
	"reflect"
	"sort"
	"testing"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ZoneSuite struct{}

var _ = Suite(&ZoneSuite{})

func (s ZoneSuite) TestNodeZoneAndRegionGCP(c *C) {
	ctx := context.Background()
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kube.FDRegionLabelName: "us-west2", kube.FDZoneLabelName: "us-west2-a"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west2", kube.TopologyZoneLabelName: "us-west2-b"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node3 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west2", kube.TopologyZoneLabelName: "us-west2-c"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	// error nodes
	node4 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node4",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west2", kube.TopologyZoneLabelName: "us-west2-c"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "False",
					Type:   "Ready",
				},
			},
		},
	}
	node5 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node5",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west2", kube.TopologyZoneLabelName: "us-west2-c"},
		},
		Spec: corev1.NodeSpec{
			Unschedulable: true,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	expectedZone := make(map[string]struct{})
	expectedZone["us-west2-a"] = struct{}{}
	expectedZone["us-west2-b"] = struct{}{}
	expectedZone["us-west2-c"] = struct{}{}
	cli := fake.NewSimpleClientset(node1, node2, node3)
	z, r, err := NodeZonesAndRegion(ctx, cli)
	c.Assert(err, IsNil)
	c.Assert(reflect.DeepEqual(z, expectedZone), Equals, true)
	c.Assert(r, Equals, "us-west2")

	cli = fake.NewSimpleClientset(node4, node5)
	_, _, err = NodeZonesAndRegion(ctx, cli)
	c.Assert(err, NotNil)
}

func (s ZoneSuite) TestNodeZoneAndRegionEBS(c *C) {
	ctx := context.Background()
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west-2", kube.TopologyZoneLabelName: "us-west-2a"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west-2", kube.TopologyZoneLabelName: "us-west-2b"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node3 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west-2", kube.TopologyZoneLabelName: "us-west-2c"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	// error nodes
	node4 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node4",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west-2", kube.TopologyZoneLabelName: "us-west-2c"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "False",
					Type:   "Ready",
				},
			},
		},
	}
	node5 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node5",
			Labels: map[string]string{kube.FDRegionLabelName: "us-west-2", kube.FDZoneLabelName: "us-west-2c"},
		},
		Spec: corev1.NodeSpec{
			Unschedulable: true,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	expectedZone := make(map[string]struct{})
	expectedZone["us-west-2a"] = struct{}{}
	expectedZone["us-west-2b"] = struct{}{}
	expectedZone["us-west-2c"] = struct{}{}
	cli := fake.NewSimpleClientset(node1, node2, node3)
	z, r, err := NodeZonesAndRegion(ctx, cli)
	c.Assert(err, IsNil)
	c.Assert(reflect.DeepEqual(z, expectedZone), Equals, true)
	c.Assert(r, Equals, "us-west-2")

	cli = fake.NewSimpleClientset(node4, node5)
	_, _, err = NodeZonesAndRegion(ctx, cli)
	c.Assert(err, NotNil)
}

func (s ZoneSuite) TestNodeZoneAndRegionAD(c *C) {
	ctx := context.Background()
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kube.TopologyRegionLabelName: "westus2", kube.TopologyZoneLabelName: "westus2-1"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kube.TopologyRegionLabelName: "westus2", kube.TopologyZoneLabelName: "westus2-2"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node3 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kube.TopologyRegionLabelName: "westus2", kube.TopologyZoneLabelName: "westus2-3"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	// non-zonal node (FaultDomain)
	node4 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node4",
			Labels: map[string]string{kube.TopologyRegionLabelName: "westus", kube.TopologyZoneLabelName: "0"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	// error nodes
	node5 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node5",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west2", kube.TopologyZoneLabelName: "us-west2-4"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "False",
					Type:   "Ready",
				},
			},
		},
	}
	node6 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node6",
			Labels: map[string]string{kube.FDRegionLabelName: "us-west2", kube.FDZoneLabelName: "us-west2-5"},
		},
		Spec: corev1.NodeSpec{
			Unschedulable: true,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}

	expectedZone := make(map[string]struct{})
	expectedZone["westus2-1"] = struct{}{}
	expectedZone["westus2-2"] = struct{}{}
	expectedZone["westus2-3"] = struct{}{}
	cli := fake.NewSimpleClientset(node1, node2, node3)
	z, r, err := NodeZonesAndRegion(ctx, cli)
	c.Assert(err, IsNil)
	c.Assert(reflect.DeepEqual(z, expectedZone), Equals, true)
	c.Assert(r, Equals, "westus2")

	// non-zonal cluster test
	cli = fake.NewSimpleClientset(node4)
	z, r, err = NodeZonesAndRegion(ctx, cli)
	c.Assert(err, IsNil)
	c.Assert(len(z) == 0, Equals, true)
	c.Assert(r, Equals, "westus")

	// error case
	cli = fake.NewSimpleClientset(node5, node6)
	_, _, err = NodeZonesAndRegion(ctx, cli)
	c.Assert(err, NotNil)
}

func (s ZoneSuite) TestSanitizeZones(c *C) {
	for _, tc := range []struct {
		availableZones map[string]struct{}
		validZoneNames []string
		out            map[string]struct{}
	}{
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"us-west1-a",
				"us-west1-b",
				"us-west1-c",
			},
			out: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
		},
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"us-west1a",
				"us-west1b",
				"us-west1c",
			},
			out: map[string]struct{}{
				"us-west1a": {},
				"us-west1b": {},
				"us-west1c": {},
			},
		},
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"us-west1a",
				"us-west1b",
				"us-west1c",
				"us-west1d",
			},
			out: map[string]struct{}{
				"us-west1a": {},
				"us-west1b": {},
				"us-west1c": {},
			},
		},
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"us-west1",
				"us-west2",
			},
			out: map[string]struct{}{
				"us-west1": {},
			},
		},
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"east",
				"west",
			},
			out: map[string]struct{}{
				"west": {},
			},
		},
	} {
		out := SanitizeAvailableZones(tc.availableZones, tc.validZoneNames)
		c.Assert(out, DeepEquals, tc.out)
	}
}

func (s ZoneSuite) TestFromSourceRegionZone(c *C) {
	ctx := context.Background()
	var t = &ebsTest{}
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west-2", kube.TopologyZoneLabelName: "us-west-2a"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west-2", kube.TopologyZoneLabelName: "us-west-2b"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node3 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west-2", kube.TopologyZoneLabelName: "us-west-2c"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}

	gceNode1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west2", kube.TopologyZoneLabelName: "us-west2-a"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}

	gceNode2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west2", kube.TopologyZoneLabelName: "us-west2-b"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}

	gceNode3 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kube.TopologyRegionLabelName: "us-west2", kube.TopologyZoneLabelName: "us-west2-c"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}

	noZonesNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "noZoneNode",
			Labels: map[string]string{kube.FDRegionLabelName: "us-east2", kube.FDZoneLabelName: "us-east2-c"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}

	cli := fake.NewSimpleClientset(node1, node2, node3)
	cligce := fake.NewSimpleClientset(gceNode1, gceNode2, gceNode3)
	cliNoZoneNode := fake.NewSimpleClientset(noZonesNode)
	cliEmpty := fake.NewSimpleClientset()

	for _, tc := range []struct {
		inRegion string
		inZones  []string
		inCli    kubernetes.Interface
		outZones []string
		outErr   error
	}{
		{ // success case
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2a"},
			inCli:    cli,
			outZones: []string{"us-west-2a"},
			outErr:   nil,
		},
		{ // success case gce multi region
			inRegion: "us-west1",
			inZones:  []string{"us-west1-a"},
			inCli:    cligce,
			outZones: []string{"us-west2-a"},
			outErr:   nil,
		},
		{ // No valid zones found
			inRegion: "noValidZones",
			inZones:  []string{"us-west-2a"},
			inCli:    nil,
			outZones: nil,
			outErr:   fmt.Errorf(".*Unable to find valid availability zones for region.*"),
		},
		{ // Kubernetes provided zones are invalid use valid sourceZones
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2a", "us-west-2b", "us-west-2d"},
			inCli:    nil,
			outZones: []string{"us-west-2a", "us-west-2b"},
			outErr:   fmt.Errorf(".*Unable to find valid availability zones for region.*"),
		},
		{ // Source zone not found but other valid zones available
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2f"},
			inCli:    cli,
			outZones: []string{"us-west-2b"},
			outErr:   nil,
		},
		{ // Source zone not found but other valid zones available
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2f"},
			inCli:    cli,
			outZones: []string{"us-west-2b"},
			outErr:   nil,
		},
		{ // Source zones found
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2a", "us-west-2b"},
			inCli:    cli,
			outZones: []string{"us-west-2a", "us-west-2b"},
			outErr:   nil,
		},
		{ // One source zone found
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2a", "us-west-2f"},
			inCli:    cli,
			outZones: []string{"us-west-2a"},
			outErr:   nil,
		},
		{ // No available zones found
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2a", "us-west-2f"},
			inCli:    cliEmpty,
			outZones: []string{"us-west-2a"},
			outErr:   nil,
		},
		{ // Region Mismatch, continue normally, consistent zone
			inRegion: "us-west2",
			inZones:  []string{"us-west2-a", "us-west2-b"},
			inCli:    cli,
			outZones: []string{"us-west-2b", "us-west-2c"},
			outErr:   nil,
		},
		{ // No zones in region
			inRegion: "empty",
			inZones:  []string{"us-west-2a", "us-west-2b"},
			inCli:    cliNoZoneNode,
			outZones: nil,
			outErr:   fmt.Errorf(".*No provider zones for region.*"),
		},
		{ // Error fetching zones for region
			inRegion: "other error",
			inZones:  []string{"us-west-2a", "us-west-2b"},
			inCli:    cliNoZoneNode,
			outZones: nil,
			outErr:   fmt.Errorf(".*No provider zones for region.*"),
		},
		{ // use Source Region zones
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2a"},
			inCli:    cliNoZoneNode,
			outZones: []string{"us-west-2a"},
			outErr:   nil,
		},
	} {
		out, err := FromSourceRegionZone(ctx, t, tc.inCli, tc.inRegion, tc.inZones...)
		sort.Strings(out)
		sort.Strings(tc.outZones)
		c.Assert(out, DeepEquals, tc.outZones)
		if err != nil {
			c.Assert(err, ErrorMatches, tc.outErr.Error())
		} else {
			c.Assert(err, IsNil)
		}
	}
}

var _ Mapper = (*ebsTest)(nil)

type ebsTest struct{}

func (et *ebsTest) FromRegion(ctx context.Context, region string) ([]string, error) {
	// Fall back to using a static map.
	switch region {
	case "us-west-2":
		return []string{"us-west-2a", "us-west-2b", "us-west-2c"}, nil
	case "us-west2":
		return []string{"us-west2-a", "us-west2-b", "us-west2-c"}, nil
	case "us-west1":
		return []string{"us-west1-a", "us-west1-b", "us-west1-c"}, nil
	case "us-east2":
		return []string{}, nil
	case "empty":
		return []string{}, nil
	case "noValidZones":
		return []string{"no", "valid", "zones"}, nil
	default:
		return nil, fmt.Errorf("Some error")
	}
}

func (s ZoneSuite) TestGetReadySchedulableNodes(c *C) {
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kube.TopologyRegionLabelName: "westus2", kube.TopologyZoneLabelName: "westus2-1"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kube.TopologyRegionLabelName: "westus2", kube.TopologyZoneLabelName: "westus2-2"},
		},
		Spec: corev1.NodeSpec{
			Unschedulable: true,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node3 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kube.FDRegionLabelName: "westus2", kube.FDZoneLabelName: "westus2-3"},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Status: "False",
					Type:   "Ready",
				},
			},
		},
	}
	cli := fake.NewSimpleClientset(node1, node2, node3)
	nl, err := GetReadySchedulableNodes(cli)
	c.Assert(err, IsNil)
	c.Assert(len(nl), Equals, 1)

	node1.Spec = corev1.NodeSpec{
		Unschedulable: true,
	}
	cli = fake.NewSimpleClientset(node1, node2, node3)
	nl, err = GetReadySchedulableNodes(cli)
	c.Assert(err, NotNil)
	c.Assert(nl, IsNil)
}

func (s ZoneSuite) TestConsistentZones(c *C) {
	// no available zones
	z := consistentZone("source", map[string]struct{}{})
	c.Assert(z, Equals, "")

	az1 := map[string]struct{}{
		"a": {},
		"b": {},
		"c": {},
	}

	az2 := map[string]struct{}{
		"c": {},
		"a": {},
		"b": {},
	}

	z1 := consistentZone("x", az1)
	z2 := consistentZone("x", az2)

	c.Assert(z1, Equals, z2)

	// different lists result in different zones
	az2["d"] = struct{}{}
	z1 = consistentZone("x", az1)
	z2 = consistentZone("x", az2)

	c.Assert(z1, Not(Equals), z2)
}
