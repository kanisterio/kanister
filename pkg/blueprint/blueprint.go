package blueprint

import (
	"fmt"

	"github.com/gobuffalo/packr/v2"
	"gopkg.in/yaml.v2"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

var Examples *packr.Box

func init() {
	// Embed static files in examples/stable to binary
	Examples = packr.New("Kanister examples", "../../examples")
}

// GetStableExampleBP return Blueprint placed at examples/stable/{app}/{app}-blueprint.yaml
func GetStableExampleBP(app string) (*crv1alpha1.Blueprint, error) {
	path := fmt.Sprintf("stable/%s/%s-blueprint.yaml", app, app)
	bpRaw, err := Examples.Find(path)
	if err != nil {
		return nil, err
	}

	var bp crv1alpha1.Blueprint
	err = yaml.Unmarshal(bpRaw, &bp)
	return &bp, err
}
