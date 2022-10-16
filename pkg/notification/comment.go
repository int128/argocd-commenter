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

func (c client) CreateCommentOnPhaseChanged(ctx context.Context, e PhaseChangedEvent) error {
	logger := logr.FromContextOrDiscard(ctx)

	if e.Application.Status.OperationState == nil {
		return fmt.Errorf("status.operationState == nil")
	}
	if e.Application.Status.OperationState.Operation.Sync == nil {
		return fmt.Errorf("status.operationState.operation.sync == nil")
	}
	revision := e.Application.Status.OperationState.Operation.Sync.Revision

	repository := github.ParseRepositoryURL(e.Application.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}

	body := generateCommentOnPhaseChanged(e)
	if body == "" {
		logger.Info("nothing to comment", "event", e)
		return nil
	}

	pulls, err := c.ghc.ListPullRequests(ctx, *repository, revision)
	if err != nil {
		return fmt.Errorf("unable to list pull requests of revision %s: %w", revision, err)
	}

	relatedPullNumbers := filterPullRequestsRelatedToEvent(pulls, e.Application)
	logger.Info("creating a comment", "repository", repository, "pulls", relatedPullNumbers)
	if err := c.ghc.CreateComment(ctx, *repository, relatedPullNumbers, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func generateCommentOnPhaseChanged(e PhaseChangedEvent) string {
	revision := e.Application.Status.OperationState.Operation.Sync.Revision
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name)

	switch e.Application.Status.OperationState.Phase {
	case synccommon.OperationRunning:
		return fmt.Sprintf(":warning: Syncing [%s](%s) to %s", e.Application.Name, argocdApplicationURL, revision)
	case synccommon.OperationSucceeded:
		return fmt.Sprintf(":white_check_mark: Synced [%s](%s) to %s", e.Application.Name, argocdApplicationURL, revision)
	case synccommon.OperationFailed, synccommon.OperationError:
		return fmt.Sprintf("## :x: Sync %s: [%s](%s)\nError while syncing to %s:\n%s",
			e.Application.Status.OperationState.Phase,
			e.Application.Name,
			argocdApplicationURL,
			revision,
			generateSyncResultComment(e),
		)
	}
	return ""
}

func generateSyncResultComment(e PhaseChangedEvent) string {
	if e.Application.Status.OperationState.SyncResult == nil {
		return ""
	}
	var b strings.Builder
	for _, r := range e.Application.Status.OperationState.SyncResult.Resources {
		namespacedName := r.Namespace + "/" + r.Name
		switch r.Status {
		case synccommon.ResultCodeSyncFailed, synccommon.ResultCodePruneSkipped:
			b.WriteString(fmt.Sprintf("- %s `%s`: %s\n", r.Status, namespacedName, r.Message))
		}
	}
	return b.String()
}

func (c client) CreateCommentOnHealthChanged(ctx context.Context, e HealthChangedEvent) error {
	logger := logr.FromContextOrDiscard(ctx)

	if e.Application.Status.OperationState == nil {
		return fmt.Errorf("status.operationState == nil")
	}
	if e.Application.Status.OperationState.Operation.Sync == nil {
		return fmt.Errorf("status.operationState.operation.sync == nil")
	}
	revision := e.Application.Status.OperationState.Operation.Sync.Revision

	repository := github.ParseRepositoryURL(e.Application.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}

	body := generateCommentOnHealthChanged(e)
	if body == "" {
		logger.Info("nothing to comment", "event", e)
		return nil
	}

	pulls, err := c.ghc.ListPullRequests(ctx, *repository, revision)
	if err != nil {
		return fmt.Errorf("unable to list pull requests of revision %s: %w", revision, err)
	}

	relatedPullNumbers := filterPullRequestsRelatedToEvent(pulls, e.Application)
	logger.Info("creating a comment", "repository", repository, "pulls", relatedPullNumbers)
	if err := c.ghc.CreateComment(ctx, *repository, relatedPullNumbers, body); err != nil {
		return fmt.Errorf("unable to create a comment: %w", err)
	}
	return nil
}

func generateCommentOnHealthChanged(e HealthChangedEvent) string {
	revision := e.Application.Status.OperationState.Operation.Sync.Revision
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name)
	switch e.Application.Status.Health.Status {
	case health.HealthStatusHealthy:
		return fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
			":white_check_mark:",
			e.Application.Status.Health.Status,
			e.Application.Name,
			argocdApplicationURL,
			revision,
		)
	case health.HealthStatusDegraded:
		return fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
			":x:",
			e.Application.Status.Health.Status,
			e.Application.Name,
			argocdApplicationURL,
			revision,
		)
	}
	return ""
}
