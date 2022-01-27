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

package validate

import (
	"fmt"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	_ "github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
)

// Do takes a blueprint and validates if the function names in phases are correct
// and all the required arguments for the kanister functions are provided. This doesn't
// check anything with template params yet.
func Do(bp *crv1alpha1.Blueprint, funcVersion string) error {
	for name, action := range bp.Actions {
		// GetPhases also checks if the function names referred in the action are correct
		phases, err := kanister.GetPhases(*bp, name, funcVersion, param.TemplateParams{})
		if err != nil {
			utils.PrintStage(fmt.Sprintf("validation of action %s", name), utils.Fail)
			return fmt.Errorf("validation of action %s, error: %s", name, err.Error())
		}

		for i, phase := range phases {
			// validate function's mandatory arguments
			if err := phase.Validate(action.Phases[i].Args); err != nil {
				utils.PrintStage(fmt.Sprintf("validation of phase %s in action %s", phase.Name(), name), utils.Fail)
				return fmt.Errorf("validation of phase %s in action %s, error: %s", phase.Name(), name, err.Error())
			}
			utils.PrintStage(fmt.Sprintf("validation of phase %s in action %s", phase.Name(), name), utils.Pass)
		}
	}

	return nil
}
