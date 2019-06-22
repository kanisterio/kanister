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
	m := joinResourceTypeMatchers(ms...)
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

func joinResourceTypeMatchers(ms ...ResourceTypeMatcher) ResourceTypeMatcher {
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
