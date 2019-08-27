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
