package notification

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) NotifyHealth(ctx context.Context, a argocdv1alpha1.Application, argoCDURL string) error {
	logger := logr.FromContextOrDiscard(ctx)
	if err := c.notifyHealthComment(ctx, logger, a); err != nil {
		logger.Error(err, "unable to notify a health comment")
	}
	if err := c.notifyHealthDeployment(ctx, logger, a, argoCDURL); err != nil {
		logger.Error(err, "unable to notify a health deployment")
	}
	return nil
}

func (c client) notifyHealthComment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application) error {
	repository := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}
	if a.Status.OperationState.Operation.Sync == nil {
		return fmt.Errorf("status.operationState.operation.sync == nil")
	}
	revision := a.Status.OperationState.Operation.Sync.Revision

	logger.Info("creating a comment", "repository", repository, "revision", revision)
	body := healthCommentFor(a)
	if err := c.ghc.CreateComment(ctx, *repository, revision, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func healthCommentFor(a argocdv1alpha1.Application) string {
	revision := a.Status.OperationState.Operation.Sync.Revision
	if a.Status.Health.Status == health.HealthStatusHealthy {
		return fmt.Sprintf("## :white_check_mark: %s is healthy\nSynced to %s", a.Name, revision)
	}
	return fmt.Sprintf("## :x: %s is %s\nSynced to %s",
		a.Name,
		strings.ToLower(string(a.Status.Health.Status)),
		revision)
}

func (c client) notifyHealthDeployment(ctx context.Context, logger logr.Logger, a argocdv1alpha1.Application, argoCDURL string) error {
	deploymentURL := a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	logger.Info("creating a deployment status", "deployment", deploymentURL)
	deploymentStatus := healthDeploymentStatusFor(a, argoCDURL)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, deploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func healthDeploymentStatusFor(a argocdv1alpha1.Application, argoCDURL string) github.DeploymentStatus {
	ds := github.DeploymentStatus{
		Description: string(a.Status.Health.Status),
		LogURL:      fmt.Sprintf("%s/applications/%s", argoCDURL, a.Name),
	}
	if len(a.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = a.Status.Summary.ExternalURLs[0]
	}
	switch a.Status.Health.Status {
	case health.HealthStatusHealthy:
		ds.State = "success"
	default:
		ds.State = "failure"
	}
	return ds
}
