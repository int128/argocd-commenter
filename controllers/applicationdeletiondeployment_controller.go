/*
Copyright 2021.

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

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/controllers/predicates"
	"github.com/int128/argocd-commenter/pkg/argocd"
	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationDeletionDeploymentReconciler reconciles an Application object on deletion
type ApplicationDeletionDeploymentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list

func (r *ApplicationDeletionDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	deploymentURL := argocd.GetDeploymentURL(app)
	if deploymentURL == "" {
		return ctrl.Result{}, nil
	}
	if !isApplicationDeleting(app) {
		return ctrl.Result{}, nil
	}
	logger = logger.WithValues(
		"health", app.Status.Health.Status,
		"deletionTimestamp", app.DeletionTimestamp,
	)
	ctx = log.IntoContext(ctx, logger)

	argoCDURL, err := argocd.GetExternalURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}
	ds := notification.NewDeploymentStatusOnDeletion(notification.DeletionEvent{
		Application: app,
		ArgoCDURL:   argoCDURL,
	})
	if ds == nil {
		logger.Info("no deployment status on this event")
		return ctrl.Result{}, nil
	}
	if err := r.Notification.CreateDeployment(ctx, *ds); err != nil {
		logger.Error(err, "unable to create a deployment status")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationDeletionDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("applicationDeletionDeployment").
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(predicates.ApplicationUpdate(applicationDeletionDeploymentFilter{})).
		Complete(r)
}

func isApplicationDeleting(app argocdv1alpha1.Application) bool {
	if !app.DeletionTimestamp.IsZero() {
		return true
	}
	if app.Status.Health.Status == health.HealthStatusMissing {
		return true
	}
	return false
}

type applicationDeletionDeploymentFilter struct{}

func (applicationDeletionDeploymentFilter) Compare(applicationOld, applicationNew argocdv1alpha1.Application) bool {
	if argocd.GetDeploymentURL(applicationNew) == "" {
		return false
	}

	// deletion timestamp has been set
	if applicationOld.DeletionTimestamp != applicationNew.DeletionTimestamp &&
		!applicationNew.DeletionTimestamp.IsZero() {
		return true
	}

	// health status has been changed to missing
	if applicationOld.Status.Health.Status != applicationNew.Status.Health.Status &&
		applicationNew.Status.Health.Status == health.HealthStatusMissing {
		return true
	}

	return false
}
