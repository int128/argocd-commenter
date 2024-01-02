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

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/controller/eventfilter"
	"github.com/int128/argocd-commenter/internal/notification"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationHealthCommentReconciler reconciles a change of Application object.
// It creates a comment when the health status is changed.
type ApplicationHealthCommentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths/status,verbs=get;update;patch

func (r *ApplicationHealthCommentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !app.DeletionTimestamp.IsZero() {
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

	sourceRevisions := argocd.GetSourceRevisions(app)
	if len(sourceRevisions) == 0 {
		return ctrl.Result{}, nil
	}
	currentRevision := sourceRevisions[0].Revision
	if appHealth.Status.LastHealthyRevision == currentRevision {
		logger.Info("current revision is already healthy", "revision", currentRevision)
		return ctrl.Result{}, nil
	}

	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}

	if err := r.Notification.CreateCommentsOnHealthChanged(ctx, app, argocdURL); err != nil {
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "CreateCommentError",
			"unable to create a comment on health status %s: %s", app.Status.Health.Status, err)
	} else {
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "CreatedComment",
			"created a comment on health status %s", app.Status.Health.Status)
	}

	if app.Status.Health.Status != health.HealthStatusHealthy {
		return ctrl.Result{}, nil
	}
	patch := client.MergeFrom(appHealth.DeepCopy())
	appHealth.Status.LastHealthyRevision = currentRevision
	if err := r.Client.Status().Patch(ctx, &appHealth, patch); err != nil {
		logger.Error(err, "unable to patch lastHealthyRevision")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	r.Recorder.Eventf(&appHealth, corev1.EventTypeNormal, "UpdatedLastHealthyRevision",
		"patched lastHealthyRevision to %s", currentRevision)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationHealthCommentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("application-health-comment")
	return ctrl.NewControllerManagedBy(mgr).
		Named("applicationHealthComment").
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(eventfilter.ApplicationChanged(filterApplicationHealthStatusForComment)).
		Complete(r)
}

func filterApplicationHealthStatusForComment(appOld, appNew argocdv1alpha1.Application) bool {
	healthOld, healthNew := appOld.Status.Health.Status, appNew.Status.Health.Status
	if healthOld == healthNew {
		return false
	}

	return slices.Contains(notification.HealthStatusesForComment, healthNew)
}
