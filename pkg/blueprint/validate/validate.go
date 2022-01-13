package validate

import (
	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
)

func Do(bp *crv1alpha1.Blueprint) error {
	for _, action := range bp.Actions {
		phases, err := kanister.GetPhases(*bp, action.Name, kanister.DefaultVersion, param.TemplateParams{})
		if err != nil {
			return err
		}

		for _, phase := range phases {
			return phase.Validate()
		}
	}

	return nil
}
