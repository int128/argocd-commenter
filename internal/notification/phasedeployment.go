package notification

import (
	"context"
	"fmt"
	"strings"

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

	phase := argocd.GetSyncOperationPhase(app)
	if phase == "" {
		return nil
	}

	ds := DeploymentStatus{
		GitHubDeployment: *deployment,
		GitHubDeploymentStatus: github.DeploymentStatus{
			LogURL:      fmt.Sprintf("%s/applications/%s", argocdURL, app.Name),
			Description: trimDescription(generateDeploymentStatusDescriptionOnPhaseChanged(app)),
		},
	}
	if len(app.Status.Summary.ExternalURLs) > 0 {
		// Split the first element of ExternalURLs by `|`
		parts := strings.SplitN(app.Status.Summary.ExternalURLs[0], "|", 2)
		if len(parts) == 2 {
			// Assign the second part to EnvironmentURL as url
			// https://argo-cd.readthedocs.io/en/stable/user-guide/external-url/
			// this is hidden supported functionality https://github.com/argoproj/argo-cd/blob/f0b03071fc00fd81433d2c16861c193992d5a093/common/common.go#L186
			ds.GitHubDeploymentStatus.EnvironmentURL = parts[1]
		} else {
			ds.GitHubDeploymentStatus.EnvironmentURL = app.Status.Summary.ExternalURLs[0]
		}
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

func generateDeploymentStatusDescriptionOnPhaseChanged(app argocdv1alpha1.Application) string {
	phase := argocd.GetSyncOperationPhase(app)
	if phase == "" {
		return ""
	}
	syncResult := app.Status.OperationState.SyncResult
	if syncResult == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s:\n", phase))
	for _, r := range syncResult.Resources {
		namespacedName := r.Namespace + "/" + r.Name
		switch r.Status {
		case synccommon.ResultCodeSyncFailed:
			b.WriteString(fmt.Sprintf("%s: %s\n", namespacedName, r.Message))
		}
	}
	return b.String()
}
