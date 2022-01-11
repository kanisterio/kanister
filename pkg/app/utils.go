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

	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	dbTemplateURI = "https://raw.githubusercontent.com/openshift/origin/%s/examples/db-templates/%s-%s-template.json"
	// PersistentStorage can be used if we want to deploy database with Persistent Volumes
	PersistentStorage storage = "persistent" // nolint:varcheck

	// EphemeralStorage can be used if we don't want to deploy database with Persistent
	EphemeralStorage storage = "ephemeral"
	// TemplateVersionOCP3_11 stores version of db template 3.11
	TemplateVersionOCP3_11 DBTemplate = "release-3.11"
	// TemplateVersionOCP4_4 stores version of db template 4.4
	TemplateVersionOCP4_4 DBTemplate = "release-4.4"
	// TemplateVersionOCP4_5 stores version of db template 4.5
	TemplateVersionOCP4_5 DBTemplate = "release-4.5"
	// TemplateVersionOCP4_9 stores version of db template 4.9
	TemplateVersionOCP4_9 DBTemplate = "release-4.9"
)

type storage string

// DBTemplate is type of openshift db template version
type DBTemplate string

// appendRandString, appends a random string to the passed string value
func appendRandString(name string) string {
	return fmt.Sprintf("%s-%s", name, rand.String(5))
}

// getOpenShiftDBTemplate accepts the application name and returns the
// db template for that application
// https://github.com/openshift/origin/tree/master/examples/db-templates
func getOpenShiftDBTemplate(appName string, templateVersion DBTemplate, storageType storage) string {
	return fmt.Sprintf(dbTemplateURI, templateVersion, appName, storageType)
}

// getLabelOfApp returns label of the passed application this label can be
// used to delete all the resources that were created while deploying this application
func getLabelOfApp(appName string, storageType storage) string {
	return fmt.Sprintf("app=%s-%s", appName, storageType)
}
