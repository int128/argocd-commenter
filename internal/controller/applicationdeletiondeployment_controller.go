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

package controller

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/controller/eventfilter"
	"github.com/int128/argocd-commenter/internal/notification"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationDeletionDeploymentReconciler reconciles an Application object.
// It creates a deployment status when the Application is deleting.
type ApplicationDeletionDeploymentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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

	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}

	if err := r.Notification.CreateDeploymentStatusOnDeletion(ctx, app, argocdURL); err != nil {
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "CreateDeploymentStatusError",
			"unable to create a deployment status on deletion: %s", err)
	} else {
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "CreatedDeploymentStatus",
			"created a deployment status on deletion")
	}
	return ctrl.Result{}, nil
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

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationDeletionDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("application-deletion-deployment")
	return ctrl.NewControllerManagedBy(mgr).
		Named("applicationDeletionDeployment").
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(eventfilter.ApplicationChanged(filterApplicationDeletionForDeploymentStatus)).
		Complete(r)
}

func filterApplicationDeletionForDeploymentStatus(appOld, appNew argocdv1alpha1.Application) bool {
	if argocd.GetDeploymentURL(appNew) == "" {
		return false
	}

	// DeletionTimestamp has been set
	if appOld.DeletionTimestamp != appNew.DeletionTimestamp && !appNew.DeletionTimestamp.IsZero() {
		return true
	}

	// The health status has been changed to missing
	healthOld, healthNew := appOld.Status.Health.Status, appNew.Status.Health.Status
	if healthOld != healthNew && healthNew == health.HealthStatusMissing {
		return true
	}

	return false
}
