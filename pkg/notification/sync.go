package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/pkg/github"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (c client) NotifySync(ctx context.Context, a argocdv1alpha1.Application) error {
	logger := log.FromContext(ctx)

	repository, err := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if err != nil {
		return nil
	}

	logger.Info("creating a comment", "application", a.Name, "revision", a.Status.Sync.Revision)
	comment := github.Comment{
		Repository: *repository,
		CommitSHA:  a.Status.Sync.Revision,
		Body:       syncStatusCommentFor(a),
	}
	if err := c.ghc.AddComment(ctx, comment); err != nil {
		return fmt.Errorf("unable to add a comment: %w", err)
	}

	deploymentURL := a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	logger.Info("creating a deployment status", "application", a.Name, "deployment", deploymentURL)
	deploymentStatus := github.DeploymentStatus{
		Deployment: *deployment,
	}
	if a.Status.Sync.Status == argocdv1alpha1.SyncStatusCodeOutOfSync {
		deploymentStatus.State = "pending"
		deploymentStatus.Description = "Out of sync"
	}
	if a.Status.Sync.Status == argocdv1alpha1.SyncStatusCodeSynced {
		deploymentStatus.State = "in_progress"
		deploymentStatus.Description = "Synced"
	}
	if err := c.ghc.CreateDeploymentStatus(ctx, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func syncStatusCommentFor(a argocdv1alpha1.Application) string {
	if a.Status.Sync.Status == argocdv1alpha1.SyncStatusCodeSynced {
		return fmt.Sprintf("## :white_check_mark: %s: %s\nSynced to %s",
			a.Status.Sync.Status,
			a.Name,
			a.Status.Sync.Revision)
	}
	return fmt.Sprintf("## :warning: %s: %s\nSyncing to %s",
		a.Status.Sync.Status,
		a.Name,
		a.Status.Sync.Revision)
}
