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

	"github.com/int128/argocd-commenter/pkg/commenter"
	"github.com/int128/argocd-commenter/pkg/github"
)

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("application", req.NamespacedName)

	var application argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		log.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cmt := commenter.ApplicationOperationState{
		Log: log,
	}
	if err := cmt.Do(ctx, application); err != nil {
		if github.IsRetryableError(err) {
			return ctrl.Result{}, err
		}
		log.Error(err, "unable to add a comment")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdv1alpha1.Application{}).
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

	// change of status.sync.status
	if applicationOld.Status.Sync.Status != applicationNew.Status.Sync.Status {
		return true
	}

	// change of status.operationState.phase
	if applicationOld.Status.OperationState == nil {
		return false
	}
	if applicationNew.Status.OperationState == nil {
		return false
	}
	if applicationOld.Status.OperationState.Phase != applicationNew.Status.OperationState.Phase {
		return true
	}

	return false
}

func (p applicationStatusUpdatePredicate) Generic(event.GenericEvent) bool {
	return false
}
