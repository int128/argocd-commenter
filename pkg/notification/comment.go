package notification

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/argocd"
	"github.com/int128/argocd-commenter/pkg/github"
	"k8s.io/apimachinery/pkg/util/errors"
)

type Comment struct {
	GitHubRepository github.Repository
	Revision         string
	Body             string
}

func NewCommentOnOnPhaseChanged(e PhaseChangedEvent) *Comment {
	repository := github.ParseRepositoryURL(e.Application.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}
	revision := argocd.GetDeployedRevision(e.Application)
	if revision == "" {
		return nil
	}
	body := generateCommentOnPhaseChanged(e)
	if body == "" {
		return nil
	}
	return &Comment{
		GitHubRepository: *repository,
		Revision:         revision,
		Body:             body,
	}
}

func generateCommentOnPhaseChanged(e PhaseChangedEvent) string {
	if e.Application.Status.OperationState == nil {
		return ""
	}
	revision := argocd.GetDeployedRevision(e.Application)
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

func NewCommentOnOnHealthChanged(e HealthChangedEvent) *Comment {
	repository := github.ParseRepositoryURL(e.Application.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}
	revision := argocd.GetDeployedRevision(e.Application)
	if revision == "" {
		return nil
	}
	body := generateCommentOnHealthChanged(e)
	if body == "" {
		return nil
	}
	return &Comment{
		GitHubRepository: *repository,
		Revision:         revision,
		Body:             body,
	}
}

func generateCommentOnHealthChanged(e HealthChangedEvent) string {
	revision := argocd.GetDeployedRevision(e.Application)
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

func (c client) CreateComment(ctx context.Context, comment Comment, app argocdv1alpha1.Application) error {
	logger := logr.FromContextOrDiscard(ctx).WithValues(
		"revision", comment.Revision,
		"repository", comment.GitHubRepository,
	)
	pulls, err := c.ghc.ListPullRequests(ctx, comment.GitHubRepository, comment.Revision)
	if err != nil {
		return fmt.Errorf("unable to list pull requests of revision %s: %w", comment.Revision, err)
	}
	relatedPullNumbers := filterPullRequestsRelatedToEvent(pulls, app)
	if len(relatedPullNumbers) == 0 {
		logger.Info("no pull request related to the revision")
		return nil
	}
	if err := c.createComment(ctx, comment.GitHubRepository, relatedPullNumbers, comment.Body); err != nil {
		return fmt.Errorf("unable to create a phase comment on revision %s: %w", comment.Revision, err)
	}
	logger.Info("created a phase comment", "pulls", relatedPullNumbers)
	return nil
}

func (c client) createComment(ctx context.Context, repository github.Repository, pullNumbers []int, body string) error {
	var errs []error
	for _, pullNumber := range pullNumbers {
		if err := c.ghc.CreateComment(ctx, repository, pullNumber, body); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}
