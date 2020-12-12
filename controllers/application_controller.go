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
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/int128/argocd-commenter/pkg/github"
)

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("application", req.NamespacedName)

	var application argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		log.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	repository, err := github.ParseRepositoryURL(application.Spec.Source.RepoURL)
	if err != nil {
		log.Error(err, "Skipped the Application for non-GitHub repository")
		return ctrl.Result{}, nil
	}
	commitComment := commitCommentFor(*repository, application)
	log.Info("Creating a commit comment", "body", commitComment.Body)
	if err := github.CreateCommitComment(ctx, commitComment); err != nil {
		if github.IsRetryableError(err) {
			return ctrl.Result{}, err
		}
		log.Error(err, "Ignored the permanent error")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func commitCommentFor(repository github.Repository, application argocdv1alpha1.Application) github.CommitComment {
	var operationMessage string
	if application.Status.OperationState != nil {
		operationMessage = application.Status.OperationState.Message
	}
	body := fmt.Sprintf(`name: %s/%s
sync-status: %s
health-status: %s (%s)
operation-message: %s`,
		application.Namespace,
		application.Name,
		application.Status.Sync.Status,
		application.Status.Health.Status,
		application.Status.Health.Message,
		operationMessage,
	)
	return github.CommitComment{
		Repository: repository,
		CommitSHA:  application.Status.Sync.Revision,
		Body:       body,
	}
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
	if applicationOld.Status.Sync != applicationNew.Status.Sync {
		return true
	}
	if applicationOld.Status.Health != applicationNew.Status.Health {
		return true
	}
	return false
}

func (p applicationStatusUpdatePredicate) Generic(event.GenericEvent) bool {
	return false
}
