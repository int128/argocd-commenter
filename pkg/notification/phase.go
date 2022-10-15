package notification

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

type PhaseChangedEvent struct {
	Application argocdv1alpha1.Application
	ArgoCDURL   string
}

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

func (c client) CreateDeploymentStatusOnPhaseChanged(ctx context.Context, e PhaseChangedEvent) error {
	logger := logr.FromContextOrDiscard(ctx)

	deploymentURL := GetDeploymentURL(e.Application)
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return nil
	}

	ds := generateDeploymentStatusOnPhaseChanged(e)
	if ds == nil {
		logger.Info("nothing to create a deployment status", "event", e)
		return nil
	}

	logger.Info("creating a deployment status", "state", ds.State, "deployment", deploymentURL)
	if err := c.ghc.CreateDeploymentStatus(ctx, *deployment, *ds); err != nil {
		return fmt.Errorf("unable to create a deployment status: %w", err)
	}
	return nil
}

func generateDeploymentStatusOnPhaseChanged(e PhaseChangedEvent) *github.DeploymentStatus {
	ds := github.DeploymentStatus{
		LogURL: fmt.Sprintf("%s/applications/%s", e.ArgoCDURL, e.Application.Name),
	}
	if len(e.Application.Status.Summary.ExternalURLs) > 0 {
		ds.EnvironmentURL = e.Application.Status.Summary.ExternalURLs[0]
	}

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
