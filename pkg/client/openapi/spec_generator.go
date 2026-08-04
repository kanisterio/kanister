/*
Copyright 2025 by contributors to the Kanister project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package openapi renders an OpenAPI spec for the Kanister CR types, which
// applyconfiguration-gen consumes to generate apply configurations.
package openapi

import (
	"encoding/json"
	"io"
	"strings"

	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/util"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// crAPIPackage is the Go import path holding the Kanister CR types.
const crAPIPackage = "github.com/kanisterio/kanister/pkg/apis/cr/"

func buildSwagger() (*spec.Swagger, error) {
	config := &common.Config{
		Info: &spec.Info{
			InfoProps: spec.InfoProps{
				Title:   "Kanister",
				Version: "v1alpha1",
			},
		},
		GetDefinitions: GetOpenAPIDefinitions,
		// applyconfiguration-gen looks definitions up by their REST-friendly name.
		GetDefinitionName: func(name string) (string, spec.Extensions) {
			return util.ToRESTFriendlyName(name), nil
		},
	}

	defs := GetOpenAPIDefinitions(func(string) spec.Ref { return spec.Ref{} })

	// Build the spec from Kanister types and their referenced dependencies.
	names := make([]string, 0, len(defs))
	for name := range defs {
		if strings.HasPrefix(name, crAPIPackage) {
			names = append(names, name)
		}
	}

	return builder.BuildOpenAPIDefinitionsForResources(config, names...)
}

func WriteSpec(w io.Writer) error {
	swagger, err := buildSwagger()
	if err != nil {
		return err
	}

	return json.NewEncoder(w).Encode(swagger)
}
