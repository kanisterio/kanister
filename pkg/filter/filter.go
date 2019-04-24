package filter

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceRequirement struct {
	Group    string `json:"group,omitempty"`
	Version  string `json:"version,omitempty"`
	Resource string `json:"resource,omitempty"`
}

func (r ResourceRequirement) Matches(gvr schema.GroupVersionResource) bool {
	return matches(r.Group, gvr.Group) && matches(r.Version, gvr.Version) && matches(r.Resource, gvr.Resource)
}

func matches(sel, val string) bool {
	return sel == "" || sel == val
}

type ResourceMatcher []ResourceRequirement

func (g ResourceMatcher) Empty() bool {
	return len(g) == 0
}

func (g ResourceMatcher) Any(gvr schema.GroupVersionResource) bool {
	for _, req := range g {
		if req.Matches(gvr) {
			return true
		}
	}
	return false
}

func (g ResourceMatcher) All(gvr schema.GroupVersionResource) bool {
	for _, req := range g {
		if !req.Matches(gvr) {
			return false
		}
	}
	return true
}

type GroupVersionResourceList []schema.GroupVersionResource

func (g GroupVersionResourceList) Include(ms ...ResourceMatcher) GroupVersionResourceList {
	return g.apply(ms, false)
}

func (g GroupVersionResourceList) Exclude(ms ...ResourceMatcher) GroupVersionResourceList {
	return g.apply(ms, true)
}

func (g GroupVersionResourceList) apply(ms []ResourceMatcher, exclude bool) GroupVersionResourceList {
	m := joinResourceMatchers(ms...)
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

func joinResourceMatchers(ms ...ResourceMatcher) ResourceMatcher {
	n := 0
	for _, m := range ms {
		n += len(m)
	}
	gvr := make(ResourceMatcher, n)
	i := 0
	for _, m := range ms {
		copy(gvr[i:], []ResourceRequirement(m))
		i += len(m)
	}
	return gvr
}
