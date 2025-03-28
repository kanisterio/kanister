package function

import (
	"context"
	"strconv"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// CallMultiName gives the name of the function
	CallMultiName          = "CallMulti"
	CallMultiNamespaceArg  = "namespace"
	CallMultiRefArg        = "ref"
	CallMultiActionNameArg = "action"
	CallMultiPhaseNameArg  = "phase"
	CallMultiIterationsArg = "iterations"
)

func init() {
	_ = kanister.Register(&callMulti{})
}

var _ kanister.Func = (*callMulti)(nil)

type callMulti struct {
	callPhases []callPhase
}

func (*callMulti) Name() string {
	return CallMultiName
}

func (cm *callMulti) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, ref, actionName, phaseName string
	if err := Arg(args, CallMultiNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err := Arg(args, CallMultiRefArg, &ref); err != nil {
		return nil, err
	}
	if err := Arg(args, CallMultiActionNameArg, &actionName); err != nil {
		return nil, err
	}
	if err := Arg(args, CallMultiPhaseNameArg, &phaseName); err != nil {
		return nil, err
	}

	var iterations []map[string]map[string]interface{}
	if err := OptArg(args, CallMultiIterationsArg, &iterations, []map[string]map[string]interface{}{}); err != nil {
		return nil, err
	}

	// Channels for parallel run
	type Res struct {
		i   int
		res map[string]interface{}
	}
	resultCh := make(chan Res)
	errCh := make(chan error)

	totalResult := map[string]interface{}{}
	// FIXME: if we have no iterations?
	for i, iteration := range iterations {
		// FIXME: check options and overrides are empty? They could be, actually
		options := iteration["options"]
		overrideArgs := iteration["overrideArgs"]

		// Serial run:
		// phase := callPhase{}
		// cm.callPhases = append(cm.callPhases, phase)
		// result, err := phase.runPhase(ctx, namespace, ref, actionName, phaseName, options, overrideArgs, tp)
		// if err != nil {
		// 	return nil, err
		// }
		// totalResult[strconv.Itoa(i)] = result

		// Parallel run:
		// Is there a better way to run it other than just plain channels??
		go func() {
			phase := callPhase{}
			cm.callPhases = append(cm.callPhases, phase)
			result, err := phase.runPhase(ctx, namespace, ref, actionName, phaseName, options, overrideArgs, tp)
			if err != nil {
				errCh <- err
			} else {
				resultCh <- Res{i: i, res: result}
			}
		}()
	}
	for {
		select {
		case result := <-resultCh:
			totalResult[strconv.Itoa(result.i)] = result.res
		case err := <-errCh:
			return nil, err
		}
		if len(totalResult) == len(iterations) {
			return totalResult, nil
		}
	}
}

func (*callMulti) RequiredArgs() []string {
	return []string{
		CallMultiNamespaceArg,
		CallMultiRefArg,
		CallMultiActionNameArg,
		CallMultiPhaseNameArg,
	}
}

func (*callMulti) Arguments() []string {
	return []string{
		CallMultiNamespaceArg,
		CallMultiRefArg,
		CallMultiActionNameArg,
		CallMultiPhaseNameArg,
		CallMultiIterationsArg,
	}
}

func (cf *callMulti) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(cf.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(cf.RequiredArgs(), args)
}

func (cf *callMulti) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	// FIXME: progress of iteration??
	// return cf.phase.Progress()
	return crv1alpha1.PhaseProgress{
		ProgressPercent: progress.StartedPercent,
	}, nil
}
