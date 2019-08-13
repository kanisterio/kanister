/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ktrl "github.com/kanisterio/kanister/pkg/controller"
	crv1beta1 "github.com/kanisterio/kanister/pkg/controllerv2/api/v1beta1"

	//importing gocheck to allow ci tests to pass.
	_ "gopkg.in/check.v1"
)

// ActionSetReconciler reconciles a ActionSet object
type ActionSetReconciler struct {
	client.Client
	Log logr.Logger
	//Kanister ktrl.Controller
}

// +kubebuilder:rbac:groups=cr.kanister.io,resources=actionsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cr.kanister.io,resources=actionsets/status,verbs=get;update;patch

func (r *ActionSetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("actionset", req.NamespacedName)

	// your logic here

	_ = ktrl.Controller{}

	instance := &crv1beta1.ActionSet{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	fmt.Printf("object found is: %#v", instance)

	return ctrl.Result{}, nil
}

func (r *ActionSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crv1beta1.ActionSet{}).
		Complete(r)
}
