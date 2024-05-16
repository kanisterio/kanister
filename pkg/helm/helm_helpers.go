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
	"regexp"
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

type Component struct {
	k8sType      K8sObjectType
	name         string
	originalDump string // This holds the dry run string output of the resource
}

const (
	K8sObjectTypeDeployment K8sObjectType = "deployment"
)

// ReleaseNameFromRenderedOutput takes as input the rendered output from a dry-run enabled HelmApp Install and tries to
// extract the release name from it for validation.
func ReleaseNameFromRenderedOutput(helmStatus string) string {
	re := regexp.MustCompile(`.*NAME:\s+(.*)\n`)
	withNameRE := regexp.MustCompile(`^Release\s+"(.*)"\s+`)
	tmpRelease := re.FindAllStringSubmatch(helmStatus, -1)
	log.Debug().Print("Parsed output for generate name install")
	if len(tmpRelease) < 1 {
		tmpRelease = withNameRE.FindAllStringSubmatch(helmStatus, -1)
		log.Debug().Print("Parsed output for specified name install/upgrade")
		if len(tmpRelease) < 1 {
			return ""
		}
	}
	if len(tmpRelease[0]) == 2 {
		return tmpRelease[0][1]
	}
	return ""
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
		ret = append(ret, Component{k8sType: K8sObjectType(strings.ToLower(tmpK8s.ObjKind)), name: tmpK8s.MetaData.Name, originalDump: objYaml})
	}
	return ret
}

func GetObjectFromComponent[T runtime.Object](component Component) (T, error) {
	var obj T
	var err error
	if err = yaml.Unmarshal([]byte(component.originalDump), &obj); err != nil {
		log.Error().Print("failed to Unmarshal k8s obj", field.M{"Error": err})
	}
	return obj, err
}

func GetFirstComponentOf(kind K8sObjectType, components []Component) *Component {
	for _, component := range components {
		if component.k8sType == kind {
			return &component
		}
	}
	return nil
}
