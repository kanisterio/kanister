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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				{},
			},
			include: []schema.GroupVersionResource{
				{},
			},
			exclude: []schema.GroupVersionResource{
				{},
			},
		},
		{
			m: ResourceTypeMatcher{},
			gvrs: []schema.GroupVersionResource{
				{},
			},
			include: []schema.GroupVersionResource{
				{},
			},
			exclude: []schema.GroupVersionResource{
				{},
			},
		},
		{
			m: ResourceTypeMatcher{ResourceTypeRequirement{}},
			gvrs: []schema.GroupVersionResource{
				{},
			},
			include: []schema.GroupVersionResource{
				{},
			},
			exclude: []schema.GroupVersionResource{},
		},
		{
			m: ResourceTypeMatcher{ResourceTypeRequirement{}},
			gvrs: []schema.GroupVersionResource{
				{
					Group: "mygroup",
				},
			},
			include: []schema.GroupVersionResource{
				{
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
				{
					Group: "mygroup",
				},
			},
			include: []schema.GroupVersionResource{
				{
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
				{
					Group: "mygroup",
				},
				{
					Version: "myversion",
				},
				{
					Group:   "mygroup",
					Version: "myversion",
				},
				{
					Group:   "mygroup",
					Version: "yourversion",
				},
				{
					Group:   "yourgroup",
					Version: "myversion",
				},
				{
					Group:   "yourgroup",
					Version: "yourversion",
				},
			},
			include: []schema.GroupVersionResource{
				{
					Group: "mygroup",
				},
				{
					Version: "myversion",
				},
				{
					Group:   "mygroup",
					Version: "myversion",
				},
				{
					Group:   "mygroup",
					Version: "yourversion",
				},
				{
					Group:   "yourgroup",
					Version: "myversion",
				},
			},
			exclude: []schema.GroupVersionResource{
				{
					Group:   "yourgroup",
					Version: "yourversion",
				},
			},
		},
		{
			m: ResourceTypeMatcher{
				ResourceTypeRequirement{
					Group:    "core",
					Resource: "myresource",
				},
			},
			gvrs: []schema.GroupVersionResource{
				{
					Group:    "",
					Resource: "myresource",
				},
				{
					Group:    "core",
					Resource: "myresource",
				},
				{
					Group:    "mygroup",
					Resource: "myresource",
				},
				{
					Group:    "",
					Resource: "yourresource",
				},
				{
					Group:    "core",
					Resource: "yourresource",
				},
				{
					Group:    "mygroup",
					Resource: "yourresource",
				},
			},
			include: []schema.GroupVersionResource{
				{
					Group:    "",
					Resource: "myresource",
				},
			},
			exclude: []schema.GroupVersionResource{
				{
					Group:    "core",
					Resource: "myresource",
				},
				{
					Group:    "mygroup",
					Resource: "myresource",
				},
				{
					Group:    "",
					Resource: "yourresource",
				},
				{
					Group:    "core",
					Resource: "yourresource",
				},
				{
					Group:    "mygroup",
					Resource: "yourresource",
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
			m:   []ResourceTypeMatcher{{}, {}},
			out: ResourceTypeMatcher{},
		},
		{
			m:   []ResourceTypeMatcher{{}, {}},
			out: ResourceTypeMatcher{},
		},
	} {
		c.Check(JoinResourceTypeMatchers(tc.m...), DeepEquals, tc.out)
	}
}

func (s *FilterSuite) TestResourceIncludeExclude(c *C) {
	ssTypeRequirement := ResourceTypeRequirement{Group: "apps", Resource: "statefulsets"}
	pvcTypeRequirement := ResourceTypeRequirement{Version: "v1", Resource: "persistentvolumeclaims"}
	ss1 := Resource{Name: "ss1", GVR: schema.GroupVersionResource{Group: "apps", Resource: "statefulsets"},
		ResourceLabels: map[string]string{"testkey1": "testval1"}}
	ss2 := Resource{Name: "specificname", GVR: schema.GroupVersionResource{Group: "apps", Resource: "statefulsets"},
		ResourceLabels: map[string]string{"testkey2": "testval2"}}
	pvc1 := Resource{Name: "pvc1", GVR: schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"},
		ResourceLabels: map[string]string{"testkey1": "testval1"}}
	pvc2 := Resource{Name: "specificname", GVR: schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"},
		ResourceLabels: map[string]string{"testkey2": "testval2"}}

	ss1nl := Resource{Name: "ss1", GVR: schema.GroupVersionResource{Group: "apps", Resource: "statefulsets"}}
	ss2nl := Resource{Name: "specificname", GVR: schema.GroupVersionResource{Group: "apps", Resource: "statefulsets"}}
	pvc1nl := Resource{Name: "pvc1", GVR: schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}}
	pvc2nl := Resource{Name: "specificname", GVR: schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}}

	ss2diff := Resource{Name: "specificname", GVR: schema.GroupVersionResource{Group: "diffapps", Resource: "diffname"}}

	for _, tc := range []struct {
		m         ResourceMatcher
		resources ResourceList
		include   ResourceList
		exclude   ResourceList
	}{
		{
			// No matcher, empty resource list
			m:         nil,
			resources: []Resource{{}},
			include:   []Resource{{}},
			exclude:   []Resource{{}},
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
				ResourceRequirement{LocalObjectReference: corev1.LocalObjectReference{Name: "pvc1"}, ResourceTypeRequirement: pvcTypeRequirement},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{pvc1},
			exclude:   []Resource{ss1, ss2, pvc2},
		},
		{
			// Match a specific resource name only (no GVR), matches only one object
			m: ResourceMatcher{
				ResourceRequirement{LocalObjectReference: corev1.LocalObjectReference{Name: "pvc1"}},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{pvc1},
			exclude:   []Resource{ss1, ss2, pvc2},
		},
		{
			// Match a specific resource name only (no GVR), matches multiple resources
			m: ResourceMatcher{
				ResourceRequirement{LocalObjectReference: corev1.LocalObjectReference{Name: "specificname"}},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{ss2, pvc2},
			exclude:   []Resource{ss1, pvc1},
		},
		{
			// Match a specific resource name with different GVR, matches only one object
			m: ResourceMatcher{
				ResourceRequirement{LocalObjectReference: corev1.LocalObjectReference{Name: "specificname"},
					ResourceTypeRequirement: ssTypeRequirement,
				},
			},
			resources: []Resource{ss1, ss2diff, pvc1, pvc2},
			include:   []Resource{},
			exclude:   []Resource{ss1, ss2diff, pvc1, pvc2},
		},
		{
			// Match by GVR and labels
			m: ResourceMatcher{
				ResourceRequirement{LocalObjectReference: corev1.LocalObjectReference{Name: "specificname"},
					LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{
						"testkey2": "testval2", // Include only the labels with 2
					}}},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{ss2, pvc2},
			exclude:   []Resource{ss1, pvc1},
		},
		{
			// Match by labels only
			m: ResourceMatcher{
				ResourceRequirement{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{
					"testkey2": "testval2", // Include only resources with these labels
				}}},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{ss2, pvc2},
			exclude:   []Resource{ss1, pvc1},
		},
		{
			// Match by labels only but with resources without labels
			m: ResourceMatcher{
				ResourceRequirement{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{
					"testkey2": "testval2", // Include only resources with these labels
				}}},
			},
			resources: []Resource{ss1nl, ss2nl, pvc1nl, pvc2nl},
			include:   []Resource{}, // There are no resources with the required labels
			exclude:   []Resource{ss1nl, ss2nl, pvc1nl, pvc2nl},
		},
		{
			// Match by labels using well formed match expression
			m: ResourceMatcher{
				ResourceRequirement{LabelSelector: metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "testkey2",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"testval2", "testval3"},
					},
				}}},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{ss2, pvc2},
			exclude:   []Resource{ss1, pvc1},
		},
		{
			// Match where resources have no labels but using match expression with not in
			m: ResourceMatcher{
				ResourceRequirement{LabelSelector: metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "testkey2",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"testval2", "testval3"},
					},
				}}},
			},
			resources: []Resource{ss1nl, ss2nl, pvc1nl, pvc2nl},
			include:   []Resource{ss1nl, ss2nl, pvc1nl, pvc2nl},
			exclude:   []Resource{},
		},
		{
			// Match by labels using mal-formed match expression
			// Will return error of: '"notsupported" is not a valid pod selector operator'
			// on call to metav1.LabelSelectorAsSelector
			// so filter will treat it as if labels to match were supplied
			// but no MatchExpressions or MatchLabels were provided and
			// will then return false in the Matches method.
			m: ResourceMatcher{
				ResourceRequirement{LabelSelector: metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "testkey2",
						Operator: "notsupported",
						Values:   []string{"testval2", "testval3"},
					},
				}}},
			},
			resources: []Resource{ss1, ss2, pvc1, pvc2},
			include:   []Resource{},
			exclude:   []Resource{ss1, ss2, pvc1, pvc2},
		},
	} {
		c.Check(tc.resources.Include(tc.m), DeepEquals, tc.include)
		c.Check(tc.resources.Exclude(tc.m), DeepEquals, tc.exclude)
	}
}

func (s *FilterSuite) TestResourceRequirementDeepCopyInto(c *C) {
	rr := ResourceRequirement{LocalObjectReference: corev1.LocalObjectReference{Name: "specificname"},
		ResourceTypeRequirement: ResourceTypeRequirement{Group: "apps", Resource: "statefulsets"},
		LabelSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"testkey2": "testval2",
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "testkey2",
					Operator: "notsupported",
					Values:   []string{"testval2", "testval3"},
				},
			}},
	}
	var rrCopy ResourceRequirement
	rr.DeepCopyInto(&rrCopy)
	c.Check(rr, DeepEquals, rrCopy)
	// Change original and check again to be sure is not equals
	rr.LocalObjectReference.Name = "newval"
	c.Check(rr, Not(DeepEquals), rrCopy)
	rr.LocalObjectReference.Name = "specificname"
	c.Check(rr, DeepEquals, rrCopy)
	rr.ResourceTypeRequirement.Group = "newgroup"
	c.Check(rr, Not(DeepEquals), rrCopy)
}
