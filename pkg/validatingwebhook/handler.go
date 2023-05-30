package validatingwebhook

import (
	"context"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blueprint/validate"
	"github.com/kanisterio/kanister/pkg/controllers/repositoryserver"
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
		return admission.Denied(fmt.Sprintf("Invalid blueprint, %s\n", err.Error()))
	}

	return admission.Allowed("")
}

// InjectDecoder injects the decoder.
func (b *BlueprintValidator) InjectDecoder(d *admission.Decoder) error {
	b.decoder = d
	return nil
}

type RepositoryServerValidator struct {
	decoder *admission.Decoder
}

func (r *RepositoryServerValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	rs := &crv1alpha1.RepositoryServer{}
	err := r.decoder.Decode(req, rs)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if err := repositoryserver.Do(rs); err != nil {
		return admission.Denied(fmt.Sprintf("Invalid repositoryserver, %s\n", err.Error()))
	}

	return admission.Allowed("")
}

// InjectDecoder injects the decoder.
func (r *RepositoryServerValidator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
