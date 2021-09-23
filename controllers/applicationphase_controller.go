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
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	phaseStatusLastRevisionAnnotationName = "argocd-commenter.int128.github.io/last-revision-phase"
)

// ApplicationPhaseReconciler reconciles a ApplicationPhase object
type ApplicationPhaseReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list

func (r *ApplicationPhaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var application argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		logger := log.FromContext(ctx)
		logger.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger := log.FromContext(ctx,
		"phase", application.Status.OperationState.Phase,
		"revision", application.Status.Sync.Revision,
	)
	ctx = log.IntoContext(ctx, logger)

	err := patchAnnotation(ctx, r.Client, application, func(annotations map[string]string) {
		annotations[phaseStatusLastRevisionAnnotationName] = application.Status.Sync.Revision
	})
	if err != nil {
		logger.Error(err, "unable to patch annotations to the Application")
		return ctrl.Result{}, err
	}

	if err := r.Notification.NotifyPhase(ctx, application); err != nil {
		logger.Error(err, "unable to notify the phase status")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationPhaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(&applicationPhaseChangePredicate{}).
		Complete(r)
}

type applicationPhaseChangePredicate struct{}

func (p applicationPhaseChangePredicate) Update(e event.UpdateEvent) bool {
	applicationOld, ok := e.ObjectOld.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	applicationNew, ok := e.ObjectNew.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}

	if applicationOld.Status.OperationState == nil {
		return false
	}
	if applicationNew.Status.OperationState == nil {
		return false
	}
	if applicationOld.Status.OperationState.Phase == applicationNew.Status.OperationState.Phase {
		return false
	}

	// notify only the following phases
	switch applicationNew.Status.OperationState.Phase {
	case common.OperationFailed, common.OperationError:
		revision, ok := applicationNew.Annotations[phaseStatusLastRevisionAnnotationName]
		// first time or new revision
		if !ok || revision != applicationNew.Status.Sync.Revision {
			return true
		}
	}
	return false
}

func (p applicationPhaseChangePredicate) Create(event.CreateEvent) bool {
	return false
}

func (p applicationPhaseChangePredicate) Delete(event.DeleteEvent) bool {
	return false
}

func (p applicationPhaseChangePredicate) Generic(event.GenericEvent) bool {
	return false
}
