package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) NotifyHealth(ctx context.Context, a argocdv1alpha1.Application, argoCDURL string) error {
	logger := logr.FromContextOrDiscard(ctx)
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", argoCDURL, a.Name)
	if err := c.notifyHealthComment(ctx, logger, a, argocdApplicationURL); err != nil {
		logger.Error(err, "unable to notify a health comment")
	}
	if err := c.notifyHealthDeployment(ctx, logger, a, argocdApplicationURL); err != nil {
		logger.Error(err, "unable to notify a health deployment")
	}
	return nil
}

func (c client) notifyHealthComment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application, argocdApplicationURL string) error {
	repository := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}
	if a.Status.OperationState.Operation.Sync == nil {
		return fmt.Errorf("status.operationState.operation.sync == nil")
	}
	revision := a.Status.OperationState.Operation.Sync.Revision

	bodyIcon := ":x:"
	if a.Status.Health.Status == health.HealthStatusHealthy {
		bodyIcon = ":white_check_mark:"
	}
	body := fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
		bodyIcon,
		a.Status.Health.Status,
		a.Name,
		argocdApplicationURL,
		revision,
	)

	logger.Info("creating a comment", "repository", repository, "revision", revision)
	if err := c.ghc.CreateComment(ctx, *repository, revision, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func (c client) notifyHealthDeployment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application, argocdApplicationURL string) error {
	deploymentURL := a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	deploymentStatus := github.DeploymentStatus{
		Description: fmt.Sprintf("Argo CD status is %s", a.Status.Health.Status),
		LogURL:      argocdApplicationURL,
	}
	if len(a.Status.Summary.ExternalURLs) > 0 {
		deploymentStatus.EnvironmentURL = a.Status.Summary.ExternalURLs[0]
	}
	switch a.Status.Health.Status {
	case health.HealthStatusHealthy:
		deploymentStatus.State = "success"
	default:
		deploymentStatus.State = "failure"
	}

	logger.Info("creating a deployment status", "deployment", deploymentURL)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}
