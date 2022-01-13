package kanctl

import (
	"errors"

	"github.com/kanisterio/kanister/pkg/blueprint"
	"github.com/kanisterio/kanister/pkg/blueprint/validate"
)

func performBlueprintValidation(p *validateParams) error {
	if p.filename == "" {
		return errors.New("Name is not supported for blueprint resources, please specify blueprint manfiest using -f.")
	}

	// read blueprint from specified file
	bp, err := blueprint.ReadFromFile(p.filename)
	if err != nil {
		return err
	}

	return validate.Do(bp)
}
