package kanister

import (
	"bytes"
	"context"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// Phase is an atomic unit of execution.
type Phase struct {
	name string
	args []string
	f    Func
}

// Name returns the name of this phase.
func (p *Phase) Name() string {
	return p.name
}

// Exec renders the argument templates in this Phase's Func and executes with
// those arguments.
func (p *Phase) Exec(ctx context.Context) error {
	return p.f.Exec(ctx, p.args...)
}

// GetPhases renders the returns a list of Phases with pre-rendered arguments.
func GetPhases(bp crv1alpha1.Blueprint, action string, tp TemplateParams) ([]*Phase, error) {
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
		args, err := renderArgs(p.Args, tp)
		if err != nil {
			return nil, err
		}
		phases = append(phases, &Phase{
			name: p.Name,
			args: args,
			f:    funcs[p.Func],
		})
	}
	return phases, nil
}

func renderArgs(args []string, tp TemplateParams) ([]string, error) {
	ras := make([]string, 0, len(args))
	for _, a := range args {
		ra, err := renderString(a, tp)
		if err != nil {
			return nil, err
		}
		ras = append(ras, ra)
	}
	return ras, nil
}

func RenderArtifacts(arts map[string]crv1alpha1.Artifact, tp TemplateParams) (map[string]crv1alpha1.Artifact, error) {
	rarts := make(map[string]crv1alpha1.Artifact, len(arts))
	for name, a := range arts {
		ra := crv1alpha1.Artifact{}
		for k, v := range a {
			rv, err := renderString(v, tp)
			if err != nil {
				return nil, err
			}
			ra[k] = rv
		}
		rarts[name] = ra
	}
	return rarts, nil
}

func renderString(arg string, tp TemplateParams) (string, error) {
	t, err := template.New("config").Funcs(sprig.TxtFuncMap()).Parse(arg)
	if err != nil {
		return "", errors.WithStack(err)
	}
	buf := bytes.NewBuffer(nil)
	if err = t.Execute(buf, tp); err != nil {
		return "", errors.WithStack(err)
	}
	return buf.String(), nil
}
