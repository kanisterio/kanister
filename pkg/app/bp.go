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
	"strings"
	"time"

	"github.com/kanisterio/blueprints"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// imagePrefix specifies the prefix an image is going to have if it's being consumed from
	// kanister's ghcr registry
	imagePrefix = "ghcr.io/kanisterio"
)

// AppBlueprint implements Blueprint() to return Blueprint specs for the app
// Blueprint() returns the Blueprint located at {app-name}/{app-name}-blueprint.yaml in blueprint repository
type AppBlueprint struct {
	App           string
	Path          string
	UseDevImages  bool
	isKanisterApp bool
}

// PITRBlueprint implements Blueprint() to return Blueprint with PITR
// Blueprint() returns Blueprint located at {app-name}/{app-name}-blueprint.yaml in blueprint repository
type PITRBlueprint struct {
	AppBlueprint
}

func NewBlueprint(app string, blueprintName string, blueprintPath string, useDevImages bool) Blueprinter {
	isKanisterApp := false

	if blueprintPath == "" {
		blueprintPath = getBlueprintPath(app, blueprintName)
		isKanisterApp = true
	}
	return &AppBlueprint{
		App:           app,
		Path:          blueprintPath,
		UseDevImages:  useDevImages,
		isKanisterApp: isKanisterApp,
	}
}

func (b AppBlueprint) Blueprint() *crv1alpha1.Blueprint {
	var bpData []byte
	var err error

	if b.isKanisterApp {
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

	if b.UseDevImages {
		updateImageTags(bpr)
	}
	return bpr
}

func updateImageTags(bp *crv1alpha1.Blueprint) {
	if bp == nil {
		return
	}
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
				// ghcr.io/kanisterio/tools:v0.xx.x => ghcr.io/kanisterio/tools:v9.99.9-dev
				phase.Args["image"] = fmt.Sprintf("%s:v9.99.9-dev", strings.Split(imageStr, ":")[0])
			}

			// Change imagePullPolicy to Always using podOverride config
			phase.Args["podOverride"] = crv1alpha1.JSONMap{
				"containers": []map[string]interface{}{
					{
						"name":            "container",
						"imagePullPolicy": "Always",
					},
				},
			}
		}
	}
}

// NewPITRBlueprint returns blueprint located at {app-name}/{app-name}-blueprint.yaml in blueprint repository
func NewPITRBlueprint(app string, blueprintName string, useDevImages bool) Blueprinter {
	return &PITRBlueprint{
		AppBlueprint{
			App:          app,
			Path:         getBlueprintPath(app, blueprintName),
			UseDevImages: useDevImages,
		},
	}
}

func (b PITRBlueprint) FormatPITR(pitr time.Time) string {
	return pitr.UTC().Format("2006-01-02T15:04:05Z")
}

func getBlueprintPath(app string, blueprintName string) string {
	var blueprintFolder string
	// If blueprintName is not provided, use app name as blueprint name
	if blueprintName == "" {
		blueprintName = app
	}
	switch app {
	case "rds-aurora-snap":
		blueprintFolder = "aws-rds-aurora-mysql"
	case "rds-postgres-snap", "rds-postgres", "rds-postgres-dump":
		blueprintFolder = "aws-rds-postgres"
	case "kafka":
		blueprintFolder = "kafka-adobe-s3-connector"
	default:
		blueprintFolder = blueprintName
	}

	return fmt.Sprintf("%s/%s-blueprint.yaml", blueprintFolder, blueprintName)
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
