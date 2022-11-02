/*
Copyright 2022.

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

package repositoryserver

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	crkanisteriov1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// RepositoryServerReconciler reconciles a RepositoryServer object
type RepositoryServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cr.kanister.io,resources=repositoryservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cr.kanister.io,resources=repositoryservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cr.kanister.io,resources=repositoryservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RepositoryServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *RepositoryServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here

	cnf, err := ctrl.GetConfig()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to get k8s config")
	}
	kubeCli, err := kubernetes.NewForConfig(cnf)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to get a k8s client")
	}

	logger.Info("Fetch the RepositoryServer CR. If found start the repositoryServer else end reconcile loop....")
	repositoryServer := &crkanisteriov1alpha1.RepositoryServer{}
	if err = r.Get(ctx, req.NamespacedName, repositoryServer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Starting the RepositoryServer and it's reconciliation....")
	repoServerHandler := newRepositoryServerHandler(ctx, req, logger, r, kubeCli, repositoryServer)
	if err := repoServerHandler.Run(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepositoryServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crkanisteriov1alpha1.RepositoryServer{}).
		Complete(r)
}

func newRepositoryServerHandler(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	reconciler *RepositoryServerReconciler,
	kubeCli kubernetes.Interface,
	repositoryServer *crkanisteriov1alpha1.RepositoryServer) RepoServerHandler {
	repoServerCROwnerRef := metav1.OwnerReference{
		APIVersion: repositoryServer.APIVersion,
		Kind:       repositoryServer.Kind,
		Name:       repositoryServer.Name,
		UID:        repositoryServer.UID,
	}
	return RepoServerHandler{
		Ctx:              ctx,
		Req:              req,
		Logger:           logger,
		Reconciler:       reconciler,
		KubeCli:          kubeCli,
		RepositoryServer: repositoryServer,
		OwnerReference:   repoServerCROwnerRef,
	}
}
