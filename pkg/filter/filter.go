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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceTypeRequirement contains group, version and resource values
type ResourceTypeRequirement struct {
	Group    string `json:"group,omitempty"`
	Version  string `json:"version,omitempty"`
	Resource string `json:"resource,omitempty"`
}

// K8sCoreGroupExactMatch is sentinel value that only matches K8s core group of ""
const K8sCoreGroupExactMatch = "core"

// Matches returns true if group, version and resource values match or are empty
// Group value of K8sCoreGroupExactMatch only matches K8s core group of ""
func (r ResourceTypeRequirement) Matches(gvr schema.GroupVersionResource) bool {
	var groupMatch bool
	if r.Group == K8sCoreGroupExactMatch {
		groupMatch = gvr.Group == ""
	} else {
		groupMatch = matches(r.Group, gvr.Group)
	}
	return groupMatch && matches(r.Version, gvr.Version) && matches(r.Resource, gvr.Resource)
}

// Empty returns true if ResourceTypeRequirement has no fields set
func (rtr ResourceTypeRequirement) Empty() bool {
	return rtr.Group == "" && rtr.Version == "" && rtr.Resource == ""
}

func matches(sel, val string) bool {
	return sel == "" || sel == val
}

// ResourceTypeMatcher is a collection of ResourceTypeRequirement objects
type ResourceTypeMatcher []ResourceTypeRequirement

// Empty returns true if ResourceTypeMatcher collection has no objects
func (rtm ResourceTypeMatcher) Empty() bool {
	return len(rtm) == 0
}

// Any returns true if there are any GVR matches in the collection
func (rtm ResourceTypeMatcher) Any(gvr schema.GroupVersionResource) bool {
	for _, req := range rtm {
		if req.Matches(gvr) {
			return true
		}
	}
	return false
}

// All returns true if all resources in the collection match
func (rtm ResourceTypeMatcher) All(gvr schema.GroupVersionResource) bool {
	for _, req := range rtm {
		if !req.Matches(gvr) {
			return false
		}
	}
	return true
}

// ResourceMatcher constructs a resource matcher
// based on a `ResourceTypeMatcher`
func (rtm ResourceTypeMatcher) ResourceMatcher() ResourceMatcher {
	rm := make(ResourceMatcher, 0, len(rtm))
	for _, rtr := range rtm {
		rm = append(rm, ResourceRequirement{ResourceTypeRequirement: rtr})
	}
	return rm
}

// GroupVersionResourceList is a collection of GroupVersionResource objects
type GroupVersionResourceList []schema.GroupVersionResource

// Include returns a GroupVersionResourceList that should be included according to the ResourceTypeMatcher filter
func (g GroupVersionResourceList) Include(ms ...ResourceTypeMatcher) GroupVersionResourceList {
	return g.apply(ms, false)
}

// Exclude returns a GroupVersionResourceList with resources removed that should be excluded according to the ResourceTypeMatcher filter
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
	// Provides the Name of the resource object
	corev1.LocalObjectReference `json:",inline,omitempty"`
	// Provides the Group, Version, and Resource values (GVR)
	ResourceTypeRequirement `json:",inline,omitempty"`
	// Specifies a set of label requirements to be used as filters for matches
	metav1.LabelSelector `json:",inline,omitempty"`
}

// DeepCopyInto provides explicit deep copy implementation to avoid
func (r ResourceRequirement) DeepCopyInto(out *ResourceRequirement) {
	r.LocalObjectReference.DeepCopyInto(&out.LocalObjectReference)
	out.ResourceTypeRequirement = r.ResourceTypeRequirement
	r.LabelSelector.DeepCopyInto(&out.LabelSelector)
}

// Matches returns true if the specified resource name/GVR/labels matches the requirement
func (r ResourceRequirement) Matches(name string, gvr schema.GroupVersionResource, resourceLabels map[string]string) bool {
	// If Name or GVR is not a match, return false
	// Empty string for Name or GVR will match any value
	if !matches(r.Name, name) || !r.ResourceTypeRequirement.Matches(gvr) {
		return false
	}
	if len(r.MatchExpressions) == 0 && len(r.MatchLabels) == 0 {
		// If there is no LabelSelector we return match of Name/GVR
		return true
	}
	sel, err := metav1.LabelSelectorAsSelector(&r.LabelSelector)
	if err != nil {
		// A match was found on Name or GVR but labels could not be evaluated
		// and resource labels were provided, so false is returned.
		return false
	}
	return sel.Matches(labels.Set(resourceLabels))
}

// ResourceMatcher is a collection of ResourceRequirement objects for filtering resources
type ResourceMatcher []ResourceRequirement

// Empty returns true if ResourceMatcher has no ResourceRequirements
func (rm ResourceMatcher) Empty() bool {
	return len(rm) == 0
}

// TypeMatcher constructs a resource type matcher
// based on a `ResourceMatcher`
//
// The `usageInclusion` flag should be set to true
// if the type matcher will be used as an include filter
func (rm ResourceMatcher) TypeMatcher(usageInclusion bool) ResourceTypeMatcher {
	rtm := make(ResourceTypeMatcher, 0, len(rm))
	for _, rr := range rm {
		// Include the type requirement from the ResourceRequirement, if
		//  - There is no "Name" and "Labels" filter or
		//  - The intended usage of the returned type matcher is "inclusion"
		//		i.e. it is OK to include *all* resources that have this type
		//			 but it is not OK to exclude *all* resources that have this
		//			 type when name or labels are specified.
		if usageInclusion || (rr.Name == "" &&
			len(rr.LabelSelector.MatchLabels) == 0 &&
			len(rr.LabelSelector.MatchExpressions) == 0) {
			rtm = append(rtm, rr.ResourceTypeRequirement)
		}
	}
	return rtm
}

// Any returns true if the specified resource matches any of the requirements
// in `ResourceMatcher`
func (rm ResourceMatcher) Any(name string, gvr schema.GroupVersionResource, resourceLabels map[string]string) bool {
	for _, req := range rm {
		if req.Matches(name, gvr, resourceLabels) {
			return true
		}
	}
	return false
}

// All returns true if the specified resource matches all of the requirements
// in `ResourceMatcher`
func (rm ResourceMatcher) All(name string, gvr schema.GroupVersionResource, resourceLabels map[string]string) bool {
	for _, req := range rm {
		if !req.Matches(name, gvr, resourceLabels) {
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
	Name           string
	GVR            schema.GroupVersionResource
	ResourceLabels map[string]string
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
		if exclude != m.Any(r.Name, r.GVR, r.ResourceLabels) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
