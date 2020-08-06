// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconcile

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/validate"
)

// ActionSet attempts to reconcile the modifications made by `f` with the
// ActionSet stored in the API server.
func ActionSet(ctx context.Context, cli crclientv1alpha1.CrV1alpha1Interface, ns, name string, f func(*crv1alpha1.ActionSet) error) error {
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err := cli.ActionSets(ns).Get(ctx, name, metav1.GetOptions{})
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
		_, err = cli.ActionSets(as.GetNamespace()).Update(ctx, as, metav1.UpdateOptions{})
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
