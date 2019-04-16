package filter

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceRequirement struct {
	Group    string
	Version  string
	Resource string
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

func (g ResourceMatcher) Include(gvrs []schema.GroupVersionResource) []schema.GroupVersionResource {
	return g.apply(gvrs, false)
}

func (g ResourceMatcher) Exclude(gvrs []schema.GroupVersionResource) []schema.GroupVersionResource {
	return g.apply(gvrs, true)
}

func (g ResourceMatcher) apply(gvrs []schema.GroupVersionResource, exclude bool) []schema.GroupVersionResource {
	if g.Empty() {
		return gvrs
	}
	filtered := make([]schema.GroupVersionResource, 0, len(gvrs))
	for _, gvr := range gvrs {
		if exclude != g.Any(gvr) {
			filtered = append(filtered, gvr)
		}
	}
	return filtered
}
