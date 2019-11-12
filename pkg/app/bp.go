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
	log "github.com/sirupsen/logrus"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	bp "github.com/kanisterio/kanister/pkg/blueprint"
)

// Blueprint implements Blueprint() to return Blueprint specs for the app
type Blueprint struct {
	app string
}

func NewBlueprint(app string) Blueprinter {
	return Blueprint{
		app: app,
	}
}

func (b Blueprint) Blueprint() *crv1alpha1.Blueprint {
	bpr, err := bp.ReadFromFile(b.app)
	if err != nil {
		log.Errorf("Failed to read Blueprint for %s: %s", b.app, err.Error())
	}
	return bpr
}
