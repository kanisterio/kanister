// Copyright 2024 The Kanister Authors.
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

package ksprig_test

import (
	"errors"
	"strings"
	"testing"
	"text/template"

	"github.com/kanisterio/kanister/pkg/ksprig"
)

func TestTemplateErrorsForUnsupportedFuncs(t *testing.T) {
	testCases := []struct {
		function     string
		templateText string
	}{
		{
			function:     "bcrypt",
			templateText: "{{bcrypt \"password\"}}",
		},
		{
			function:     "derivePassword",
			templateText: "{{derivePassword 1 \"long\" \"password\" \"user\" \"example.com\"}}",
		},
		{
			function:     "genPrivateKey",
			templateText: "{{genPrivateKey \"dsa\"}}",
		},
		{
			function:     "htpasswd",
			templateText: "{{htpasswd \"username\" \"password\"}}",
		},
	}

	for _, tc := range testCases {
		funcMap := ksprig.TxtFuncMap()

		t.Run(tc.function, func(t *testing.T) {
			if _, ok := funcMap[tc.function]; !ok {
				t.Skipf("Function %s is not supported by sprig.TxtFuncMap()", tc.function)
			}

			temp, err := template.New("test").Funcs(funcMap).Parse(tc.templateText)
			if err != nil {
				t.Fatalf("Unexpected template parse error: %s", err)
			}

			err = temp.Execute(nil, "")
			if err == nil {
				t.Fatal("Unexpected success for template execution")
			}

			if !errors.As(err, &ksprig.UnsupportedSprigUsageErr{}) {
				t.Fatalf("Expected error of type UnsupportedSprigFuncErr")
			}
		})
	}
}

func TestTemplateWorksForSupportedFuncs(t *testing.T) {
	testCases := []struct {
		description  string
		function     string
		templateText string
	}{
		// The supported funcs are not limited to these test cases
		{
			description:  "genPrivateKey for rsa key",
			function:     "genPrivateKey",
			templateText: "{{genPrivateKey \"rsa\"}}",
		},
		{
			description:  "genPrivateKey for ecdsa key",
			function:     "genPrivateKey",
			templateText: "{{genPrivateKey \"ecdsa\"}}",
		},
		{
			description:  "genPrivateKey for ed25519 key",
			function:     "genPrivateKey",
			templateText: "{{genPrivateKey \"ed25519\"}}",
		},
	}

	for _, tc := range testCases {
		funcMap := ksprig.TxtFuncMap()

		t.Run(tc.description, func(t *testing.T) {
			if _, ok := funcMap[tc.function]; !ok {
				t.Skipf("Function %s is not supported by sprig.TxtFuncMap()", tc.function)
			}

			temp, err := template.New("test").Funcs(funcMap).Parse(tc.templateText)
			if err != nil {
				t.Fatalf("Unexpected template parse error: %s", err)
			}

			err = temp.Execute(&strings.Builder{}, "")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
		})
	}
}
