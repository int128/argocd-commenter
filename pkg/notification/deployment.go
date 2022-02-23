package notification

import (
	"context"
	"fmt"

	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) Deployment(ctx context.Context, e Event) error {
	logger := logr.FromContextOrDiscard(ctx)

	deploymentURL := e.Application.Annotations["argocd-commenter.int128.github.io/deployment-url"]
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
		ds.Description = fmt.Sprintf("Application is %s,\n%s",
			e.Application.Status.OperationState.Phase,
			e.Application.Status.OperationState.Message,
		)
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
		}
	}

	if e.HealthIsChanged {
		ds.Description = fmt.Sprintf("Application is %s,\n%s",
			e.Application.Status.Health.Status,
			e.Application.Status.Health.Message,
		)
		switch e.Application.Status.Health.Status {
		case health.HealthStatusHealthy:
			ds.State = "success"
			return &ds
		case health.HealthStatusDegraded:
			ds.State = "failure"
			return &ds
		}
	}

	return nil
}
