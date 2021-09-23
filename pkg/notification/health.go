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
	if err := c.notifyHealthComment(ctx, logger, a); err != nil {
		logger.Error(err, "unable to notify a health comment")
	}
	if err := c.notifyHealthDeployment(ctx, logger, a); err != nil {
		logger.Error(err, "unable to notify a health deployment")
	}
	return nil
}

func (c client) notifyHealthComment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application) error {
	repository := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}

	logger.Info("creating a comment", "repository", repository)
	body := healthCommentFor(a)
	if err := c.ghc.CreateComment(ctx, *repository, a.Status.Sync.Revision, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func healthCommentFor(a argocdv1alpha1.Application) string {
	if a.Status.Health.Status == health.HealthStatusHealthy {
		return fmt.Sprintf(":white_check_mark: %s: %s", a.Name, a.Status.Health.Status)
	}
	return fmt.Sprintf(":x: %s: %s", a.Name, a.Status.Health.Status)
}

func (c client) notifyHealthDeployment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application) error {
	deploymentURL := a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	logger.Info("creating a deployment status", "deployment", deploymentURL)
	deploymentStatus := healthDeploymentStatusFor(a)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func healthDeploymentStatusFor(a argocdv1alpha1.Application) github.DeploymentStatus {
	if a.Status.Health.Status == health.HealthStatusHealthy {
		return github.DeploymentStatus{State: "success", Description: string(a.Status.Health.Status)}
	}
	return github.DeploymentStatus{State: "failure", Description: string(a.Status.Health.Status)}
}
