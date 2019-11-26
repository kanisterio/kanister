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
	"context"
	"time"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// App represents an application we can install into a namespace.
type App interface {
	// Init instantiates the app based on the environemnt configuration,
	// including environement variables and state in the Kubernetes cluster. If
	// any required configuration is not discoverable, Init will return an
	// error.
	Init(context.Context) error
	// Install will install the app into a specified namespace. Install should
	// be called after Init.
	Install(ctx context.Context, namespace string) error
	// IsReady returns true once an app is running. If we detect a failure state
	// that will not recover from on it's own, then IsReady will return an
	// error.
	IsReady(context.Context) (bool, error)
	// Object returns a reference to this app to be used by ActionSets.
	Object() crv1alpha1.ObjectReference
	// Uninstall deletes an app and all it's components from a Namespace.
	Uninstall(context.Context) error
}

// DatabaseApp inherits methods from App, but also includes method to add, read
// and remove entries stored byt the App.
type DatabaseApp interface {
	App
	// Ping will issue trivial request to the database to see if it is
	// accessable.
	Ping(context.Context) error
	// Insert adds n entries to the database.
	Insert(ctx context.Context) error
	// Count returns the number of entries in the database.
	Count(context.Context) (int, error)
	// Reset Removes all entries from the database.
	Reset(context.Context) error
}

// ConfigApp describes an App installs additional configuration that can be
// referenced from an ActionSet and used in a Blueprint. Not all apps will
// create this additional configuration.
type ConfigApp interface {
	App
	// ConfigMaps returns named references to ConfigMaps when installing App.
	ConfigMaps() map[string]crv1alpha1.ObjectReference
	// Secrets returns named references to Secrets when installing App.
	Secrets() map[string]crv1alpha1.ObjectReference
}

// Blueprinter is the interface used to create a Blueprint.
type Blueprinter interface {
	// Blueprint returns a new non-namespaced Blueprint.
	Blueprint() *crv1alpha1.Blueprint
}

// PITRBlueprinter is optionally implemented if a Blueprint supports
// point-in-time recovery.
type PITRBlueprinter interface {
	Blueprinter
	// FormatPITR takes a time.Time struct and returns a string for use by a
	// Blueprint that supports PITR.
	FormatPITR(time.Time) string
}
