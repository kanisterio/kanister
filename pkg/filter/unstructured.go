package filter

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Specs map[schema.GroupVersionResource][]unstructured.Unstructured

func (s Specs) keys() GroupVersionResourceList {
	gvrs := make(GroupVersionResourceList, 0, len(s))
	for gvr := range s {
		gvrs = append(gvrs, gvr)
	}
	return gvrs
}

func (s Specs) Include(ms ...ResourceTypeMatcher) Specs {
	gvrs := s.keys().Include(ms...)
	ret := make(Specs, len(gvrs))
	for _, gvr := range gvrs {
		ret[gvr] = s[gvr]
	}
	return ret
}

func (s Specs) Exclude(ms ...ResourceTypeMatcher) Specs {
	gvrs := s.keys().Exclude(ms...)
	ret := make(Specs, len(gvrs))
	for _, gvr := range gvrs {
		ret[gvr] = s[gvr]
	}
	return ret
}
