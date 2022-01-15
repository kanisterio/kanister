package validate

import (
	"fmt"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	_ "github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
)

func Do(bp *crv1alpha1.Blueprint) error {
	for name, action := range bp.Actions {
		// GetPhases also checks if the function names referred in the action are correct
		phases, err := kanister.GetPhases(*bp, name, kanister.DefaultVersion, param.TemplateParams{})
		if err != nil {
			utils.PrintStage(fmt.Sprintf("Validation of action %s, error %s", name, err.Error()), utils.Fail)
			return err
		}

		for i, phase := range phases {
			// validate function's mandatory arguments
			if err := phase.Validate(action.Phases[i].Args); err != nil {
				utils.PrintStage(fmt.Sprintf("Validation of phase %s of action %s, error %s", phase.Name(), name, err.Error()), utils.Fail)
				return err
			}
			utils.PrintStage(fmt.Sprintf("Validation of phase %s of action %s", phase.Name(), name), utils.Pass)
		}

	}

	return nil
}
