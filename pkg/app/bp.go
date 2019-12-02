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
	"time"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	bp "github.com/kanisterio/kanister/pkg/blueprint"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	blueprintsRepo = "./blueprints"
)

// Blueprint implements Blueprint() to return Blueprint specs for the app
// Blueprint() returns Blueprint placed at ./blueprints/{app-name}-blueprint.yaml
type AppBlueprint struct {
	app  string
	path string
}

// PITRBlueprint implements Blueprint() to return Blueprint with PITR
// Blueprint() returns Blueprint placed at ./blueprints/{app-name}-blueprint.yaml
type PITRBlueprint struct {
	AppBlueprint
}

func NewBlueprint(app string) Blueprinter {
	return &AppBlueprint{
		app:  app,
		path: fmt.Sprintf("%s/%s-blueprint.yaml", blueprintsRepo, app),
	}
}

func (b AppBlueprint) Blueprint() *crv1alpha1.Blueprint {
	bpr, err := bp.ReadFromFile(b.path)
	if err != nil {
		log.Error().WithError(err).Print("Failed to read Blueprint", field.M{"app": b.app})
	}
	return bpr
}

// Blueprint returns Blueprint placed at ./blueprints/{app-name}-blueprint.yaml
func NewPITRBlueprint(app string) Blueprinter {
	return &PITRBlueprint{
		AppBlueprint{
			app:  app,
			path: fmt.Sprintf("%s/%s-blueprint.yaml", blueprintsRepo, app),
		},
	}
}

func (b PITRBlueprint) FormatPITR(pitr time.Time) string {
	return pitr.UTC().Format("2006-01-02T15:04:05Z")
}
