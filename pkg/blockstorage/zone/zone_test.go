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
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ZoneSuite struct{}

var _ = Suite(&ZoneSuite{})

func (s ZoneSuite) TestConsistentZone(c *C) {
	// We don't care what the answer is as long as it's consistent.
	for _, tc := range []struct {
		sourceZone string
		nzs        map[string]struct{}
		out        string
	}{
		{
			sourceZone: "",
			nzs: map[string]struct{}{
				"zone1": {},
			},
			out: "zone1",
		},
		{
			sourceZone: "",
			nzs: map[string]struct{}{
				"zone1": {},
				"zone2": {},
			},
			out: "zone2",
		},
		{
			sourceZone: "from1",
			nzs: map[string]struct{}{
				"zone1": {},
				"zone2": {},
			},
			out: "zone1",
		},
	} {
		out, err := consistentZone(tc.sourceZone, tc.nzs, make(map[string]struct{}))
		c.Assert(err, IsNil)
		c.Assert(out, Equals, tc.out)
	}
}

func (s ZoneSuite) TestNodeZoneAndRegionGCP(c *C) {
	ctx := context.Background()
	node1 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-a"},
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
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-b"},
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
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-c"},
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
	// error nodes
	node4 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node4",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-c"},
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				v1.NodeCondition{
					Status: "False",
					Type:   "Ready",
				},
			},
		},
	}
	node5 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node5",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-c"},
		},
		Spec: v1.NodeSpec{
			Unschedulable: true,
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
	// error nodes
	node4 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node4",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west-2", kubevolume.PVZoneLabelName: "us-west-2c"},
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				v1.NodeCondition{
					Status: "False",
					Type:   "Ready",
				},
			},
		},
	}
	node5 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node5",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west-2", kubevolume.PVZoneLabelName: "us-west-2c"},
		},
		Spec: v1.NodeSpec{
			Unschedulable: true,
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
	node1 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-1"},
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
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-2"},
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
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-3"},
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
	// error nodes
	node4 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node4",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-4"},
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				v1.NodeCondition{
					Status: "False",
					Type:   "Ready",
				},
			},
		},
	}
	node5 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node5",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "us-west2", kubevolume.PVZoneLabelName: "us-west2-5"},
		},
		Spec: v1.NodeSpec{
			Unschedulable: true,
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

	expectedZone := make(map[string]struct{})
	expectedZone["westus2-1"] = struct{}{}
	expectedZone["westus2-2"] = struct{}{}
	expectedZone["westus2-3"] = struct{}{}
	cli := fake.NewSimpleClientset(node1, node2, node3)
	z, r, err := NodeZonesAndRegion(ctx, cli)
	c.Assert(err, IsNil)
	c.Assert(reflect.DeepEqual(z, expectedZone), Equals, true)
	c.Assert(r, Equals, "westus2")

	cli = fake.NewSimpleClientset(node4, node5)
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
		out := sanitizeAvailableZones(tc.availableZones, tc.validZoneNames)
		c.Assert(out, DeepEquals, tc.out)
	}
}

func (s ZoneSuite) TestGetReadySchedulableNodes(c *C) {
	node1 := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node1",
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-1"},
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
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-2"},
		},
		Spec: v1.NodeSpec{
			Unschedulable: true,
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
			Labels: map[string]string{kubevolume.PVRegionLabelName: "westus2", kubevolume.PVZoneLabelName: "westus2-3"},
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				v1.NodeCondition{
					Status: "False",
					Type:   "Ready",
				},
			},
		},
	}

	cli := fake.NewSimpleClientset(node1, node2, node3)
	nl, err := GetReadySchedulableNodes(cli)
	c.Assert(err, IsNil)
	c.Assert(len(nl.Items), Equals, 1)

	node1.Spec = v1.NodeSpec{
		Unschedulable: true,
	}
	cli = fake.NewSimpleClientset(node1, node2, node3)
	nl, err = GetReadySchedulableNodes(cli)
	c.Assert(err, NotNil)
	c.Assert(nl, IsNil)
}
