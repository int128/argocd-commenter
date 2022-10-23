package notification

import (
	"context"
	"fmt"

	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/argocd"
	"github.com/int128/argocd-commenter/pkg/github"
)

type DeploymentStatus struct {
	GitHubDeployment       github.Deployment
	GitHubDeploymentStatus github.DeploymentStatus
}

func NewDeploymentStatusOnPhaseChanged(e PhaseChangedEvent) *DeploymentStatus {
	deploymentURL := argocd.GetDeploymentURL(e.Application)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	ds := generateDeploymentStatusOnPhaseChanged(e)
	if ds == nil {
		return nil
	}
	return &DeploymentStatus{
		GitHubDeployment:       *deployment,
		GitHubDeploymentStatus: *ds,
	}
}

func generateDeploymentStatusOnPhaseChanged(e PhaseChangedEvent) *github.DeploymentStatus {
	if e.Application.Status.OperationState == nil {
		return nil
	}
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name),
	}
	if len(e.Application.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = e.Application.Status.Summary.ExternalURLs[0]
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

func NewDeploymentStatusOnHealthChanged(e HealthChangedEvent) *DeploymentStatus {
	deploymentURL := argocd.GetDeploymentURL(e.Application)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	ds := generateHealthDeploymentStatus(e)
	if ds == nil {
		return nil
	}
	return &DeploymentStatus{
		GitHubDeployment:       *deployment,
		GitHubDeploymentStatus: *ds,
	}
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

func NewDeploymentStatusOnDeletion(e DeletionEvent) *DeploymentStatus {
	deploymentURL := argocd.GetDeploymentURL(e.Application)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name),
		State:  "inactive",
	}
	return &DeploymentStatus{
		GitHubDeployment:       *deployment,
		GitHubDeploymentStatus: ds,
	}
}

func trimDescription(s string) string {
	// The maximum description length is 140 characters.
	// https://docs.github.com/en/rest/reference/deployments#create-a-deployment-status
	if len(s) < 140 {
		return s
	}
	return s[0:139]
}

func (c client) CreateDeployment(ctx context.Context, ds DeploymentStatus) error {
	logger := logr.FromContextOrDiscard(ctx).WithValues(
		"deployment", ds.GitHubDeployment,
		"state", ds.GitHubDeploymentStatus.State,
	)
	if err := c.ghc.CreateDeploymentStatus(ctx, ds.GitHubDeployment, ds.GitHubDeploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status of %s: %w", ds.GitHubDeploymentStatus.State, err)
	}
	logger.Info("created a deployment status")
	return nil
}
