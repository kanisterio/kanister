package function

import (
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

type BaseFunction struct {
}

func NewBaseFunction() *BaseFunction {
	return &BaseFunction{}
}

func (bf *BaseFunction) ExecProgress() (crv1alpha1.PhaseProgress, error) {
	return crv1alpha1.PhaseProgress{}, nil
}
