package filter

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Specs map[schema.GroupVersionResource][]unstructured.Unstructured

func (s Specs) keys() []schema.GroupVersionResource {
	gvrs := make([]schema.GroupVersionResource, 0, len(s))
	for gvr := range s {
		gvrs = append(gvrs, gvr)
	}
	return gvrs
}

func (s Specs) Include(g ResourceMatcher) Specs {
	gvrs := g.Include(s.keys())
	ret := make(Specs, len(gvrs))
	for _, gvr := range gvrs {
		ret[gvr] = s[gvr]
	}
	return ret
}

func (s Specs) Exclude(g ResourceMatcher) Specs {
	gvrs := g.Exclude(s.keys())
	ret := make(Specs, len(gvrs))
	for _, gvr := range gvrs {
		ret[gvr] = s[gvr]
	}
	return ret
}
