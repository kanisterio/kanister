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
	"fmt"
	"os/exec"
	"strings"
	"time"

	bp "github.com/kanisterio/blueprints"
	"k8s.io/apimachinery/pkg/util/rand"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	// imagePrefix specifies the prefix an image is going to have if it's being consumed from
	// kanister's ghcr registry
	imagePrefix = "ghcr.io/kanisterio"
)

// AppBlueprint implements Blueprint() to return Blueprint specs for the app
// Blueprint() returns the Blueprint located at {app-name}/{app-name}-blueprint.yaml in blueprint repository
type AppBlueprint struct {
	App          string
	Path         string
	UseDevImages bool
	IsEmbedded   bool
}

// PITRBlueprint embeds AppBlueprint and provides Blueprint functionality with PITR support.
// Blueprint() returns the Blueprint located at {app-name}/{app-name}-blueprint.yaml in blueprint repository
type PITRBlueprint struct {
	AppBlueprint
}

func NewBlueprint(app string, blueprintName string, blueprintPath string, useDevImages bool) Blueprinter {
	isEmbedded := false
	if blueprintPath == "" {
		blueprintPath = getBlueprintPath(app, blueprintName)
		isEmbedded = true
	}

	return &AppBlueprint{
		App:          app,
		Path:         blueprintPath,
		IsEmbedded:   isEmbedded,
		UseDevImages: useDevImages,
	}
}

func (b AppBlueprint) Blueprint() *crv1alpha1.Blueprint {
	var bpr *crv1alpha1.Blueprint
	var err error

	if b.IsEmbedded {
		// reads from embedded blueprints in blueprint repository
		bpr, err = bp.ReadFromEmbeddedFile(b.Path)
	} else {
		// reads from local file system
		bpr, err = bp.ReadFromFile(b.Path)
	}
	if err != nil {
		log.Error().WithError(err).Print("Failed to read Blueprint", field.M{"app": b.App})
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

	shortCommit := getShortCommitSHA()
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
				// ghcr.io/kanisterio/tools:v0.xx.x => ghcr.io/kanisterio/tools:short-commit-xxxxxxxx
				image := fmt.Sprintf("%s:short-commit-%s", strings.Split(imageStr, ":")[0], shortCommit)
				log.Info().Print("Updating image to use dev image", field.M{"image": image})
				phase.Args["image"] = image
				//fmt.Sprintf("%s:short-commit-%s", strings.Split(imageStr, ":")[0], shortCommit)
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
			IsEmbedded:   true,
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
	case "kafka":
		blueprintFolder = "kafka-adobe-s3-connector"
	default:
		blueprintFolder = blueprintName
	}

	return fmt.Sprintf("%s/%s-blueprint.yaml", blueprintFolder, blueprintName)
}

func getShortCommitSHA() string {
	cmd := exec.Command("git", "rev-parse", "--short=12", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		log.Error().WithError(err).Print("Failed to get git commit SHA")
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
