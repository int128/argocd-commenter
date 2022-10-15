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
	"github.com/int128/argocd-commenter/pkg/notification"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	requeueIntervalWhenDeploymentNotFound = 30 * time.Second
)

// ApplicationHealthDeploymentReconciler reconciles an Application object
type ApplicationHealthDeploymentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths/status,verbs=get;update;patch

func (r *ApplicationHealthDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		logger.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	deploymentURL := notification.GetDeploymentURL(app)
	if deploymentURL == "" {
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
	if deploymentURL == appHealth.Status.LastHealthyDeploymentURL && app.Status.Health.Status != health.HealthStatusMissing {
		return ctrl.Result{}, nil
	}

	argoCDURL, err := findArgoCDURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}
	e := notification.Event{
		HealthIsChanged: true,
		Application:     app,
		ArgoCDURL:       argoCDURL,
	}
	if err := r.Notification.Deployment(ctx, e); err != nil {
		if notification.IsNotFoundError(err) {
			// Retry until the application is synced with a valid GitHub Deployment.
			// https://github.com/int128/argocd-commenter/issues/762
			logger.Info("requeue because deployment is not found", "after", requeueIntervalWhenDeploymentNotFound, "error", err)
			return ctrl.Result{RequeueAfter: requeueIntervalWhenDeploymentNotFound}, nil
		}
		logger.Error(err, "unable to send a deployment status")
	}

	if app.Status.Health.Status != health.HealthStatusHealthy {
		return ctrl.Result{}, nil
	}
	patch := client.MergeFrom(appHealth.DeepCopy())
	appHealth.Status.LastHealthyDeploymentURL = deploymentURL
	if err := r.Client.Status().Patch(ctx, &appHealth, patch); err != nil {
		logger.Error(err, "unable to patch the status of ApplicationHealth")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("patched the status of ApplicationHealth")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationHealthDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
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

	// Reconcile when the health status is changed to one:
	switch applicationNew.Status.Health.Status {
	case health.HealthStatusHealthy, health.HealthStatusDegraded, health.HealthStatusMissing:
		return true
	}
	return false
}
