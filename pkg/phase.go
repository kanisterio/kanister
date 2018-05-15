package kanister

import (
	"context"

	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
)

// Phase is an atomic unit of execution.
type Phase struct {
	name string
	args map[string]interface{}
	f    Func
}

// Name returns the name of this phase.
func (p *Phase) Name() string {
	return p.name
}

// Exec renders the argument templates in this Phase's Func and executes with
// those arguments.
func (p *Phase) Exec(ctx context.Context) error {
	return p.f.Exec(ctx, p.args)
}

// GetPhases renders the returns a list of Phases with pre-rendered arguments.
func GetPhases(bp crv1alpha1.Blueprint, action string, tp param.TemplateParams) ([]*Phase, error) {
	a, ok := bp.Actions[action]
	if !ok {
		return nil, errors.Errorf("Action {%s} not found in action map", action)
	}
	funcMu.RLock()
	defer funcMu.RUnlock()
	// We first check that all requested phases are registered.
	for _, p := range a.Phases {
		if _, ok := funcs[p.Func]; !ok {
			return nil, errors.Errorf("Requested function {%s} has not been registered", p.Func)
		}
	}
	phases := make([]*Phase, 0, len(a.Phases))
	for _, p := range a.Phases {
		args, err := param.RenderArgs(p.Args, tp)
		if err != nil {
			return nil, err
		}
		if err = checkRequiredArgs(funcs[p.Func].RequiredArgs(), args); err != nil {
			return nil, errors.Wrapf(err, "Reqired args missing for function %s", funcs[p.Func].Name())
		}
		phases = append(phases, &Phase{
			name: p.Name,
			args: args,
			f:    funcs[p.Func],
		})
	}
	return phases, nil
}

func checkRequiredArgs(reqArgs []string, args map[string]interface{}) error {
	for _, a := range reqArgs {
		if _, ok := args[a]; !ok {
			return errors.Errorf("Required arg missing: %s", a)
		}
	}
	return nil
}
