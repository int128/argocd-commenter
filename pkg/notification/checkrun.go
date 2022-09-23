package notification

import (
	"context"
	"fmt"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
	"strings"
)

func (c client) CheckRun(ctx context.Context, e Event) error {
	logger := logr.FromContextOrDiscard(ctx)

	commitURL := e.Application.Annotations["argocd-commenter.int128.github.io/commit-url"]
	commit := github.ParseCommitURL(commitURL)
	if commit == nil {
		return nil
	}

	cr := generateCheckRun(e)
	if cr == nil {
		logger.Info("nothing to update the check run", "event", e)
		return nil
	}

	logger.Info("updating the check run", "checkRun", commitURL)
	if err := c.ghc.CreateCheckRun(ctx, *commit, *cr); err != nil {
		return fmt.Errorf("unable to create a check run: %w", err)
	}
	return nil
}

func generateCheckRun(e Event) *github.CheckRun {
	applicationURL := fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name)
	externalURLs := strings.Join(e.Application.Status.Summary.ExternalURLs, "\n")

	var checkRun github.CheckRun
	checkRun.Name = e.Application.Name
	checkRun.Summary = fmt.Sprintf(`
## Argo CD
%s

## External URL
%s
`, applicationURL, externalURLs)

	if e.PhaseIsChanged {
		if e.Application.Status.OperationState == nil {
			return nil
		}
		switch e.Application.Status.OperationState.Phase {
		case synccommon.OperationRunning:
			checkRun.Status = "in_progress"
			checkRun.Title = fmt.Sprintf("Syncing: %s", e.Application.Status.OperationState.Message)
			return &checkRun
		case synccommon.OperationSucceeded:
			checkRun.Status = "completed"
			checkRun.Conclusion = "success"
			checkRun.Title = fmt.Sprintf("Synced: %s", e.Application.Status.OperationState.Message)
			return &checkRun
		case synccommon.OperationFailed:
			checkRun.Status = "completed"
			checkRun.Conclusion = "failure"
			checkRun.Title = fmt.Sprintf("Sync Failed: %s", e.Application.Status.OperationState.Message)
			return &checkRun
		case synccommon.OperationError:
			checkRun.Status = "completed"
			checkRun.Conclusion = "failure"
			checkRun.Title = fmt.Sprintf("Sync Error: %s", e.Application.Status.OperationState.Message)
			return &checkRun
		}
	}

	if e.HealthIsChanged {
		switch e.Application.Status.Health.Status {
		case health.HealthStatusProgressing:
			checkRun.Status = "in_progress"
			checkRun.Title = fmt.Sprintf("Progressing: %s", e.Application.Status.Health.Message)
			return &checkRun
		case health.HealthStatusHealthy:
			checkRun.Status = "completed"
			checkRun.Conclusion = "success"
			checkRun.Title = fmt.Sprintf("Healthy: %s", e.Application.Status.Health.Message)
			return &checkRun
		case health.HealthStatusDegraded:
			checkRun.Status = "completed"
			checkRun.Conclusion = "failure"
			checkRun.Title = fmt.Sprintf("Degraded: %s", e.Application.Status.Health.Message)
			return &checkRun
		}
	}

	return nil
}
