package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/pkg/github"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (c client) NotifyHealth(ctx context.Context, a argocdv1alpha1.Application) error {
	logger := log.FromContext(ctx)

	repository, err := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if err != nil {
		return nil
	}

	logger.Info("creating a comment", "application", a.Name, "revision", a.Status.Sync.Revision)
	comment := github.Comment{
		Repository: *repository,
		CommitSHA:  a.Status.Sync.Revision,
		Body:       healthStatusCommentFor(a),
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
	deploymentStatus := github.DeploymentStatus{
		Deployment:  *deployment,
		Description: string(a.Status.Health.Status),
	}
	if a.Status.Health.Status == health.HealthStatusHealthy {
		deploymentStatus.State = "success"
	} else {
		deploymentStatus.State = "failure"
	}
	if err := c.ghc.CreateDeploymentStatus(ctx, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func healthStatusCommentFor(a argocdv1alpha1.Application) string {
	if a.Status.Health.Status == health.HealthStatusHealthy {
		return fmt.Sprintf(":white_check_mark: %s: %s",
			a.Status.Health.Status,
			a.Name)
	}
	return fmt.Sprintf(":warning: %s: %s",
		a.Status.Sync.Status,
		a.Name)
}
