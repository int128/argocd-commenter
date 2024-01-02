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
	"slices"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/controller/predicates"
	"github.com/int128/argocd-commenter/internal/notification"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationPhaseDeploymentReconciler reconciles an Application object.
// It creates a deployment status when the sync operation phase is changed.
type ApplicationPhaseDeploymentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *ApplicationPhaseDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !app.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}
	phase := argocd.GetSyncOperationPhase(app)
	if phase == "" {
		return ctrl.Result{}, nil
	}

	deploymentURL := argocd.GetDeploymentURL(app)
	if deploymentURL == "" {
		return ctrl.Result{}, nil
	}
	deploymentIsAlreadyHealthy, err := r.Notification.CheckIfDeploymentIsAlreadyHealthy(ctx, deploymentURL)
	if notification.IsNotFoundError(err) {
		// Retry until the application is synced with a valid GitHub Deployment.
		// https://github.com/int128/argocd-commenter/issues/762
		lastOperationAt := argocd.GetLastOperationAt(app).Time
		if time.Since(lastOperationAt) < requeueTimeoutWhenDeploymentNotFound {
			r.Recorder.Eventf(&app, corev1.EventTypeNormal, "DeploymentNotFound",
				"deployment %s not found, retry after %s", deploymentURL, requeueIntervalWhenDeploymentNotFound)
			return ctrl.Result{RequeueAfter: requeueIntervalWhenDeploymentNotFound}, nil
		}
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "DeploymentNotFoundRetryTimeout",
			"deployment %s not found but retry timed out", deploymentURL)
		return ctrl.Result{}, nil
	}
	if deploymentIsAlreadyHealthy {
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "DeploymentAlreadyHealthy",
			"skip on phase %s because deployment %s is already healthy", phase, deploymentURL)
		return ctrl.Result{}, nil
	}

	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}

	if err := r.Notification.CreateDeploymentStatusOnPhaseChanged(ctx, app, argocdURL); err != nil {
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "CreateDeploymentStatusError",
			"unable to create a deployment status on phase %s: %s", phase, err)
	} else {
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "CreatedDeploymentStatus",
			"created a deployment status on phase %s", phase)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationPhaseDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("application-phase-deployment")
	return ctrl.NewControllerManagedBy(mgr).
		Named("applicationPhaseDeployment").
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(predicates.ApplicationUpdate(applicationPhaseDeploymentFilter{})).
		Complete(r)
}

type applicationPhaseDeploymentFilter struct{}

func (applicationPhaseDeploymentFilter) Compare(applicationOld, applicationNew argocdv1alpha1.Application) bool {
	phaseOld, phaseNew := argocd.GetSyncOperationPhase(applicationOld), argocd.GetSyncOperationPhase(applicationNew)
	if phaseNew == "" {
		return false
	}
	if phaseOld == phaseNew {
		return false
	}
	if argocd.GetDeploymentURL(applicationNew) == "" {
		return false
	}

	return slices.Contains(notification.SyncOperationPhasesForDeploymentStatus, phaseNew)
}
