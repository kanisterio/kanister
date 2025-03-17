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
	// CallFuncName gives the name of the function
	CallFuncName          = "CallFunc"
	CallFuncNamespaceArg  = "namespace"
	CallFuncRefArg        = "ref"
	CallFuncActionNameArg = "action"
	CallFuncPhaseNameArg  = "phase"
	CallFuncArgsArg       = "args"
)

func init() {
	_ = kanister.Register(&callFunc{})
}

var _ kanister.Func = (*callFunc)(nil)

type callFunc struct {
	// FIXME: how to propagate progress??
	phase kanister.Phase
}

func (*callFunc) Name() string {
	return CallFuncName
}

func (cf *callFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, ref, actionName, phaseName string
	if err := Arg(args, CallFuncNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err := Arg(args, CallFuncRefArg, &ref); err != nil {
		return nil, err
	}
	if err := Arg(args, CallFuncActionNameArg, &actionName); err != nil {
		return nil, err
	}
	if err := Arg(args, CallFuncPhaseNameArg, &phaseName); err != nil {
		return nil, err
	}
	// FIXME: is this optional?
	var argsArg map[string]interface{}
	if err := Arg(args, CallFuncArgsArg, &argsArg); err != nil {
		return nil, err
	}

	renderedArgs, err := param.RenderArgs(argsArg, tp)
	if err != nil {
		// FIXME: wrap errors
		return nil, err
	}

	for k, v := range renderedArgs {
		tp.Args[k] = v.(string)
	}

	// FIXME: we can avoid returning blueprint if exec was not using it
	phase, bp, err := cf.getPhase(ctx, namespace, ref, actionName, phaseName, tp)
	if err != nil {
		// FIXME: wrap errors
		return nil, err
	}
	cf.phase = *phase

	return cf.phase.Exec(ctx, *bp, actionName, tp)
}

func (*callFunc) getPhase(ctx context.Context, namespace, ref, actionName, phaseName string, tp param.TemplateParams) (*kanister.Phase, *crv1alpha1.Blueprint, error) {
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

func (*callFunc) RequiredArgs() []string {
	return []string{
		CallFuncNamespaceArg,
		CallFuncRefArg,
		CallFuncActionNameArg,
		CallFuncPhaseNameArg,
		CallFuncArgsArg,
	}
}

func (*callFunc) Arguments() []string {
	return []string{
		CallFuncNamespaceArg,
		CallFuncRefArg,
		CallFuncActionNameArg,
		CallFuncPhaseNameArg,
		CallFuncArgsArg,
	}
}

func (cf *callFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(cf.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(cf.RequiredArgs(), args)
}

func (cf *callFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	return cf.phase.Progress()
}
