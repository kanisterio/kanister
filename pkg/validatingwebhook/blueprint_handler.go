package validatingwebhook

import (
	"context"
	"fmt"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blueprint/validate"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=validate/v1alpha1/blueprint,mutating=false,failurePolicy=fail,sideEffects=None,groups=cr.kanister.io,resources=blueprints,verbs=create,versions=v1alpha1,name=blueprint.cr.kanister.io

type BlueprintValidator struct{}

var _ webhook.CustomValidator = &BlueprintValidator{}

func (b *BlueprintValidator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := logf.FromContext(ctx)
	bp, ok := obj.(*crv1alpha1.Blueprint)
	if !ok {
		return nil, fmt.Errorf("expected a Blueprint but got a %T", obj)
	}
	log.Info("Validating blueprint")
	if err := validate.Do(bp, kanister.DefaultVersion); err != nil {
		return nil, fmt.Errorf("invalid blueprint, %s", err.Error())
	}

	return nil, nil
}

func (b *BlueprintValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return b.validate(ctx, obj)
}

func (b *BlueprintValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	return b.validate(ctx, newObj)
}

func (b BlueprintValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return b.validate(ctx, obj)
}
