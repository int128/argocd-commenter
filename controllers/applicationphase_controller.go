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
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/pkg/github"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	lastRevisionPhase = "argocd-commenter.int128.github.io/last-revision-phase"
)

// ApplicationPhaseReconciler reconciles a ApplicationPhase object
type ApplicationPhaseReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	GitHubClient github.Client
}

//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;watch;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ApplicationPhase object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ApplicationPhaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var application argocdv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		logger.Error(err, "unable to get the Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	lastRevision, ok := application.Annotations[lastRevisionPhase]
	if ok {
		if lastRevision == application.Status.Sync.Revision {
			logger.Info("already added a comment", "revision", lastRevision)
			return ctrl.Result{}, nil
		}
	}
	err := patchAnnotation(ctx, r.Client, application, func(annotations map[string]string) {
		annotations[lastRevisionPhase] = application.Status.Sync.Revision
	})
	if err != nil {
		logger.Error(err, "unable to patch annotations to the Application")
		return ctrl.Result{}, err
	}

	repository, err := github.ParseRepositoryURL(application.Spec.Source.RepoURL)
	if err != nil {
		logger.Error(err, "skip non-GitHub URL", "url", application.Spec.Source.RepoURL)
		return ctrl.Result{}, nil
	}
	comment := github.Comment{
		Repository: *repository,
		CommitSHA:  application.Status.Sync.Revision,
		Body:       phaseCommentFor(application),
	}
	logger.Info("adding a comment", "phase", application.Status.OperationState.Phase, "comment", comment)
	if err := r.GitHubClient.AddComment(ctx, comment); err != nil {
		logger.Error(err, "unable to add a comment", "comment", comment)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func phaseCommentFor(a argocdv1alpha1.Application) string {
	var resources strings.Builder
	if a.Status.OperationState.SyncResult != nil {
		for _, r := range a.Status.OperationState.SyncResult.Resources {
			namespacedName := r.Namespace + "/" + r.Name
			switch r.Status {
			case common.ResultCodeSyncFailed, common.ResultCodePruneSkipped:
				_, _ = fmt.Fprintf(&resources, "- %s `%s`: %s\n", r.Status, namespacedName, r.Message)
			}
		}
	}

	return fmt.Sprintf("## :x: Sync %s: %s\nError while syncing to %s\n%s",
		a.Status.OperationState.Phase,
		a.Name,
		a.Status.Sync.Revision,
		resources.String(),
	)
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

	// notify only failed or error
	switch applicationNew.Status.OperationState.Phase {
	case common.OperationFailed, common.OperationError:
		return true
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
