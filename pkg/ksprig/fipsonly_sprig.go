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

package ksprig

import (
	"fmt"
	"html/template"

	"github.com/Masterminds/sprig"
)

// TxtFuncMap provides a FIPS compliant version of sprig.TxtFuncMap().
// Usage of a FIPS non-compatible function from the function map will result in an error.
func TxtFuncMap() template.FuncMap {
	return replaceNonCompliantFuncs(sprig.TxtFuncMap())
}

func replaceNonCompliantFuncs(m map[string]interface{}) map[string]interface{} {
	for name, fn := range fipsNonCompliantFuncs {
		if _, ok := m[name]; ok {
			m[name] = fn
		}
	}
	return m
}

// fipsNonCompliantFuncs is a map of sprig function name to its replacement function.
// Functions identified for Sprig v3.2.3.
var fipsNonCompliantFuncs = map[string]interface{}{
	"bcrypt": func(input string) (string, error) {
		return "", NewUnsupportedSprigFuncErr("bcrypt")
	},

	"derivePassword": func(counter uint32, password_type, password, user, site string) (string, error) {
		return "", NewUnsupportedSprigFuncErr("derivePassword")
	},

	"genPrivateKey": func(typ string) (string, error) {
		switch typ {
		case "rsa", "ecdsa", "ed25519":
			fn, ok := sprig.TxtFuncMap()["genPrivateKey"].(func(string) string)
			if !ok {
				return "", NewUnsupportedSprigFuncErr("genPrivateKey")
			}
			return fn(typ), nil
		}
		return "", NewUnsupportedSprigFuncErr(fmt.Sprintf("genPrivateKey for %s key", typ))
	},

	"htpasswd": func(username string, password string) (string, error) {
		return "", NewUnsupportedSprigFuncErr("htpasswd")
	},
}

// NewUnsupportedSprigFuncErr returns an UnsupportedSprigFuncErr.
func NewUnsupportedSprigFuncErr(function string) UnsupportedSprigFuncErr {
	return UnsupportedSprigFuncErr{function: function}
}

// UnsupportedSprigFuncErr indicates the usage of a FIPS non-compatible function.
type UnsupportedSprigFuncErr struct {
	function string
}

// Error returns an error string indicating the unsupported function.
func (err UnsupportedSprigFuncErr) Error() string {
	return fmt.Sprintf("sprig function '%s' is not allowed by kanister as it is not FIPS compatible", err.function)
}
