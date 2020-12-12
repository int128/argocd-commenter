/*


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

	argocdv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
)

// MonitorReconciler reconciles a Monitor object
type MonitorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=monitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=monitors/status,verbs=get;update;patch

func (r *MonitorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("monitor", req.NamespacedName)

	var application argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		log.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var operationMessage string
	if application.Status.OperationState != nil {
		operationMessage = application.Status.OperationState.Message
	}

	log.Info("Application",
		"name", application.Name,
		"sync", application.Status.Sync.Status,
		"health", application.Status.Health.Status,
		"operation", operationMessage,
		"revision", application.Status.Sync.Revision,
	)

	return ctrl.Result{}, nil
}

func (r *MonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdcommenterv1.Monitor{}).
		Watches(&source.Kind{Type: &argocdv1alpha1.Application{}}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(&applicationStatusUpdatePredicate{}).
		Complete(r)
}

type applicationStatusUpdatePredicate struct{}

func (p applicationStatusUpdatePredicate) Create(event.CreateEvent) bool {
	return false
}

func (p applicationStatusUpdatePredicate) Delete(event.DeleteEvent) bool {
	return false
}

func (p applicationStatusUpdatePredicate) Update(e event.UpdateEvent) bool {
	applicationOld, ok := e.ObjectOld.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	applicationNew, ok := e.ObjectNew.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	if applicationOld.Status.Sync != applicationNew.Status.Sync {
		return true
	}
	return false
}

func (p applicationStatusUpdatePredicate) Generic(event.GenericEvent) bool {
	return false
}
