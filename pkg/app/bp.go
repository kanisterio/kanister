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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	bp "github.com/kanisterio/kanister/pkg/blueprint"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	blueprintsRepo = "./blueprints"
	// imagePrefix specifies the prefix an image is going to have if it's being consumed from
	// kanister's ghcr registry
	imagePrefix = "ghcr.io/kanisterio"
)

// AppBlueprint implements Blueprint() to return Blueprint specs for the app
// Blueprint() returns Blueprint placed at ./blueprints/{app-name}-blueprint.yaml
type AppBlueprint struct {
	App          string
	Path         string
	UseDevImages bool
}

// PITRBlueprint implements Blueprint() to return Blueprint with PITR
// Blueprint() returns Blueprint placed at ./blueprints/{app-name}-blueprint.yaml
type PITRBlueprint struct {
	AppBlueprint
}

func NewBlueprint(app string, bpReposPath string, useDevImages bool) Blueprinter {
	if bpReposPath == "" {
		bpReposPath = blueprintsRepo
	}
	return &AppBlueprint{
		App:          app,
		Path:         fmt.Sprintf("%s/%s-blueprint.yaml", bpReposPath, app),
		UseDevImages: useDevImages,
	}
}

func (b AppBlueprint) Blueprint() *crv1alpha1.Blueprint {
	bpr, err := bp.ReadFromFile(b.Path)
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

// NewPITRBlueprint returns blueprint placed at ./blueprints/{app-name}-blueprint.yaml
func NewPITRBlueprint(app string, bpReposPath string, useDevImages bool) Blueprinter {
	if bpReposPath == "" {
		bpReposPath = blueprintsRepo
	}
	return &PITRBlueprint{
		AppBlueprint{
			App:          app,
			Path:         fmt.Sprintf("%s/%s-blueprint.yaml", bpReposPath, app),
			UseDevImages: useDevImages,
		},
	}
}

func (b PITRBlueprint) FormatPITR(pitr time.Time) string {
	return pitr.UTC().Format("2006-01-02T15:04:05Z")
}
