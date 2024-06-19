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
	name string
	// renderedManifest holds the dry run raw yaml of the resource.
	renderedManifest string
}

type ResourceFilter func(kind K8sObjectType) bool

// ResourcesFromRenderedManifest extracts optionally filtered raw resource yamls from a given rendered manifest.
func ResourcesFromRenderedManifest(manifest string, filter ResourceFilter) []RenderedResource {
	var ret []RenderedResource
	// Get rid of the notes section, shown at the very end of the output.
	manifestSections := strings.Split(manifest, "NOTES:")
	// The actual rendered manifests start after first occurrence of `---`.
	// Before this we have chart details, for example Name, Last Deployed, Status etc.
	renderedResourcesYaml := strings.Split(manifestSections[0], "---")
	for _, resourceYaml := range renderedResourcesYaml[1:] {
		obj := k8sObj{}
		if err := yaml.Unmarshal([]byte(resourceYaml), &obj); err != nil {
			log.Error().Print("Failed to unmarshal k8s obj", field.M{"Error": err})
			continue
		}
		k8sType := K8sObjectType(strings.ToLower(obj.ObjKind))
		// Either append all rendered resource or filter.
		if filter == nil || filter(k8sType) {
			ret = append(ret, RenderedResource{
				name:             obj.MetaData.Name,
				renderedManifest: resourceYaml,
			})
		}
	}
	return ret
}

// K8sObjectsFromRenderedResources unmarshals a list of rendered Kubernetes manifests
// into a map of Kubernetes objects name and object itself.
func K8sObjectsFromRenderedResources[T runtime.Object](resources []RenderedResource) (map[string]T, error) {
	var nameAndObj = make(map[string]T)
	var err error
	for _, resource := range resources {
		var obj T
		if err = yaml.Unmarshal([]byte(resource.renderedManifest), &obj); err != nil {
			log.Error().Print("Failed to unmarshal rendered resource yaml to K8s obj", field.M{"Error": err})
			return nil, err
		}
		nameAndObj[resource.name] = obj
	}
	return nameAndObj, nil
}
