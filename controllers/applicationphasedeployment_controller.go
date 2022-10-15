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
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/controllers/predicates"
	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationPhaseDeploymentReconciler reconciles a ApplicationPhaseDeployment object
type ApplicationPhaseDeploymentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Notification notification.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;watch;list

func (r *ApplicationPhaseDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "controller", "ApplicationPhaseDeployment")
	ctx = log.IntoContext(ctx, logger)

	var application argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		logger.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if application.Status.OperationState == nil {
		logger.Info("skip notification due to application.status.operationState == nil")
		return ctrl.Result{}, nil
	}

	argoCDURL, err := findArgoCDURL(ctx, r.Client, req.Namespace)
	if err != nil {
		logger.Info("unable to determine Argo CD URL", "error", err)
	}
	e := notification.Event{
		PhaseIsChanged: true,
		Application:    application,
		ArgoCDURL:      argoCDURL,
	}
	if err := r.Notification.Deployment(ctx, e); err != nil {
		logger.Error(err, "unable to send a deployment status")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationPhaseDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdv1alpha1.Application{}).
		WithEventFilter(predicates.ApplicationUpdate(applicationPhaseDeploymentFilter{})).
		Complete(r)
}

type applicationPhaseDeploymentFilter struct{}

func (applicationPhaseDeploymentFilter) Compare(applicationOld, applicationNew argocdv1alpha1.Application) bool {
	if applicationNew.Status.OperationState == nil {
		return false
	}
	if applicationOld.Status.OperationState != nil &&
		applicationOld.Status.OperationState.Phase == applicationNew.Status.OperationState.Phase {
		return false
	}

	switch applicationNew.Status.OperationState.Phase {
	case synccommon.OperationRunning, synccommon.OperationSucceeded, synccommon.OperationFailed, synccommon.OperationError:
		return true
	}
	return false
}
