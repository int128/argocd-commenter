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
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	"github.com/int128/argocd-commenter/controllers/predicates"
	"github.com/int128/argocd-commenter/pkg/argocd"
	"github.com/int128/argocd-commenter/pkg/notification"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// When the GitHub Deployment is not found, this action will retry by this interval
	// until the application is synced with a valid GitHub Deployment.
	// This should be reasonable to avoid the rate limit of GitHub API.
	requeueIntervalWhenDeploymentNotFound = 30 * time.Second

	// When the GitHub Deployment is not found, this action will retry by this timeout.
	// Argo CD refreshes an application every 3 minutes by default.
	// This should be reasonable to avoid the rate limit of GitHub API.
	requeueTimeoutWhenDeploymentNotFound = 10 * time.Minute
)

// ApplicationHealthDeploymentReconciler reconciles an Application object
type ApplicationHealthDeploymentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *ApplicationHealthDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !app.DeletionTimestamp.IsZero() {
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
		lastOperationAt := argocd.GetLastOperationAt(app)
		if time.Now().After(lastOperationAt.Add(requeueTimeoutWhenDeploymentNotFound)) {
			logger.Info("requeue timeout because last operation is too old", "lastOperationAt", lastOperationAt)
			return ctrl.Result{}, nil
		}
		logger.Info("requeue because deployment is not found", "after", requeueIntervalWhenDeploymentNotFound, "error", err)
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "DeploymentNotFound",
			"deployment %s is not found, requeue after %s", deploymentURL, requeueIntervalWhenDeploymentNotFound)
		return ctrl.Result{RequeueAfter: requeueIntervalWhenDeploymentNotFound}, nil
	}
	if deploymentIsAlreadyHealthy {
		logger.Info("skip notification because the deployment is already healthy", "deployment", deploymentURL)
		return ctrl.Result{}, nil
	}

	var appHealth argocdcommenterv1.ApplicationHealth
	if err := r.Client.Get(ctx, req.NamespacedName, &appHealth); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to get the ApplicationHealth")
			return ctrl.Result{}, err
		}
		appHealth.ObjectMeta = metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      req.Name,
		}
		if err := ctrl.SetControllerReference(&app, &appHealth, r.Scheme); err != nil {
			logger.Error(err, "unable to set the controller reference to the ApplicationHealth")
			return ctrl.Result{}, err
		}
		if err := r.Client.Create(ctx, &appHealth); err != nil {
			logger.Error(err, "unable to create an ApplicationHealth")
			return ctrl.Result{}, err
		}
		logger.Info("created an ApplicationHealth")
	}

	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}
	ds := notification.NewDeploymentStatusOnHealthChanged(app, argocdURL)
	if ds == nil {
		logger.Info("no deployment status on this health event")
		return ctrl.Result{}, nil
	}
	if err := r.Notification.CreateDeployment(ctx, *ds); err != nil {
		logger.Error(err, "unable to create a deployment status")
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "CreateDeploymentError",
			"unable to create a deployment status by %s: %s", app.Status.Health.Status, err)
	} else {
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "CreatedDeployment", "created a deployment status by %s", app.Status.Health.Status)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationHealthDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("application-health-deployment")
	return ctrl.NewControllerManagedBy(mgr).
		Named("applicationHealthDeployment").
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(predicates.ApplicationUpdate(applicationHealthDeploymentFilter{})).
		Complete(r)
}

type applicationHealthDeploymentFilter struct{}

func (applicationHealthDeploymentFilter) Compare(applicationOld, applicationNew argocdv1alpha1.Application) bool {
	if applicationOld.Status.Health.Status == applicationNew.Status.Health.Status {
		return false
	}
	if argocd.GetDeploymentURL(applicationNew) == "" {
		return false
	}

	// Reconcile when the health status is changed to one:
	switch applicationNew.Status.Health.Status {
	case health.HealthStatusHealthy, health.HealthStatusDegraded:
		return true
	}
	return false
}
