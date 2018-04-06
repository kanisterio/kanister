package reconcile

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/validate"
)

// ActionSet attempts to reconcile the modifications made by `f` with the
// ActionSet stored in the API server.
func ActionSet(ctx context.Context, cli crclientv1alpha1.CrV1alpha1Interface, ns, name string, f func(*crv1alpha1.ActionSet) error) error {
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err := cli.ActionSets(ns).Get(name, v1.GetOptions{})
		if err != nil {
			return false, errors.WithStack(err)
		}
		if err = validate.ActionSet(as); err != nil {
			return false, err
		}
		if err = f(as); err != nil {
			return false, err
		}
		if err = validate.ActionSet(as); err != nil {
			return false, err
		}
		_, err = cli.ActionSets(as.GetNamespace()).Update(as)
		// If we get a version conflict, we backoff and try again.
		if apierrors.IsConflict(err) {
			return false, nil
		}
		if err != nil {
			msg := fmt.Sprintf("Failed to update ActionSet %s", name)
			return false, errors.Wrap(err, msg)
		}
		return true, nil
	})
}
