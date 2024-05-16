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
	"sigs.k8s.io/yaml"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type k8sObj struct {
	ObjKind  string            `json:"kind"`
	MetaData metav1.ObjectMeta `json:"metadata"`
}

type K8sObjectType string

type Component struct {
	k8sType K8sObjectType
	name    string
}

// ComponentsFromManifest is helper utility function that takes input the rendered output from dry-run enabled HelmApp Install and
// return a slice of struct Component. This struct holds basic information about all the resources that are going to be deployed when
// helm install is run in actual.
func ComponentsFromManifest(manifest string) []Component {
	var ret []Component
	// Get rid of the notes section
	parts := strings.Split(manifest, "NOTES:")
	/*
		Ignore the part that includes:
		  NAME: something

		  LAST DEPLOYED: something

		  NAMESPACE: something

		  STATUS: something

		  REVISION: something

		  TEST SUITE: something

		  HOOKS:

		  MANIFEST:
	*/

	// The actual rendered manifests start after first occurrence of `---`.
	// Before this we have chart details, for example Name, Last Deployed, Status etc.
	parts = strings.Split(parts[0], "---")
	for _, objYaml := range parts[1:] {
		tmpK8s := k8sObj{}
		if err := yaml.Unmarshal([]byte(objYaml), &tmpK8s); err != nil {
			log.Error().Print("failed to Unmarshal k8s obj", field.M{"Error": err})
			continue
		}
		ret = append(ret, Component{k8sType: K8sObjectType(strings.ToLower(tmpK8s.ObjKind)), name: tmpK8s.MetaData.Name})
	}
	return ret
}
