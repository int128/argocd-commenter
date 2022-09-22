package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func GetDeploymentURL(a argocdv1alpha1.Application) string {
	return a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
}

func (c client) Deployment(ctx context.Context, e Event) error {
	logger := logr.FromContextOrDiscard(ctx)

	deploymentURL := GetDeploymentURL(e.Application)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	ds := generateDeploymentStatus(e)
	if ds == nil {
		logger.Info("nothing to create a deployment status", "event", e)
		return nil
	}

	logger.Info("creating a deployment status", "deployment", deploymentURL)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func generateDeploymentStatus(e Event) *github.DeploymentStatus {
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name),
	}
	if len(e.Application.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = e.Application.Status.Summary.ExternalURLs[0]
	}

	if e.PhaseIsChanged {
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
			ds.State = "in_progress"
			return &ds
		case synccommon.OperationFailed:
			ds.State = "failure"
			return &ds
		case synccommon.OperationError:
			ds.State = "failure"
			return &ds
		case synccommon.OperationTerminating:
			ds.State = "inactive"
			return &ds
		}
	}

	if e.HealthIsChanged {
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
