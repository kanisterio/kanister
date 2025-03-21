package function

import (
	"context"

	"github.com/kanisterio/errkit"
	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CallPhaseName gives the name of the function
	CallPhaseName          = "CallPhase"
	CallPhaseNamespaceArg  = "namespace"
	CallPhaseRefArg        = "ref"
	CallPhaseActionNameArg = "action"
	CallPhasePhaseNameArg  = "phase"
	CallPhaseOptionsArg    = "options"
	CallPhaseOverrideArg   = "overrideArgs"
)

func init() {
	_ = kanister.Register(&callPhase{})
}

var _ kanister.Func = (*callPhase)(nil)

type callPhase struct {
	phase kanister.Phase
}

func (*callPhase) Name() string {
	return CallPhaseName
}

func (cp *callPhase) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, ref, actionName, phaseName string
	if err := Arg(args, CallPhaseNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err := Arg(args, CallPhaseRefArg, &ref); err != nil {
		return nil, err
	}
	if err := Arg(args, CallPhaseActionNameArg, &actionName); err != nil {
		return nil, err
	}
	if err := Arg(args, CallPhasePhaseNameArg, &phaseName); err != nil {
		return nil, err
	}

	var optionsArg map[string]interface{}
	if err := OptArg(args, CallPhaseOptionsArg, &optionsArg, map[string]interface{}{}); err != nil {
		return nil, err
	}

	var overrideArgsArg map[string]interface{}
	if err := OptArg(args, CallPhaseOverrideArg, &overrideArgsArg, map[string]interface{}{}); err != nil {
		return nil, err
	}

	return cp.runPhase(ctx, namespace, ref, actionName, phaseName, optionsArg, overrideArgsArg, tp)

}

func (cp *callPhase) runPhase(ctx context.Context, namespace, ref, actionName, phaseName string, optionsArg, overrideArgsArg map[string]interface{}, tp param.TemplateParams) (map[string]interface{}, error) {
	renderedOptions, err := param.RenderArgs(optionsArg, tp)
	if err != nil {

		return nil, errkit.Wrap(err, "Error rendering options")
	}

	if tp.Options == nil {
		tp.Options = map[string]string{}
	}
	for k, v := range renderedOptions {
		val_str, ok := v.(string)
		if ok {
			tp.Options[k] = val_str
		} else {
			return nil, errkit.New("Option is not a string", "option", k, "value", v)
		}
	}

	renderedOverrides, err := param.RenderArgs(overrideArgsArg, tp)
	if err != nil {

		return nil, errkit.Wrap(err, "Error rendering arg overrides")
	}
	tp.Args = renderedOverrides

	// TODO: we can avoid returning blueprint if exec was not using it
	phase, bp, err := cp.getPhase(ctx, namespace, ref, actionName, phaseName, tp)
	if err != nil {
		return nil, errkit.Wrap(err, "Error getting phase to call", "namespace", namespace, "ref", ref, "actionName", actionName, "phaseName", phaseName)
	}
	cp.phase = *phase

	return cp.phase.Exec(ctx, *bp, actionName, tp)
}

func (*callPhase) getPhase(ctx context.Context, namespace, ref, actionName, phaseName string, tp param.TemplateParams) (*kanister.Phase, *crv1alpha1.Blueprint, error) {
	config, err := kube.LoadConfig()
	if err != nil {
		return nil, nil, errkit.Wrap(err, "Failed to load Kubernetes config")
	}
	client, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}

	bp, err := client.CrV1alpha1().Blueprints(namespace).Get(ctx, ref, metav1.GetOptions{})
	if err != nil {
		return nil, nil, errkit.Wrap(err, "Failed to get referenced blueprint", "namespace", namespace, "ref", ref)
	}

	// FIXME: pass version from the actionset
	p, err := kanister.GetPhase(*bp, actionName, kanister.DefaultVersion, phaseName, tp)
	if err != nil {
		return nil, nil, err
	}
	return p, bp, nil
}

func (*callPhase) RequiredArgs() []string {
	return []string{
		CallPhaseNamespaceArg,
		CallPhaseRefArg,
		CallPhaseActionNameArg,
		CallPhasePhaseNameArg,
	}
}

func (*callPhase) Arguments() []string {
	return []string{
		CallPhaseNamespaceArg,
		CallPhaseRefArg,
		CallPhaseActionNameArg,
		CallPhasePhaseNameArg,
		CallPhaseOptionsArg,
		CallPhaseOverrideArg,
	}
}

func (cf *callPhase) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(cf.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(cf.RequiredArgs(), args)
}

func (cf *callPhase) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	return cf.phase.Progress()
}
