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
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/controller/predicates"
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

var (
	// When the GitHub Deployment is not found, this action will retry by this interval
	// until the application is synced with a valid GitHub Deployment.
	// This should be reasonable to avoid the rate limit of GitHub API.
	requeueIntervalWhenDeploymentNotFound = 30 * time.Second

	// When the GitHub Deployment is not found, this action will retry by this timeout.
	// Argo CD refreshes an application every 3 minutes by default.
	// This should be reasonable to avoid the rate limit of GitHub API.
	requeueTimeoutWhenDeploymentNotFound = 10 * time.Minute
)

// ApplicationHealthReconciler reconciles an Application object
type ApplicationHealthReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Notification notification.Client
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

	r.notifyComment(ctx, app)
	r.notifyDeployment(ctx, app)
	return ctrl.Result{}, nil
}

func (r *ApplicationHealthReconciler) notifyComment(ctx context.Context, app argocdv1alpha1.Application) {
	logger := log.FromContext(ctx)

	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, app.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}
	comments := notification.NewCommentsOnOnHealthChanged(app, argocdURL)
	if len(comments) == 0 {
		logger.Info("no comment on this health event")
		return
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
}

func (r *ApplicationHealthReconciler) notifyDeployment(ctx context.Context, app argocdv1alpha1.Application) {
	logger := log.FromContext(ctx)

	argocdURL, err := argocd.GetExternalURL(ctx, r.Client, app.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}
	ds := notification.NewDeploymentStatusOnHealthChanged(app, argocdURL)
	if ds == nil {
		logger.Info("no deployment status on this health event")
		return
	}
	if err := r.Notification.CreateDeployment(ctx, *ds); err != nil {
		logger.Error(err, "unable to create a deployment status")
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "CreateDeploymentError",
			"unable to create a deployment status by %s: %s", app.Status.Health.Status, err)
	} else {
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "CreatedDeployment", "created a deployment status by %s", app.Status.Health.Status)
	}
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
