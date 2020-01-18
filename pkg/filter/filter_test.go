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

package filter

import (
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type FilterSuite struct{}

var _ = Suite(&FilterSuite{})

func (s *FilterSuite) TestGVRRequirement(c *C) {
	for _, tc := range []struct {
		gvrr     ResourceTypeRequirement
		gvr      schema.GroupVersionResource
		expected bool
	}{
		// Basic case
		{
			gvrr: ResourceTypeRequirement{
				Group:    "",
				Version:  "",
				Resource: "",
			},
			gvr: schema.GroupVersionResource{
				Group:    "",
				Version:  "",
				Resource: "",
			},
			expected: true,
		},

		// Case w/ Version Requirements
		{
			gvrr: ResourceTypeRequirement{
				Group:    "",
				Version:  "v1",
				Resource: "",
			},
			gvr: schema.GroupVersionResource{
				Group:    "",
				Version:  "",
				Resource: "",
			},
			expected: false,
		},
		{
			gvrr: ResourceTypeRequirement{
				Group:    "",
				Version:  "v1",
				Resource: "",
			},
			gvr: schema.GroupVersionResource{
				Group:    "",
				Version:  "v2",
				Resource: "",
			},
			expected: false,
		},
		{
			gvrr: ResourceTypeRequirement{
				Group:    "",
				Version:  "v1",
				Resource: "",
			},
			gvr: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "",
			},
			expected: true,
		},
		{
			gvrr: ResourceTypeRequirement{
				Group:    "",
				Version:  "v1",
				Resource: "",
			},
			gvr: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			expected: true,
		},

		// Wrong group
		{
			gvrr: ResourceTypeRequirement{
				Group:    "apps",
				Version:  "v1",
				Resource: "services",
			},
			gvr: schema.GroupVersionResource{
				Group:    "myapps",
				Version:  "v1 ",
				Resource: "services",
			},
			expected: false,
		},

		// Wrong object
		{
			gvrr: ResourceTypeRequirement{
				Group:    "",
				Version:  "v1",
				Resource: "services",
			},
			gvr: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1 ",
				Resource: "pods",
			},
			expected: false,
		},
	} {
		c.Check(tc.gvrr.Matches(tc.gvr), Equals, tc.expected, Commentf("GVRR: %v, GVR: %v", tc.gvrr, tc.gvr))
	}
}

func (s *FilterSuite) TestGroupVersionResourceEmpty(c *C) {
	var g ResourceTypeMatcher
	c.Assert(g.Empty(), Equals, true)
	g = ResourceTypeMatcher{}
	c.Assert(g.Empty(), Equals, true)
	g = ResourceTypeMatcher{ResourceTypeRequirement{}}
	c.Assert(g.Empty(), Equals, false)
}

func (s *FilterSuite) TestGroupVersionResourceAnyAll(c *C) {
	for _, tc := range []struct {
		g   ResourceTypeMatcher
		gvr schema.GroupVersionResource
		any bool
		all bool
	}{
		// Note: If we feel this behavior is unexpected, we can modify the implementation.
		{
			g:   nil,
			gvr: schema.GroupVersionResource{},
			any: false,
			all: true,
		},
		{
			g:   ResourceTypeMatcher{},
			gvr: schema.GroupVersionResource{},
			any: false,
			all: true,
		},
		{
			g: ResourceTypeMatcher{
				ResourceTypeRequirement{},
			},
			gvr: schema.GroupVersionResource{},
			any: true,
			all: true,
		},
		{
			g: ResourceTypeMatcher{
				ResourceTypeRequirement{Group: "mygroup"},
			},
			gvr: schema.GroupVersionResource{Group: "mygroup"},
			any: true,
			all: true,
		},
		{
			g: ResourceTypeMatcher{
				ResourceTypeRequirement{Group: "mygroup"},
			},
			gvr: schema.GroupVersionResource{Group: "yourgroup"},
			any: false,
			all: false,
		},
		{
			g: ResourceTypeMatcher{
				ResourceTypeRequirement{Group: "mygroup"},
				ResourceTypeRequirement{Group: "yourgroup"},
			},
			gvr: schema.GroupVersionResource{Group: "yourgroup"},
			any: true,
			all: false,
		},
		{
			g: ResourceTypeMatcher{
				ResourceTypeRequirement{Group: "mygroup"},
				ResourceTypeRequirement{Group: "yourgroup"},
			},
			gvr: schema.GroupVersionResource{Group: "ourgroup"},
			any: false,
			all: false,
		},
	} {
		c.Check(tc.g.Any(tc.gvr), Equals, tc.any)
		c.Check(tc.g.All(tc.gvr), Equals, tc.all)
	}
}

func (s *FilterSuite) TestGroupVersionResourceIncludeExclude(c *C) {
	for _, tc := range []struct {
		m       ResourceTypeMatcher
		gvrs    GroupVersionResourceList
		include GroupVersionResourceList
		exclude GroupVersionResourceList
	}{
		{
			m: nil,
			gvrs: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
			include: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
			exclude: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
		},
		{
			m: ResourceTypeMatcher{},
			gvrs: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
			include: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
			exclude: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
		},
		{
			m: ResourceTypeMatcher{ResourceTypeRequirement{}},
			gvrs: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
			include: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
			exclude: []schema.GroupVersionResource{},
		},
		{
			m: ResourceTypeMatcher{ResourceTypeRequirement{}},
			gvrs: []schema.GroupVersionResource{
				schema.GroupVersionResource{
					Group: "mygroup",
				},
			},
			include: []schema.GroupVersionResource{
				schema.GroupVersionResource{
					Group: "mygroup",
				},
			},
			exclude: []schema.GroupVersionResource{},
		},
		{
			m: ResourceTypeMatcher{
				ResourceTypeRequirement{
					Group: "mygroup",
				},
			},
			gvrs: []schema.GroupVersionResource{
				schema.GroupVersionResource{
					Group: "mygroup",
				},
			},
			include: []schema.GroupVersionResource{
				schema.GroupVersionResource{
					Group: "mygroup",
				},
			},
			exclude: []schema.GroupVersionResource{},
		},
		{
			m: ResourceTypeMatcher{
				ResourceTypeRequirement{
					Group: "mygroup",
				},
				ResourceTypeRequirement{
					Version: "myversion",
				},
			},
			gvrs: []schema.GroupVersionResource{
				schema.GroupVersionResource{
					Group: "mygroup",
				},
				schema.GroupVersionResource{
					Version: "myversion",
				},
				schema.GroupVersionResource{
					Group:   "mygroup",
					Version: "myversion",
				},
				schema.GroupVersionResource{
					Group:   "mygroup",
					Version: "yourversion",
				},
				schema.GroupVersionResource{
					Group:   "yourgroup",
					Version: "myversion",
				},
				schema.GroupVersionResource{
					Group:   "yourgroup",
					Version: "yourversion",
				},
			},
			include: []schema.GroupVersionResource{
				schema.GroupVersionResource{
					Group: "mygroup",
				},
				schema.GroupVersionResource{
					Version: "myversion",
				},
				schema.GroupVersionResource{
					Group:   "mygroup",
					Version: "myversion",
				},
				schema.GroupVersionResource{
					Group:   "mygroup",
					Version: "yourversion",
				},
				schema.GroupVersionResource{
					Group:   "yourgroup",
					Version: "myversion",
				},
			},
			exclude: []schema.GroupVersionResource{
				schema.GroupVersionResource{
					Group:   "yourgroup",
					Version: "yourversion",
				},
			},
		},
	} {
		c.Check(tc.gvrs.Include(tc.m), DeepEquals, tc.include)
		c.Check(tc.gvrs.Exclude(tc.m), DeepEquals, tc.exclude)
	}
}

func (s *FilterSuite) TestJoin(c *C) {
	for _, tc := range []struct {
		m   []ResourceTypeMatcher
		out ResourceTypeMatcher
	}{
		{
			m:   []ResourceTypeMatcher{ResourceTypeMatcher{}, ResourceTypeMatcher{}},
			out: ResourceTypeMatcher{},
		},
		{
			m:   []ResourceTypeMatcher{ResourceTypeMatcher{}, ResourceTypeMatcher{}},
			out: ResourceTypeMatcher{},
		},
	} {
		c.Check(JoinResourceTypeMatchers(tc.m...), DeepEquals, tc.out)
	}
}

func (s *FilterSuite) TestResourceIncludeExclude(c *C) {
	ssTypeRequirement := ResourceTypeRequirement{Group: "apps", Resource: "statefulsets"}
	pvcTypeRequirement := ResourceTypeRequirement{Version: "v1", Resource: "persistentvolumeclaims"}
	ss1 := Resource{Name: "ss1", GVR: schema.GroupVersionResource{Group: "apps", Resource: "statefulsets"}}
	ss2 := Resource{Name: "specificname", GVR: schema.GroupVersionResource{Group: "apps", Resource: "statefulsets"}}
	pvc1 := Resource{Name: "pvc1", GVR: schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}}
	pvc2 := Resource{Name: "specificname", GVR: schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}}

	for _, tc := range []struct {
		m         ResourceMatcher
		resources ResourceList
		include   ResourceList
		exclude   ResourceList
	}{
		{
			// No matcher, empty resource list
			m:         nil,
			resources: []Resource{Resource{}},
			include:   []Resource{Resource{}},
			exclude:   []Resource{Resource{}},
		},
		{
			// No matcher, include/exclude is a no-op
			m:         nil,
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{ss1, ss2, pvc1, pvc2},
			exclude:   []Resource{ss1, ss2, pvc1, pvc2},
		},
		{
			// Empty matcher, include/exclude is a no-op
			m:         ResourceMatcher{},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{ss1, ss2, pvc1, pvc2},
			exclude:   []Resource{ss1, ss2, pvc1, pvc2},
		},
		{
			// Match all types
			m: ResourceMatcher{
				ResourceRequirement{ResourceTypeRequirement: ssTypeRequirement},
				ResourceRequirement{ResourceTypeRequirement: pvcTypeRequirement},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{ss1, ss2, pvc1, pvc2},
			exclude:   []Resource{},
		},
		{
			// Match one type
			m: ResourceMatcher{
				ResourceRequirement{ResourceTypeRequirement: pvcTypeRequirement},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{pvc1, pvc2},
			exclude:   []Resource{ss1, ss2},
		},
		{
			// Match a specific resource
			m: ResourceMatcher{
				ResourceRequirement{LocalObjectReference: v1.LocalObjectReference{Name: "pvc1"}, ResourceTypeRequirement: pvcTypeRequirement},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{pvc1},
			exclude:   []Resource{ss1, ss2, pvc2},
		},
		{
			// Match a specific resource name only (no GVR), matches only one object
			m: ResourceMatcher{
				ResourceRequirement{LocalObjectReference: v1.LocalObjectReference{Name: "pvc1"}},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{pvc1},
			exclude:   []Resource{ss1, ss2, pvc2},
		},
		{
			// Match a specific resource name only (no GVR), matches mulitple resources
			m: ResourceMatcher{
				ResourceRequirement{LocalObjectReference: v1.LocalObjectReference{Name: "specificname"}},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{ss2, pvc2},
			exclude:   []Resource{ss1, pvc1},
		},
	} {
		c.Check(tc.resources.Include(tc.m), DeepEquals, tc.include)
		c.Check(tc.resources.Exclude(tc.m), DeepEquals, tc.exclude)
	}
}
