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
	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	annotationNameOfLastDeploymentOfHealthy = "argocd-commenter.int128.github.io/last-deployment-healthy"
)

// ApplicationHealthDeploymentReconciler reconciles a ApplicationHealthDeployment object
type ApplicationHealthDeploymentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list

func (r *ApplicationHealthDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		logger.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if app.Status.Health.Status == health.HealthStatusHealthy {
		patch := client.MergeFrom(app.DeepCopy())
		if app.Annotations == nil {
			app.Annotations = make(map[string]string)
		}
		app.Annotations[annotationNameOfLastDeploymentOfHealthy] = notification.GetDeploymentURL(app)
		if err := r.Client.Patch(ctx, &app, patch); err != nil {
			logger.Error(err, "unable to patch the Application")
			return ctrl.Result{}, err
		}
		logger.Info("patched the Application", "annotations", app.Annotations)
	}

	argoCDURL, err := findArgoCDURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Error(err, "unable to determine Argo CD URL")
	}

	e := notification.Event{
		HealthIsChanged: true,
		Application:     app,
		ArgoCDURL:       argoCDURL,
	}
	if err := r.Notification.Deployment(ctx, e); err != nil {
		logger.Error(err, "unable to send a deployment status")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationHealthDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(predicates.ApplicationUpdate(applicationHealthDeploymentComparer{})).
		Complete(r)
}

type applicationHealthDeploymentComparer struct{}

func (applicationHealthDeploymentComparer) Compare(applicationOld, applicationNew argocdv1alpha1.Application) bool {
	if applicationOld.Status.Health.Status == applicationNew.Status.Health.Status {
		return false
	}

	currentDeploymentURL := notification.GetDeploymentURL(applicationNew)
	if currentDeploymentURL == "" {
		return false
	}

	switch applicationNew.Status.Health.Status {
	case health.HealthStatusHealthy, health.HealthStatusDegraded:
		lastNotifiedDeploymentURL := applicationNew.Annotations[annotationNameOfLastDeploymentOfHealthy]
		return currentDeploymentURL != lastNotifiedDeploymentURL
	case health.HealthStatusMissing:
		return true
	}
	return false
}
