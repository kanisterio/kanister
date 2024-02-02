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

package test

import "strings"

var redactedFlags = []string{
	"--password",
	"--user-password",
	"--server-password",
	"--server-control-password",
	"--server-cert-fingerprint",
}

// RedactCLI redacts sensitive information from the CLI command for tests.
func RedactCLI(cli []string) string {
	redactedCLI := make([]string, len(cli))
	for i, arg := range cli {
		redactField := ""
		for _, rf := range redactedFlags {
			if strings.HasPrefix(arg, rf) {
				redactField = rf
				break
			}
		}
		if len(redactField) > 0 {
			redactedCLI[i] = redactField + "=<****>"
		} else {
			redactedCLI[i] = arg
		}
	}
	return strings.Join(redactedCLI, " ")
}
