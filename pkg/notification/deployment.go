package notification

import (
	"context"
	"fmt"

	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) CreateDeploymentStatusOnPhaseChanged(ctx context.Context, e PhaseChangedEvent, deploymentURL string) error {
	logger := logr.FromContextOrDiscard(ctx)

	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	ds := generateDeploymentStatusOnPhaseChanged(e)
	if ds == nil {
		logger.Info("nothing to create a deployment status", "event", e)
		return nil
	}

	logger.Info("creating a deployment status", "state", ds.State, "deployment", deploymentURL)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func generateDeploymentStatusOnPhaseChanged(e PhaseChangedEvent) *github.DeploymentStatus {
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name),
	}
	if len(e.Application.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = e.Application.Status.Summary.ExternalURLs[0]
	}

	if e.Application.Status.OperationState == nil {
		return nil
	}
	ds.Description = trimDescription(fmt.Sprintf("%s:\n%s",
		e.Application.Status.OperationState.Phase,
		e.Application.Status.OperationState.Message,
	))
	switch e.Application.Status.OperationState.Phase {
	case synccommon.OperationRunning:
		ds.State = "queued"
		return &ds
	case synccommon.OperationSucceeded:
		// Some resources (such as CronJob) do not trigger Progressing status.
		// If healthy, complete the deployment as success.
		if e.Application.Status.Health.Status == health.HealthStatusHealthy {
			ds.State = "success"
			return &ds
		}
		ds.State = "in_progress"
		return &ds
	case synccommon.OperationFailed:
		ds.State = "failure"
		return &ds
	case synccommon.OperationError:
		ds.State = "failure"
		return &ds
	}
	return nil
}

func (c client) CreateDeploymentStatusOnHealthChanged(ctx context.Context, e HealthChangedEvent, deploymentURL string) error {
	logger := logr.FromContextOrDiscard(ctx)

	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	ds := generateHealthDeploymentStatus(e)
	if ds == nil {
		logger.Info("nothing to create a deployment status", "event", e)
		return nil
	}

	logger.Info("creating a deployment status", "state", ds.State, "deployment", deploymentURL)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func generateHealthDeploymentStatus(e HealthChangedEvent) *github.DeploymentStatus {
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name),
	}
	if len(e.Application.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = e.Application.Status.Summary.ExternalURLs[0]
	}
	ds.Description = trimDescription(fmt.Sprintf("%s:\n%s",
		e.Application.Status.Health.Status,
		e.Application.Status.Health.Message,
	))
	switch e.Application.Status.Health.Status {
	case health.HealthStatusHealthy:
		ds.State = "success"
		return &ds
	case health.HealthStatusDegraded:
		ds.State = "failure"
		return &ds
	case health.HealthStatusMissing:
		ds.State = "inactive"
		return &ds
	}
	return nil
}

func trimDescription(s string) string {
	// The maximum description length is 140 characters.
	// https://docs.github.com/en/rest/reference/deployments#create-a-deployment-status
	if len(s) < 140 {
		return s
	}
	return s[0:139]
}
