package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) NotifyHealth(ctx context.Context, a argocdv1alpha1.Application) error {
	logger := logr.FromContextOrDiscard(ctx)

	repository := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}

	logger.Info("creating a comment")
	comment := github.Comment{
		Repository: *repository,
		CommitSHA:  a.Status.Sync.Revision,
		Body:       healthCommentFor(a),
	}
	if err := c.ghc.AddComment(ctx, comment); err != nil {
		return fmt.Errorf("unable to add a comment: %w", err)
	}

	deploymentURL := a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	logger.Info("creating a deployment status", "deployment", deploymentURL)
	deploymentStatus := healthDeploymentStatusFor(a)
	deploymentStatus.Deployment = *deployment
	deploymentStatus.Description = string(a.Status.Health.Status)
	if err := c.ghc.CreateDeploymentStatus(ctx, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func healthCommentFor(a argocdv1alpha1.Application) string {
	if a.Status.Health.Status == health.HealthStatusHealthy {
		return fmt.Sprintf(":white_check_mark: %s: %s",
			a.Status.Health.Status,
			a.Name)
	}
	return fmt.Sprintf(":warning: %s: %s",
		a.Status.Health.Status,
		a.Name)
}

func healthDeploymentStatusFor(a argocdv1alpha1.Application) github.DeploymentStatus {
	if a.Status.Health.Status == health.HealthStatusHealthy {
		return github.DeploymentStatus{State: "success"}
	}
	return github.DeploymentStatus{State: "failure"}
}
