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
	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationSyncStatusReconciler reconciles a ApplicationSyncStatus object
type ApplicationSyncStatusReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ApplicationSyncStatus object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ApplicationSyncStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var application argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		logger.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	err := patchAnnotation(ctx, r.Client, application, func(annotations map[string]string) {
		annotationName := syncStatusLastRevisionAnnotationName(application.Status.Sync.Status)
		annotations[annotationName] = application.Status.Sync.Revision
	})
	if err != nil {
		logger.Error(err, "unable to patch annotations to the Application")
		return ctrl.Result{}, err
	}

	if err := r.Notification.NotifySync(ctx, application); err != nil {
		logger.Error(err, "unable to notify the sync status")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationSyncStatusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(&applicationSyncStatusChangePredicate{}).
		Complete(r)
}

type applicationSyncStatusChangePredicate struct{}

func (p applicationSyncStatusChangePredicate) Create(event.CreateEvent) bool {
	return false
}

func (p applicationSyncStatusChangePredicate) Delete(event.DeleteEvent) bool {
	return false
}

func (p applicationSyncStatusChangePredicate) Update(e event.UpdateEvent) bool {
	applicationOld, ok := e.ObjectOld.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	applicationNew, ok := e.ObjectNew.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	if applicationOld.Status.Sync.Status == applicationNew.Status.Sync.Status {
		return false
	}

	// notify only the following statuses
	switch applicationNew.Status.Sync.Status {
	case argocdv1alpha1.SyncStatusCodeSynced, argocdv1alpha1.SyncStatusCodeOutOfSync:
		annotationName := syncStatusLastRevisionAnnotationName(applicationNew.Status.Sync.Status)
		revision, ok := applicationNew.Annotations[annotationName]
		// first time or new revision
		if !ok || revision != applicationNew.Status.Sync.Revision {
			return true
		}
	}
	return false
}

func (p applicationSyncStatusChangePredicate) Generic(event.GenericEvent) bool {
	return false
}

func syncStatusLastRevisionAnnotationName(status argocdv1alpha1.SyncStatusCode) string {
	switch status {
	case argocdv1alpha1.SyncStatusCodeSynced:
		return "argocd-commenter.int128.github.io/last-revision-synced"
	case argocdv1alpha1.SyncStatusCodeOutOfSync:
		return "argocd-commenter.int128.github.io/last-revision-out-of-sync"
	}
	return ""
}
