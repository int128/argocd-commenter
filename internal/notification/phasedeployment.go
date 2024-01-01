package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

var SyncOperationPhasesForDeploymentStatus = []synccommon.OperationPhase{
	synccommon.OperationRunning,
	synccommon.OperationSucceeded,
	synccommon.OperationFailed,
	synccommon.OperationError,
}

func (c client) CreateDeploymentStatusOnPhaseChanged(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error {
	ds := generateDeploymentStatusOnPhaseChanged(app, argocdURL)
	if ds == nil {
		return nil
	}
	if err := c.createDeploymentStatus(ctx, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func generateDeploymentStatusOnPhaseChanged(app argocdv1alpha1.Application, argocdURL string) *DeploymentStatus {
	deploymentURL := argocd.GetDeploymentURL(app)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	phase := argocd.GetOperationPhase(app)
	if phase == "" {
		return nil
	}

	ds := DeploymentStatus{
		GitHubDeployment: *deployment,
		GitHubDeploymentStatus: github.DeploymentStatus{
			LogURL:      fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
			Description: trimDescription(fmt.Sprintf("%s:\n%s", phase, app.Status.OperationState.Message)),
		},
	}
	if len(app.Status.Summary.ExternalURLs) > 0 {
		ds.GitHubDeploymentStatus.EnvironmentURL = app.Status.Summary.ExternalURLs[0]
	}
	switch phase {
	case synccommon.OperationRunning:
		ds.GitHubDeploymentStatus.State = "queued"
		return &ds
	case synccommon.OperationSucceeded:
		ds.GitHubDeploymentStatus.State = "in_progress"
		return &ds
	case synccommon.OperationFailed:
		ds.GitHubDeploymentStatus.State = "failure"
		return &ds
	case synccommon.OperationError:
		ds.GitHubDeploymentStatus.State = "failure"
		return &ds
	}
	return nil
}
