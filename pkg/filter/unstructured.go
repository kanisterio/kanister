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
