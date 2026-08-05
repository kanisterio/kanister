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

package main

import (
	"fmt"
	"os"

	"github.com/kanisterio/kanister/pkg/client/openapi"
)

// Generate the OpenAPI schema for applyconfiguration client generation.
func main() {
	if err := openapi.WriteSpec(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "failed to render OpenAPI spec: %v\n", err)
		os.Exit(1)
	}
}
