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

// Package main provides the entry point for the kanctl command-line tool,
// which is part of the Kanister project. Kanister is a framework for
// application-level data management on Kubernetes. The kanctl tool allows
// users to interact with Kanister resources and perform various operations
// such as creating, executing, and managing blueprints and actions.
package main

import (
	"github.com/kanisterio/kanister/pkg/kanctl"
	"github.com/kanisterio/kanister/pkg/log"
)

func init() {
	// We silence all non-fatal log messages.
	// logrus.SetLevel(logrus.ErrorLevel)
}

func main() {
	log.SetupClusterNameInLogVars()

	kanctl.Execute()
}
