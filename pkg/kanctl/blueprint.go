// Copyright 2022 The Kanister Authors.
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

package kanctl

import (
	"errors"

	"github.com/kanisterio/kanister/pkg/blueprint"
	"github.com/kanisterio/kanister/pkg/blueprint/validate"
)

func performBlueprintValidation(p *validateParams) error {
	if p.filename == "" {
		return errors.New("--name is not supported for blueprint resources, please specify blueprint manifest using -f.")
	}

	// read blueprint from specified file
	bp, err := blueprint.ReadFromFile(p.filename)
	if err != nil {
		return err
	}

	return validate.Do(bp, p.functionVersion)
}
