package filter

import (
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
