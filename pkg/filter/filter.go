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
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceTypeRequirement struct {
	Group    string `json:"group,omitempty"`
	Version  string `json:"version,omitempty"`
	Resource string `json:"resource,omitempty"`
}

func (r ResourceTypeRequirement) Matches(gvr schema.GroupVersionResource) bool {
	return matches(r.Group, gvr.Group) && matches(r.Version, gvr.Version) && matches(r.Resource, gvr.Resource)
}

func matches(sel, val string) bool {
	return sel == "" || sel == val
}

type ResourceTypeMatcher []ResourceTypeRequirement

func (g ResourceTypeMatcher) Empty() bool {
	return len(g) == 0
}

func (g ResourceTypeMatcher) Any(gvr schema.GroupVersionResource) bool {
	for _, req := range g {
		if req.Matches(gvr) {
			return true
		}
	}
	return false
}

func (g ResourceTypeMatcher) All(gvr schema.GroupVersionResource) bool {
	for _, req := range g {
		if !req.Matches(gvr) {
			return false
		}
	}
	return true
}

// ResourceMatcher constructs a resource matcher
// based on a `ResourceTypeMatcher`
func (g ResourceTypeMatcher) ResourceMatcher() ResourceMatcher {
	rm := make(ResourceMatcher, 0, len(g))
	for _, rtr := range g {
		rm = append(rm, ResourceRequirement{ResourceTypeRequirement: rtr})
	}
	return rm
}

type GroupVersionResourceList []schema.GroupVersionResource

func (g GroupVersionResourceList) Include(ms ...ResourceTypeMatcher) GroupVersionResourceList {
	return g.apply(ms, false)
}

func (g GroupVersionResourceList) Exclude(ms ...ResourceTypeMatcher) GroupVersionResourceList {
	return g.apply(ms, true)
}

func (g GroupVersionResourceList) apply(ms []ResourceTypeMatcher, exclude bool) GroupVersionResourceList {
	m := JoinResourceTypeMatchers(ms...)
	if m.Empty() {
		return g
	}
	filtered := make([]schema.GroupVersionResource, 0, len(g))
	for _, gvr := range g {
		if exclude != m.Any(gvr) {
			filtered = append(filtered, gvr)
		}
	}
	return filtered
}

// JoinResourceTypeMatchers joins multiple ResourceTypeMatchers into one
func JoinResourceTypeMatchers(ms ...ResourceTypeMatcher) ResourceTypeMatcher {
	n := 0
	for _, m := range ms {
		n += len(m)
	}
	gvr := make(ResourceTypeMatcher, n)
	i := 0
	for _, m := range ms {
		copy(gvr[i:], []ResourceTypeRequirement(m))
		i += len(m)
	}
	return gvr
}

// ResourceRequirement allows specifying a resource requirement by type and/or name
type ResourceRequirement struct {
	v1.LocalObjectReference `json:",inline,omitempty"`
	ResourceTypeRequirement `json:",inline,omitempty"`
}

// Matches returns true if the specified resource name/GVR matches the requirement
func (r ResourceRequirement) Matches(name string, gvr schema.GroupVersionResource) bool {
	// If the requirement does not specify a resource name - only check the
	// ResourceTypeRequirement i.e. GVR match
	if r.LocalObjectReference.Name == "" {
		return r.ResourceTypeRequirement.Matches(gvr)
	}
	return matches(r.Name, name) && r.ResourceTypeRequirement.Matches(gvr)
}

type ResourceMatcher []ResourceRequirement

func (g ResourceMatcher) Empty() bool {
	return len(g) == 0
}

// TypeMatcher constructs a resource type matcher
// based on a `ResourceMatcher`
//
// The `usageExclusion` flag should be set to true
// if the type matcher will be used as an exclude filter
func (rm ResourceMatcher) TypeMatcher(usageInclusion bool) ResourceTypeMatcher {
	rtm := make(ResourceTypeMatcher, 0, len(rm))
	for _, rr := range rm {
		// Include the type requirement from the ResourceRequirement, if
		//  - There is no "Name" filter or
		//  - The intended usage of the returned type matcher is "inclusion"
		//		i.e. it is OK to include *all* resources that have this type
		//			 but is not OK to exclude *all* resources that have this
		//			 type
		if rr.Name == "" || usageInclusion {
			rtm = append(rtm, rr.ResourceTypeRequirement)
		}
	}
	return rtm
}

// Any returns true if the specified resource matches any of the requirements
// in `ResourceMatcher`
func (g ResourceMatcher) Any(name string, gvr schema.GroupVersionResource) bool {
	for _, req := range g {
		if req.Matches(name, gvr) {
			return true
		}
	}
	return false
}

// All returns true if the specified resource matches all of the requirements
// in `ResourceMatcher`
func (g ResourceMatcher) All(name string, gvr schema.GroupVersionResource) bool {
	for _, req := range g {
		if !req.Matches(name, gvr) {
			return false
		}
	}
	return true
}

// Resource represents a named Kubernetes object (name + GVR). This provides
// methods to use for filtering/selection.
//
// Note: This does not include 'Namespace'. The assumption is that the caller
// is responsible for determining what namespace scope to use for filtering
type Resource struct {
	Name string
	GVR  schema.GroupVersionResource
}

// ResourceList is a collection of Resource objects
type ResourceList []Resource

// Include returns any resources from the ResourceList that
// match the criteria in the specified ResourceMatcher
func (rl ResourceList) Include(m ResourceMatcher) ResourceList {
	return rl.apply(m, false)
}

// Exclude returns any resources from the ResourceList that
// do not match the criteria in the specified ResourceMatcher
func (rl ResourceList) Exclude(m ResourceMatcher) ResourceList {
	return rl.apply(m, true)
}

func (rl ResourceList) apply(m ResourceMatcher, exclude bool) ResourceList {
	if m.Empty() {
		return rl
	}
	filtered := make([]Resource, 0, len(rl))
	for _, r := range rl {
		if exclude != m.Any(r.Name, r.GVR) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
