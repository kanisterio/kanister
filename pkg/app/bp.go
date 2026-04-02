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

package app

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kanisterio/blueprints"
	bpPathUtil "github.com/kanisterio/blueprints/utils"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	// imagePrefix specifies the prefix an image is going to have if it's being consumed from
	// kanister's ghcr registry
	imagePrefix = "ghcr.io/kanisterio"

	// default dev tag for kanister images
	DefaultImageTag = "v9.99.9-dev"

	defaultImageRegistry = "ghcr.io"
	defaultImageOrg      = "kanisterio"
)

// AppBlueprint implements Blueprint() to return Blueprint specs for the app
// Blueprint() returns the blueprint from the blueprint repository or the given path.
// An empty path defaults to {app-name}/{app-name}-blueprint.yaml in the repository.
type AppBlueprint struct {
	App            string
	Path           string
	readFromBPRepo bool
	devImageTag    string
}

// PITRBlueprint implements Blueprint() to return Blueprint with PITR
// Blueprint() returns the blueprint from the blueprint repository located at {app-name}/{app-name}-blueprint.yaml.
type PITRBlueprint struct {
	AppBlueprint
}

func NewBlueprint(app string, blueprintName string, blueprintPath string, devImageTag string) Blueprinter {
	isEmbeddedBlueprint := false

	if blueprintPath == "" {
		blueprintPath = bpPathUtil.GetBlueprintPathByName(app, blueprintName)
		isEmbeddedBlueprint = true
	}
	return &AppBlueprint{
		App:            app,
		Path:           blueprintPath,
		readFromBPRepo: isEmbeddedBlueprint,
		devImageTag:    devImageTag,
	}
}

func (b AppBlueprint) Blueprint() *crv1alpha1.Blueprint {
	var bpData []byte
	var err error

	if b.readFromBPRepo {
		bpData, err = blueprints.ReadFromEmbeddedFile(b.Path)
	} else {
		bpData, err = blueprints.ReadFromFile(b.Path)
	}
	if err != nil {
		log.Error().WithError(err).Print("Failed to read Blueprint", field.M{"app": b.App})
		return nil
	}

	bpr, err := ParseBlueprint(bpData)
	if err != nil {
		log.Error().WithError(err).Print("Failed to parse Blueprint", field.M{"app": b.App})
		return nil
	}

	// set the name to a dynamically generated value
	// so that the name wont conflict with the same application
	// installed using other ways
	bpr.ObjectMeta.Name = fmt.Sprintf("%s-%s", bpr.ObjectMeta.Name, rand.String(5))

	if b.devImageTag != "" {
		updateImageTags(bpr, b.devImageTag)
	}
	return bpr
}

func updateImageTags(bp *crv1alpha1.Blueprint, devTag string) {
	if bp == nil {
		return
	}
	registry := imageRegistry()
	org := imageOrg()
	tag := imageTag(devTag)
	customSource := registry != defaultImageRegistry || org != defaultImageOrg

	for _, a := range bp.Actions {
		for _, phase := range a.Phases {
			image, ok := phase.Args["image"]
			if !ok {
				continue
			}
			imageStr, ok := image.(string)
			if !ok {
				continue
			}

			if strings.HasPrefix(imageStr, imagePrefix) {
				repo, _ := splitImage(imageStr)
				parts := strings.Split(repo, "/")
				appName := parts[len(parts)-1]
				var updatedImage string
				if registry != "" {
					updatedImage = fmt.Sprintf("%s/%s/%s:%s", registry, org, appName, tag)
				} else {
					updatedImage = fmt.Sprintf("%s/%s:%s", org, appName, tag)
				}
				phase.Args["image"] = updatedImage
				log.Info().Print("updated image", field.M{"image": updatedImage})
			}

			// Use IfNotPresent for custom image sources (local registry or kind-loaded images)
			// so Kubernetes uses the locally available image without forcing a remote pull.
			// Released images from ghcr.io use Always to stay current.
			pullPolicy := "Always"
			if customSource {
				pullPolicy = "IfNotPresent"
			}
			phase.Args["podOverride"] = crv1alpha1.JSONMap{
				"containers": []map[string]interface{}{
					{
						"name":            "container",
						"imagePullPolicy": pullPolicy,
					},
				},
			}
		}
	}
}

// NewPITRBlueprint returns blueprint located at {app-name}/{app-name}-blueprint.yaml in blueprint repository
func NewPITRBlueprint(app string, blueprintName string, devImageTag string) Blueprinter {
	return &PITRBlueprint{
		AppBlueprint{
			App:         app,
			Path:        bpPathUtil.GetBlueprintPathByName(app, blueprintName),
			devImageTag: devImageTag,
		},
	}
}

func (b PITRBlueprint) FormatPITR(pitr time.Time) string {
	return pitr.UTC().Format("2006-01-02T15:04:05Z")
}

// ParseBlueprint parses YAML data into a Blueprint struct
func ParseBlueprint(data []byte) (*crv1alpha1.Blueprint, error) {
	var bp crv1alpha1.Blueprint
	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 1000)
	if err := dec.Decode(&bp); err != nil {
		return nil, err
	}
	return &bp, nil
}

// imageRegistry returns the container registry to use for kanister images.
// Defaults to ghcr.io; override with IMAGE_REGISTRY env var.
// Setting IMAGE_REGISTRY to an empty string disables the registry prefix
// (e.g. for kind-loaded images that have no registry component).
func imageRegistry() string {
	v, ok := os.LookupEnv("IMAGE_REGISTRY")
	if ok {
		return v
	}
	return defaultImageRegistry
}

// imageOrg returns the org/namespace within the registry to use for kanister images.
// Defaults to "kanisterio"; override with IMAGE_ORG env var.
// For kind-loaded images without a registry, set IMAGE_ORG to include the repository
// path component (e.g. "kanisterio/test-images") and leave IMAGE_REGISTRY unset.
func imageOrg() string {
	if v := os.Getenv("IMAGE_ORG"); v != "" {
		return v
	}
	return defaultImageOrg
}

// imageTag returns the tag to use for kanister images.
// Falls back to devTag if IMAGE_TAG env var is not set.
func imageTag(devTag string) string {
	if v := os.Getenv("IMAGE_TAG"); v != "" {
		return v
	}
	return devTag
}

func splitImage(image string) (repo string, tag string) {
	image = strings.Split(image, "@")[0] // drop digest if present

	lastColon := strings.LastIndex(image, ":")
	lastSlash := strings.LastIndex(image, "/")

	if lastColon > lastSlash {
		return image[:lastColon], image[lastColon+1:]
	}

	return image, "" // no tag
}
