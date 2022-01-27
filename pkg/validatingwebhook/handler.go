package validatingwebhook

import (
	"context"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blueprint/validate"
)

type BlueprintValidator struct {
	decoder *admission.Decoder
}

func (b *BlueprintValidator) Handle(ctx context.Context, r admission.Request) admission.Response {
	bp := &crv1alpha1.Blueprint{}
	err := b.decoder.Decode(r, bp)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if err := validate.Do(bp, kanister.DefaultVersion); err != nil {
		return admission.Denied(fmt.Sprintf("Failed to validate blueprint, error %s\n", err.Error()))
	}

	return admission.Allowed("")
}

// BlueprintValidator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (b *BlueprintValidator) InjectDecoder(d *admission.Decoder) error {
	b.decoder = d
	return nil
}
