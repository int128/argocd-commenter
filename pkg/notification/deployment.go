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
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	logger := logr.FromContextOrDiscard(ctx).WithValues(
		"phase", e.Application.Status.OperationState.Phase,
		"deployment", deploymentURL,
	)

	ds := generateDeploymentStatusOnPhaseChanged(e)
	if ds == nil {
		logger.Info("no deployment status on this phase")
		return nil
	}

	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status of %s on phase changed: %w", ds.State, err)
	}
	logger.Info("created a deployment status", "state", ds.State)
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
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	logger := logr.FromContextOrDiscard(ctx).WithValues(
		"health", e.Application.Status.Health.Status,
		"deployment", deploymentURL,
	)

	ds := generateHealthDeploymentStatus(e)
	if ds == nil {
		logger.Info("no deployment status on this health status")
		return nil
	}

	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status of %s on health changed: %w", ds.State, err)
	}
	logger.Info("created a deployment status", "state", ds.State)
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
	}
	return nil
}

func (c client) CreateDeploymentStatusOnDeletion(ctx context.Context, e DeletionEvent, deploymentURL string) error {
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	logger := logr.FromContextOrDiscard(ctx).WithValues(
		"health", e.Application.Status.Health.Status,
		"deployment", deploymentURL,
	)

	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name),
		State:  "inactive",
	}
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, ds); err != nil {
		return fmt.Errorf("unable to create a deployment status of %s on application deletion: %w", ds.State, err)
	}
	logger.Info("created a deployment status", "state", ds.State)
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
