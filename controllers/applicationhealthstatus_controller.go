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
	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	healthStatusLastRevisionAnnotationName = "argocd-commenter.int128.github.io/last-revision-healthy"
)

// ApplicationHealthStatusReconciler reconciles a ApplicationHealthStatus object
type ApplicationHealthStatusReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list

func (r *ApplicationHealthStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "namespacedName", req.NamespacedName)
	ctx = log.IntoContext(ctx, logger)

	var application argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		logger.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.WithValues(
		"health", application.Status.Health.Status,
		"revision", application.Status.Sync.Revision,
	)

	err := patchAnnotation(ctx, r.Client, application, func(annotations map[string]string) {
		annotations[healthStatusLastRevisionAnnotationName] = application.Status.Sync.Revision
	})
	if err != nil {
		logger.Error(err, "unable to patch annotations to the Application")
		return ctrl.Result{}, err
	}

	if err := r.Notification.NotifyHealth(ctx, application); err != nil {
		logger.Error(err, "unable to notify the health status")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationHealthStatusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(&applicationHealthStatusChangePredicate{}).
		Complete(r)
}

type applicationHealthStatusChangePredicate struct{}

func (p applicationHealthStatusChangePredicate) Create(event.CreateEvent) bool {
	return false
}

func (p applicationHealthStatusChangePredicate) Delete(event.DeleteEvent) bool {
	return false
}

func (p applicationHealthStatusChangePredicate) Update(e event.UpdateEvent) bool {
	applicationOld, ok := e.ObjectOld.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	applicationNew, ok := e.ObjectNew.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	if applicationOld.Status.Health.Status == applicationNew.Status.Health.Status {
		return false
	}

	// notify only the following statuses
	switch applicationNew.Status.Health.Status {
	case health.HealthStatusHealthy, health.HealthStatusDegraded:
		revision, ok := applicationNew.Annotations[healthStatusLastRevisionAnnotationName]
		// first time or new revision
		if !ok || revision != applicationNew.Status.Sync.Revision {
			return true
		}
	}
	return false
}

func (p applicationHealthStatusChangePredicate) Generic(event.GenericEvent) bool {
	return false
}
