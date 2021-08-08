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
	"time"

	"github.com/int128/argocd-commenter/pkg/github"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
)

// GitHubCommentReconciler reconciles a GitHubComment object
type GitHubCommentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	GitHubClient github.Client
	Clock        clock.Clock
}

//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=githubcomments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=githubcomments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=argocdcommenter.int128.github.io,resources=githubcomments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GitHubComment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *GitHubCommentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var c argocdcommenterv1.GitHubComment
	if err := r.Get(ctx, req.NamespacedName, &c); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if r.Clock.Since(c.CreationTimestamp.Time) < 5*time.Second {
		after := 5*time.Second - r.Clock.Since(c.CreationTimestamp.Time)
		logger.Info("requeue for notification", "after", after)
		return ctrl.Result{Requeue: true, RequeueAfter: after}, nil
	}

	ghc := github.Comment{
		Repository: github.Repository{
			Owner: c.Spec.RepositoryOwner,
			Name:  c.Spec.RepositoryName,
		},
		CommitSHA: c.Spec.Revision,
	}
	for _, e := range c.Spec.Events {
		ghc.Body += e.Message + "\n"
	}

	if err := r.GitHubClient.AddComment(ctx, ghc); err != nil {
		logger.Error(err, "unable to add a comment to the revision", "comment", ghc)
		return ctrl.Result{}, nil
	}
	if err := r.Delete(ctx, &c); err != nil {
		logger.Error(err, "unable to delete the comment")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitHubCommentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argocdcommenterv1.GitHubComment{}).
		Complete(r)
}
