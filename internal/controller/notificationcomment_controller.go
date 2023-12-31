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
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/notification"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NotificationCommentReconciler reconciles a Notification object
type NotificationCommentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=notifications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=notifications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=notifications/finalizers,verbs=update
//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list

func (r *NotificationCommentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var appNotification argocdcommenterv1.Notification
	if err := r.Client.Get(ctx, req.NamespacedName, &appNotification); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if appNotification.Status.CommentState == appNotification.Status.State {
		return ctrl.Result{}, nil
	}

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !app.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if appNotification.Status.CommentState == argocdcommenterv1.NotificationStateHealthy {
		return ctrl.Result{}, nil
	}

	switch appNotification.Status.State {
	case argocdcommenterv1.NotificationStateSyncing,
		argocdcommenterv1.NotificationStateSynced,
		argocdcommenterv1.NotificationStateSyncFailed:
		if err := r.handlePhaseChanged(ctx, app); err != nil {
			return ctrl.Result{}, err
		}

	case argocdcommenterv1.NotificationStateProgressing,
		argocdcommenterv1.NotificationStateHealthy,
		argocdcommenterv1.NotificationStateDegraded:
		if err := r.handleHealthChanged(ctx, app); err != nil {
			return ctrl.Result{}, err
		}
	}

	patch := client.MergeFrom(appNotification.DeepCopy())
	appNotification.Status.CommentState = appNotification.Status.State
	if err := r.Client.Status().Patch(ctx, &appNotification, patch); err != nil {
		logger.Error(err, "unable to patch the Notification")
		return ctrl.Result{}, err
	}
	logger.Info("patched the Notification", "commentState", appNotification.Status.CommentState)
	return ctrl.Result{}, nil
}

func (r *NotificationCommentReconciler) handlePhaseChanged(ctx context.Context, app argocdv1alpha1.Application) error {
	logger := log.FromContext(ctx)

	phase := argocd.GetOperationPhase(app)
	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, app.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}
	comments := notification.NewCommentsOnOnPhaseChanged(app, argocdURL)
	if len(comments) == 0 {
		logger.Info("no comment on this phase event", "phase", phase)
		return nil
	}
	for _, comment := range comments {
		if err := r.Notification.CreateComment(ctx, comment, app); err != nil {
			logger.Error(err, "unable to create a comment")
			r.Recorder.Eventf(&app, corev1.EventTypeWarning, "CreateCommentError",
				"unable to create a comment by %s: %s", phase, err)
		} else {
			r.Recorder.Eventf(&app, corev1.EventTypeNormal, "CreatedComment", "created a comment by %s", phase)
		}
	}
	return nil
}

func (r *NotificationCommentReconciler) handleHealthChanged(ctx context.Context, app argocdv1alpha1.Application) error {
	logger := log.FromContext(ctx)

	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, app.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}
	comments := notification.NewCommentsOnOnHealthChanged(app, argocdURL)
	if len(comments) == 0 {
		logger.Info("no comment on this health event")
		return nil
	}
	for _, comment := range comments {
		if err := r.Notification.CreateComment(ctx, comment, app); err != nil {
			logger.Error(err, "unable to create a comment")
			r.Recorder.Eventf(&app, corev1.EventTypeWarning, "CreateCommentError",
				"unable to create a comment by %s: %s", app.Status.Health.Status, err)
		} else {
			r.Recorder.Eventf(&app, corev1.EventTypeNormal, "CreatedComment", "created a comment by %s", app.Status.Health.Status)
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NotificationCommentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("notification-comment-controller")
	return ctrl.NewControllerManagedBy(mgr).
		Named("notificationComment").
		For(&argocdcommenterv1.Notification{}).
		Complete(r)
}
