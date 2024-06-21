// Copyright 2021 The Kanister Authors.
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

package jsonpath

import (
	"bytes"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
)

// jsonpathRegex represents pattern in which jsonpath is specified in the wait conditions
// e.g { $.status.phase }
var jsonpathRegex = regexp.MustCompile(`(?m){\s*\$([^}]*)`)

// FindJsonpathArgs returns matched jsonpath args in the string
func FindJsonpathArgs(s string) map[string]string {
	matchMap := make(map[string]string)
	for _, matchList := range jsonpathRegex.FindAllSubmatch([]byte(s), -1) {
		matchedSource := ""
		for i := range matchList {
			if i == 0 {
				// Add ending "}" excluded by regex
				matchedSource = string(matchList[i]) + "}"
				continue
			}
			matchMap[matchedSource] = string(matchList[i])
		}
	}
	return matchMap
}

// ResolveJsonpathToString resolves jsonpath value from the k8s resource object
func ResolveJsonpathToString(obj runtime.Object, jsonpathStr string) (string, error) {
	var buff bytes.Buffer
	jp, err := printers.NewJSONPathPrinter(jsonpathStr)
	if err != nil {
		return "", nil
	}
	err = jp.PrintObj(obj, &buff)
	return buff.String(), err
}
