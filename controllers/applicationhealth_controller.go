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
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const myFinalizerName = "argocdcommenter.int128.github.io/finalizer"

// ApplicationHealthReconciler reconciles an Application object
type ApplicationHealthReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=applicationhealths/finalizers,verbs=update

func (r *ApplicationHealthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var appHealth argocdcommenterv1.ApplicationHealth
	if err := r.Get(ctx, req.NamespacedName, &appHealth); err != nil {
		logger.Error(err, "unable to get the ApplicationHealth")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !appHealth.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&appHealth, myFinalizerName) {
			return ctrl.Result{}, nil
		}
		if err := r.notifyDeployment(ctx, req); err != nil {
			// Don't retry to avoid locking the finalizer
			logger.Error(err, "unable to notify a deployment status")
		}
		controllerutil.RemoveFinalizer(&appHealth, myFinalizerName)
		if err := r.Update(ctx, &appHealth); err != nil {
			logger.Error(err, "unable to update the ApplicationHealth")
			return ctrl.Result{}, err
		}
		logger.Info("patched the ApplicationHealth to remove the finalizer")
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&appHealth, myFinalizerName) {
		patch := client.MergeFrom(appHealth.DeepCopy())
		controllerutil.AddFinalizer(&appHealth, myFinalizerName)
		if err := r.Patch(ctx, &appHealth, patch); err != nil {
			logger.Error(err, "unable to patch the ApplicationHealth")
			return ctrl.Result{}, err
		}
		logger.Info("patched the ApplicationHealth to add the finalizer")
	}
	return ctrl.Result{}, nil
}

func (r *ApplicationHealthReconciler) notifyDeployment(ctx context.Context, req ctrl.Request) error {
	var app argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return fmt.Errorf("unable to get the Application: %w", err)
	}
	e := notification.Event{
		ApplicationIsDeleting: true,
		Application:           app,
	}
	if err := r.Notification.Deployment(ctx, e); err != nil {
		return fmt.Errorf("unable to send a deployment status: %w", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationHealthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdcommenterv1.ApplicationHealth{}).
		Complete(r)
}
