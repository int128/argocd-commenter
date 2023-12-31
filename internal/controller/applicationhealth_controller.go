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

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	"github.com/int128/argocd-commenter/internal/controller/predicates"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationHealthReconciler reconciles a ApplicationHealth object
type ApplicationHealthReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *ApplicationHealthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !app.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	var appNotification argocdcommenterv1.Notification
	if err := r.Client.Get(ctx, req.NamespacedName, &appNotification); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to get the Notification")
			return ctrl.Result{}, err
		}
		appNotification.ObjectMeta = metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      req.Name,
		}
		if err := ctrl.SetControllerReference(&app, &appNotification, r.Scheme); err != nil {
			logger.Error(err, "unable to set the controller reference to the Notification")
			return ctrl.Result{}, err
		}
		if err := r.Client.Create(ctx, &appNotification); err != nil {
			logger.Error(err, "unable to create an Notification")
			return ctrl.Result{}, err
		}
		logger.Info("created an Notification")
	}

	appHealth := app.Status.Health.Status
	switch {
	case appHealth == health.HealthStatusProgressing:
		patch := client.MergeFrom(appNotification.DeepCopy())
		appNotification.Status.State = argocdcommenterv1.NotificationStateProgressing
		if err := r.Client.Status().Patch(ctx, &appNotification, patch); err != nil {
			logger.Error(err, "unable to patch ApplicationHealth")
			return ctrl.Result{}, err
		}
		logger.Info("patched the Notification", "state", appNotification.Status.State)
		return ctrl.Result{}, nil

	case appHealth == health.HealthStatusHealthy:
		patch := client.MergeFrom(appNotification.DeepCopy())
		appNotification.Status.State = argocdcommenterv1.NotificationStateHealthy
		if err := r.Client.Status().Patch(ctx, &appNotification, patch); err != nil {
			logger.Error(err, "unable to patch ApplicationHealth")
			return ctrl.Result{}, err
		}
		logger.Info("patched the Notification", "state", appNotification.Status.State)
		return ctrl.Result{}, nil

	case appHealth == health.HealthStatusDegraded:
		patch := client.MergeFrom(appNotification.DeepCopy())
		appNotification.Status.State = argocdcommenterv1.NotificationStateDegraded
		if err := r.Client.Status().Patch(ctx, &appNotification, patch); err != nil {
			logger.Error(err, "unable to patch ApplicationHealth")
			return ctrl.Result{}, err
		}
		logger.Info("patched the Notification", "state", appNotification.Status.State)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationHealthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("application-health-controller")
	return ctrl.NewControllerManagedBy(mgr).
		Named("applicationHealth").
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(predicates.ApplicationUpdate(applicationHealthChangedFilter{})).
		Complete(r)
}

type applicationHealthChangedFilter struct{}

func (applicationHealthChangedFilter) Compare(applicationOld, applicationNew argocdv1alpha1.Application) bool {
	if applicationOld.Status.Health.Status == applicationNew.Status.Health.Status {
		return false
	}
	return true
}
