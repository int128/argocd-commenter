package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) Comment(ctx context.Context, e Event) error {
	logger := logr.FromContextOrDiscard(ctx)

	if e.Application.Status.OperationState == nil {
		return fmt.Errorf("status.operationState == nil")
	}
	if e.Application.Status.OperationState.Operation.Sync == nil {
		return fmt.Errorf("status.operationState.operation.sync == nil")
	}

	repository := github.ParseRepositoryURL(e.Application.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}

	body := generateComment(e)
	if body == "" {
		logger.Info("nothing to comment", "event", e)
		return nil
	}

	revision := e.Application.Status.OperationState.Operation.Sync.Revision
	logger.Info("creating a comment", "repository", repository, "revision", revision)
	if err := c.ghc.CreateComment(ctx, *repository, revision, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func generateComment(e Event) string {
	revision := e.Application.Status.OperationState.Operation.Sync.Revision
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name)

	if e.PhaseIsChanged {
		if e.Application.Status.OperationState.Phase == synccommon.OperationRunning {
			return fmt.Sprintf(":warning: Syncing [%s](%s) to %s", e.Application.Name, argocdApplicationURL, revision)
		}
		if e.Application.Status.OperationState.Phase == synccommon.OperationSucceeded {
			return fmt.Sprintf(":white_check_mark: Synced [%s](%s) to %s", e.Application.Name, argocdApplicationURL, revision)
		}

		var b strings.Builder
		b.WriteString(fmt.Sprintf("## :x: Sync %s: [%s](%s)\nError while syncing to %s:\n",
			e.Application.Status.OperationState.Phase,
			e.Application.Name,
			argocdApplicationURL,
			revision,
		))
		if e.Application.Status.OperationState.SyncResult != nil {
			for _, r := range e.Application.Status.OperationState.SyncResult.Resources {
				namespacedName := r.Namespace + "/" + r.Name
				switch r.Status {
				case synccommon.ResultCodeSyncFailed, synccommon.ResultCodePruneSkipped:
					b.WriteString(fmt.Sprintf("- %s `%s`: %s\n", r.Status, namespacedName, r.Message))
				}
			}
		}
		return b.String()
	}

	if e.HealthIsChanged {
		bodyIcon := ":x:"
		if e.Application.Status.Health.Status == health.HealthStatusHealthy {
			bodyIcon = ":white_check_mark:"
		}
		return fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
			bodyIcon,
			e.Application.Status.Health.Status,
			e.Application.Name,
			argocdApplicationURL,
			revision,
		)
	}

	return ""
}
