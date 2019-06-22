package filter

import (
	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type UnstructuredSuite struct {
}

var _ = Suite(&UnstructuredSuite{})

func (s *UnstructuredSuite) TestIncludeExclude(c *C) {
	for _, tc := range []struct {
		s       Specs
		gvr     ResourceTypeMatcher
		include Specs
		exclude Specs
	}{
		{
			s:       nil,
			gvr:     nil,
			include: Specs{},
			exclude: Specs{},
		},
		{
			s:       Specs{},
			gvr:     ResourceTypeMatcher{},
			include: Specs{},
			exclude: Specs{},
		},
		{
			s: Specs{
				schema.GroupVersionResource{Group: "mygroup"}: nil,
			},
			gvr: ResourceTypeMatcher{},
			include: Specs{
				schema.GroupVersionResource{Group: "mygroup"}: nil,
			},
			exclude: Specs{
				schema.GroupVersionResource{Group: "mygroup"}: nil,
			},
		},
		{
			s: Specs{
				schema.GroupVersionResource{Group: "mygroup"}: nil,
			},
			gvr: ResourceTypeMatcher{ResourceTypeRequirement{Group: "mygroup"}},
			include: Specs{
				schema.GroupVersionResource{Group: "mygroup"}: nil,
			},
			exclude: Specs{},
		},
		{
			s: Specs{
				schema.GroupVersionResource{Group: "mygroup"}: nil,
			},
			gvr:     ResourceTypeMatcher{ResourceTypeRequirement{Group: "yourgroup"}},
			include: Specs{},
			exclude: Specs{
				schema.GroupVersionResource{Group: "mygroup"}: nil,
			},
		},
	} {
		c.Check(tc.s.Include(tc.gvr), DeepEquals, tc.include)
		c.Check(tc.s.Exclude(tc.gvr), DeepEquals, tc.exclude)
	}
}
