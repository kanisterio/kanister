// Copyright 2023 The Kanister Authors.
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

package repositoryserver

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	crkanisteriov1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// RepositoryServerReconciler reconciles a RepositoryServer object
type RepositoryServerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=cr.kanister.io,resources=repositoryservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cr.kanister.io,resources=repositoryservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cr.kanister.io,resources=repositoryservers/finalizers,verbs=update
//+kubebuilder:rbac:groups=corev1,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networkingv1,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=corev1,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=corev1,resources=pods/status,verbs=get

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
	// logging the messages at debug level by default by
	// specifying the verbosity of logger to 1
	logger = logger.V(1)
	cnf, err := ctrl.GetConfig()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to get k8s config")
	}
	kubeCli, err := kubernetes.NewForConfig(cnf)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to get a k8s client")
	}

	repositoryServer := &crkanisteriov1alpha1.RepositoryServer{}
	if err = r.Get(ctx, req.NamespacedName, repositoryServer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	repositoryServer.Status.Progress = crkanisteriov1alpha1.Pending

	repoServerHandler := newRepositoryServerHandler(ctx, req, logger, r, kubeCli, repositoryServer)
	repoServerHandler.RepositoryServer = repositoryServer

	if err = r.Status().Update(ctx, repoServerHandler.RepositoryServer); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Create or update owned resources by Repository Server CR")
	if err := repoServerHandler.CreateOrUpdateOwnedResources(ctx); err != nil {
		logger.Info("Setting the CR status as 'Failed' since an error occurred in create/update event")
		if uerr := repoServerHandler.updateRepoServerProgress(ctx, crkanisteriov1alpha1.Failed); uerr != nil {
			return ctrl.Result{}, uerr
		}
		r.Recorder.Event(repoServerHandler.RepositoryServer, corev1.EventTypeWarning, "Failed", err.Error())
		return ctrl.Result{}, err
	}

	logger.Info("Connect to Kopia Repository")
	if err := repoServerHandler.connectToKopiaRepository(); err != nil {
		if uerr := repoServerHandler.updateRepoServerProgress(ctx, crkanisteriov1alpha1.Failed); uerr != nil {
			return ctrl.Result{}, uerr
		}
		return ctrl.Result{}, err
	}

	if result, err := repoServerHandler.setupKopiaRepositoryServer(ctx, logger); err != nil {
		return result, err
	}

	if uerr := repoServerHandler.updateRepoServerProgress(ctx, crkanisteriov1alpha1.Ready); uerr != nil {
		return ctrl.Result{}, uerr
	}

	return ctrl.Result{}, nil
}

func newRepositoryServerHandler(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	reconciler *RepositoryServerReconciler,
	kubeCli kubernetes.Interface,
	repositoryServer *crkanisteriov1alpha1.RepositoryServer) RepoServerHandler {
	return RepoServerHandler{
		Req:              req,
		Logger:           logger,
		Reconciler:       reconciler,
		KubeCli:          kubeCli,
		RepositoryServer: repositoryServer,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepositoryServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// The 'Owns' function allows the controller to set owner refs on
	// child resources and run the same reconcile loop for all events on child resources
	r.Recorder = mgr.GetEventRecorderFor("RepositoryServer")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crkanisteriov1alpha1.RepositoryServer{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
