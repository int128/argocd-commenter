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

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
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

// ApplicationPhaseCommentReconciler reconciles an Application object.
// It creates a comment when the sync operation phase is changed.
type ApplicationPhaseCommentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *ApplicationPhaseCommentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}

	if err := r.Notification.CreateCommentsOnPhaseChanged(ctx, app, argocdURL); err != nil {
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "CreateCommentError",
			"unable to create a comment on sync operation phase %s: %s", phase, err)
	} else {
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "CreatedComment",
			"created a comment on sync operation phase %s", phase)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationPhaseCommentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("application-phase-comment")
	return ctrl.NewControllerManagedBy(mgr).
		Named("applicationPhaseComment").
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(eventfilter.ApplicationChanged(filterApplicationSyncOperationPhaseForComment)).
		Complete(r)
}

func filterApplicationSyncOperationPhaseForComment(appOld, appNew argocdv1alpha1.Application) bool {
	phaseOld, phaseNew := argocd.GetSyncOperationPhase(appOld), argocd.GetSyncOperationPhase(appNew)
	if phaseNew == "" {
		return false
	}
	if phaseOld == phaseNew {
		return false
	}

	return slices.Contains(notification.SyncOperationPhasesForComment, phaseNew)
}
