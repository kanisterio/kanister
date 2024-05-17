// Copyright 2022 The Kanister Authors.
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

package helm

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type k8sObj struct {
	ObjKind  string            `json:"kind"`
	MetaData metav1.ObjectMeta `json:"metadata"`
}

type K8sObjectType string

const (
	K8sObjectTypeDeployment K8sObjectType = "deployment"
)

type RenderedResource struct {
	name             string
	renderedManifest string // This holds the dry run string output of the resource
}

type ResourceFilter func(kind K8sObjectType) bool

// ResourcesFromRenderedManifest extracts and optionally filters rendered resources from a given rendered manifest.
//
// This function processes a manifest string, splits it into individual resource YAMLs, and parses each resource into
// a RenderedResource struct. It can optionally filter resources based on a provided ResourceFilter function.
//
// Parameters:
//   - manifest: A string containing the full rendered manifest, which may include notes and multiple YAML documents.
//   - filter: A ResourceFilter function that returns true for the types of resources to include in the result. If nil,
//     all resources are included.
//
// Returns:
// - A slice of RenderedResource structs containing the name and the YAML of each resource that passes the filter.
func ResourcesFromRenderedManifest(manifest string, filter ResourceFilter) []RenderedResource {
	var ret []RenderedResource
	// Get rid of the notes section, shown at the very end of the output
	manifestSections := strings.Split(manifest, "NOTES:")
	// The actual rendered manifests start after first occurrence of `---`.
	// Before this we have chart details, for example Name, Last Deployed, Status etc.
	renderedResourcesYaml := strings.Split(manifestSections[0], "---")
	for _, resourceYaml := range renderedResourcesYaml[1:] {
		obj := k8sObj{}
		if err := yaml.Unmarshal([]byte(resourceYaml), &obj); err != nil {
			log.Error().Print("failed to Unmarshal k8s obj", field.M{"Error": err})
			continue
		}
		k8sType := K8sObjectType(strings.ToLower(obj.ObjKind))
		// Either append all rendered resource or filter using the filter func
		if filter == nil || filter(k8sType) {
			ret = append(ret, RenderedResource{
				name:             obj.MetaData.Name,
				renderedManifest: resourceYaml,
			})
		}
	}
	return ret
}

// GetK8sObjectsFromRenderedManifest unmarshals a list of rendered Kubernetes manifests
// into a map of Kubernetes objects.
//
// This function takes a slice of RenderedResource, each containing a rendered manifest
// in YAML format. It then unmarshals each manifest into an object of the specified type `T`,
// which must implement the runtime.Object interface. The unmarshaled objects are stored in
// a map where the keys are the names of the resources and the values are the unmarshaled objects.
//
// Type Parameters:
//
//	T - The type of Kubernetes objects to unmarshal, which must implement runtime.Object.
//
// Parameters:
//
//	resources - A slice of RenderedResource, where each element contains the name and
//	             the rendered manifest of a Kubernetes resource.
//
// Returns:
//
//	A map where the keys are the names of the resources and the values are the unmarshaled
//	Kubernetes objects of type `T`.
//	An error if any manifest fails to unmarshal.
func GetK8sObjectsFromRenderedManifest[T runtime.Object](resources []RenderedResource) (map[string]T, error) {
	var mapOfObjects = make(map[string]T)
	var err error
	for _, resource := range resources {
		var obj T
		if err = yaml.Unmarshal([]byte(resource.renderedManifest), &obj); err != nil {
			log.Error().Print("Failed to unmarshal k8s obj", field.M{"Error": err})
			return nil, err
		}
		mapOfObjects[resource.name] = obj
	}
	return mapOfObjects, nil
}
