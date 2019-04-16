package filter

import (
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type FilterSuite struct{}

var _ = Suite(&FilterSuite{})

func (s *FilterSuite) TestGVRRequirement(c *C) {
	for _, tc := range []struct {
		gvrr     ResourceRequirement
		gvr      schema.GroupVersionResource
		expected bool
	}{
		// Basic case
		{
			gvrr: ResourceRequirement{
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
			gvrr: ResourceRequirement{
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
			gvrr: ResourceRequirement{
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
			gvrr: ResourceRequirement{
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
			gvrr: ResourceRequirement{
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
			gvrr: ResourceRequirement{
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
			gvrr: ResourceRequirement{
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
	var g ResourceMatcher
	c.Assert(g.Empty(), Equals, true)
	g = ResourceMatcher{}
	c.Assert(g.Empty(), Equals, true)
	g = ResourceMatcher{ResourceRequirement{}}
	c.Assert(g.Empty(), Equals, false)
}

func (s *FilterSuite) TestGroupVersionResourceAnyAll(c *C) {
	for _, tc := range []struct {
		g   ResourceMatcher
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
			g:   ResourceMatcher{},
			gvr: schema.GroupVersionResource{},
			any: false,
			all: true,
		},
		{
			g: ResourceMatcher{
				ResourceRequirement{},
			},
			gvr: schema.GroupVersionResource{},
			any: true,
			all: true,
		},
		{
			g: ResourceMatcher{
				ResourceRequirement{Group: "mygroup"},
			},
			gvr: schema.GroupVersionResource{Group: "mygroup"},
			any: true,
			all: true,
		},
		{
			g: ResourceMatcher{
				ResourceRequirement{Group: "mygroup"},
			},
			gvr: schema.GroupVersionResource{Group: "yourgroup"},
			any: false,
			all: false,
		},
		{
			g: ResourceMatcher{
				ResourceRequirement{Group: "mygroup"},
				ResourceRequirement{Group: "yourgroup"},
			},
			gvr: schema.GroupVersionResource{Group: "yourgroup"},
			any: true,
			all: false,
		},
		{
			g: ResourceMatcher{
				ResourceRequirement{Group: "mygroup"},
				ResourceRequirement{Group: "yourgroup"},
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
		g       ResourceMatcher
		gvrs    []schema.GroupVersionResource
		include []schema.GroupVersionResource
		exclude []schema.GroupVersionResource
	}{
		{
			g: nil,
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
			g: ResourceMatcher{},
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
			g: ResourceMatcher{ResourceRequirement{}},
			gvrs: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
			include: []schema.GroupVersionResource{
				schema.GroupVersionResource{},
			},
			exclude: []schema.GroupVersionResource{},
		},
		{
			g: ResourceMatcher{ResourceRequirement{}},
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
			g: ResourceMatcher{
				ResourceRequirement{
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
			g: ResourceMatcher{
				ResourceRequirement{
					Group: "mygroup",
				},
				ResourceRequirement{
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
		c.Check(tc.g.Include(tc.gvrs), DeepEquals, tc.include)
		c.Check(tc.g.Exclude(tc.gvrs), DeepEquals, tc.exclude)
	}
}
