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
	"github.com/int128/argocd-commenter/controllers/predicates"
	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationHealthCheckRunReconciler reconciles a ApplicationHealthCheckRun object
type ApplicationHealthCheckRunReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list

func (r *ApplicationHealthCheckRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		logger.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
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
	if err := r.Notification.CheckRun(ctx, e); err != nil {
		logger.Error(err, "unable to send a check run notification")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationHealthCheckRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(predicates.ApplicationUpdate(applicationHealthCheckRunComparer{})).
		Complete(r)
}

type applicationHealthCheckRunComparer struct{}

func (applicationHealthCheckRunComparer) Compare(applicationOld, applicationNew argocdv1alpha1.Application) bool {
	return applicationOld.Status.Health.Status != applicationNew.Status.Health.Status
}
