package notification

import (
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

func NewDeploymentStatusOnPhaseChanged(app argocdv1alpha1.Application, argocdURL string) *DeploymentStatus {
	deploymentURL := argocd.GetDeploymentURL(app)
	deployment := github.ParseDeploymentURL(deploymentURL)
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
