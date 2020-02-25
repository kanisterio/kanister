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

	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ZoneSuite struct{}

var _ = Suite(&ZoneSuite{})

func (s ZoneSuite) TestNodeZoneAndRegionGCP(c *C) {
	ctx := context.Background()
	node1 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-a"},
		},
	}
	node2 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-b"},
		},
	}
	node3 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-c"},
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
}

func (s ZoneSuite) TestNodeZoneAndRegionEBS(c *C) {
	ctx := context.Background()
	node1 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west-2", kubevolume.PVZoneLabelName: "us-west-2a"},
		},
	}
	node2 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west-2", kubevolume.PVZoneLabelName: "us-west-2b"},
		},
	}
	node3 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west-2", kubevolume.PVZoneLabelName: "us-west-2c"},
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
}

func (s ZoneSuite) TestNodeZoneAndRegionAD(c *C) {
	ctx := context.Background()
	node1 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-1"},
		},
	}
	node2 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-2"},
		},
	}
	node3 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-3"},
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
	node1 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west-2", kubevolume.PVZoneLabelName: "us-west-2a"},
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				v1.NodeCondition{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node2 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node2",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west-2", kubevolume.PVZoneLabelName: "us-west-2b"},
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				v1.NodeCondition{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}
	node3 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node3",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west-2", kubevolume.PVZoneLabelName: "us-west-2c"},
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				v1.NodeCondition{
					Status: "True",
					Type:   "Ready",
				},
			},
		},
	}

	cli := fake.NewSimpleClientset(node1, node2, node3)
	cliEmpty := fake.NewSimpleClientset()

	for _, tc := range []struct {
		inRegion string
		inZones  []string
		inCli    *fake.Clientset
		outZones []string
		outErr   error
	}{
		{ //success case
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2a"},
			inCli:    cli,
			outZones: []string{"us-west-2a"},
			outErr:   nil,
		},
		{ // No valid zones found
			inRegion: "noValidZones",
			inZones:  []string{"us-west-2a"},
			inCli:    nil,
			outZones: nil,
			outErr:   fmt.Errorf(".*Unable to find valid availabilty zones for region.*"),
		},
		{ // Kubernetes provided zones are invalid use valid sourceZones
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2a", "us-west-2b", "us-west-2d"},
			inCli:    nil,
			outZones: []string{"us-west-2a", "us-west-2b"},
			outErr:   fmt.Errorf(".*Unable to find valid availabilty zones for region.*"),
		},
		{ // Source zone not found but other valid zones available
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2f"},
			inCli:    cli,
			outZones: []string{"us-west-2a", "us-west-2b", "us-west-2c"},
			outErr:   nil,
		},
		{ // Source zone not found but other valid zones available
			inRegion: "us-west-2",
			inZones:  []string{"us-west-2f"},
			inCli:    cli,
			outZones: []string{"us-west-2a", "us-west-2b", "us-west-2c"},
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
		{ // Region Mismatch, continue normally
			inRegion: "us-west2",
			inZones:  []string{"us-west-2a", "us-west-2b"},
			inCli:    cli,
			outZones: []string{"us-west-2a", "us-west-2b"},
			outErr:   nil,
		},
		{ // No zones in region
			inRegion: "empty",
			inZones:  []string{"us-west-2a", "us-west-2b"},
			inCli:    cli,
			outZones: nil,
			outErr:   fmt.Errorf(".*No provider zones for region.*"),
		},
		{ // Error fetching zones for region
			inRegion: "other error",
			inZones:  []string{"us-west-2a", "us-west-2b"},
			inCli:    cli,
			outZones: nil,
			outErr:   fmt.Errorf(".*No provider zones for region.*"),
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
		return []string{"us-west-2a", "us-west-2b", "us-west-2c"}, nil
	case "empty":
		return []string{}, nil
	case "noValidZones":
		return []string{"no", "valid", "zones"}, nil
	default:
		return nil, fmt.Errorf("Some error")
	}
}
