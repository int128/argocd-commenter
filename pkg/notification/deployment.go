package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	"github.com/int128/argocd-commenter/pkg/argocd"
	"github.com/int128/argocd-commenter/pkg/github"
)

type DeploymentStatus struct {
	GitHubDeployment       github.Deployment
	GitHubDeploymentStatus github.DeploymentStatus
}

func NewDeploymentStatusOnPhaseChanged(app argocdv1alpha1.Application, ghd argocdcommenterv1.GitHubDeployment, argocdURL string) *DeploymentStatus {
	deployment := github.ParseDeploymentURL(ghd.Spec.DeploymentURL)
	if deployment == nil {
		return nil
	}
	ds := generateDeploymentStatusOnPhaseChanged(app, argocdURL)
	if ds == nil {
		return nil
	}
	return &DeploymentStatus{
		GitHubDeployment:       *deployment,
		GitHubDeploymentStatus: *ds,
	}
}

func generateDeploymentStatusOnPhaseChanged(app argocdv1alpha1.Application, argocdURL string) *github.DeploymentStatus {
	phase := argocd.GetOperationPhase(app)
	if phase == "" {
		return nil
	}
	ds := github.DeploymentStatus{
		LogURL:      fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
		Description: trimDescription(fmt.Sprintf("%s:\n%s", phase, app.Status.OperationState.Message)),
	}
	if len(app.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = app.Status.Summary.ExternalURLs[0]
	}
	switch phase {
	case synccommon.OperationRunning:
		ds.State = "queued"
		return &ds
	case synccommon.OperationSucceeded:
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

func NewDeploymentStatusOnHealthChanged(app argocdv1alpha1.Application, ghd argocdcommenterv1.GitHubDeployment, argocdURL string) *DeploymentStatus {
	deployment := github.ParseDeploymentURL(ghd.Spec.DeploymentURL)
	if deployment == nil {
		return nil
	}
	ds := generateHealthDeploymentStatus(app, argocdURL)
	if ds == nil {
		return nil
	}
	return &DeploymentStatus{
		GitHubDeployment:       *deployment,
		GitHubDeploymentStatus: *ds,
	}
}

func generateHealthDeploymentStatus(app argocdv1alpha1.Application, argocdURL string) *github.DeploymentStatus {
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
	}
	if len(app.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = app.Status.Summary.ExternalURLs[0]
	}
	ds.Description = trimDescription(fmt.Sprintf("%s:\n%s",
		app.Status.Health.Status,
		app.Status.Health.Message,
	))
	switch app.Status.Health.Status {
	case health.HealthStatusHealthy:
		ds.State = "success"
		return &ds
	case health.HealthStatusDegraded:
		ds.State = "failure"
		return &ds
	}
	return nil
}

func NewDeploymentStatusOnDeletion(app argocdv1alpha1.Application, ghd argocdcommenterv1.GitHubDeployment, argocdURL string) *DeploymentStatus {
	deployment := github.ParseDeploymentURL(ghd.Spec.DeploymentURL)
	if deployment == nil {
		return nil
	}
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
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

func (c client) CheckIfDeploymentIsAlreadyHealthy(ctx context.Context, deploymentURL string) (bool, error) {
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return false, nil
	}
	latestDeploymentStatus, err := c.ghc.FindLatestDeploymentStatus(ctx, *deployment)
	if err != nil {
		return false, fmt.Errorf("unable to find the latest deployment status: %w", err)
	}
	if latestDeploymentStatus == nil {
		return false, nil
	}
	return latestDeploymentStatus.State == "success", nil
}
