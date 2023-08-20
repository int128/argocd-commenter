package notification

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
	"k8s.io/apimachinery/pkg/util/errors"
)

type Comment struct {
	GitHubRepository github.Repository
	Revision         string
	Body             string
}

func NewCommentsOnOnPhaseChanged(app argocdv1alpha1.Application, argocdURL string) []Comment {
	sourceRevisions := argocd.GetSourceRevisions(app)
	var comments []Comment
	for _, sourceRevision := range sourceRevisions {
		comment := generateCommentOnPhaseChanged(app, argocdURL, sourceRevision)
		if comment == nil {
			continue
		}
		comments = append(comments, *comment)
	}
	return comments
}

func generateCommentOnPhaseChanged(app argocdv1alpha1.Application, argocdURL string, sourceRevision argocd.SourceRevision) *Comment {
	repository := github.ParseRepositoryURL(sourceRevision.Source.RepoURL)
	if repository == nil {
		return nil
	}
	body := generateCommentBodyOnPhaseChanged(app, argocdURL, sourceRevision)
	if body == "" {
		return nil
	}
	return &Comment{
		GitHubRepository: *repository,
		Revision:         sourceRevision.Revision,
		Body:             body,
	}
}

func generateCommentBodyOnPhaseChanged(app argocdv1alpha1.Application, argocdURL string, sourceRevision argocd.SourceRevision) string {
	if app.Status.OperationState == nil {
		return ""
	}
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", argocdURL, app.Name)
	phase := app.Status.OperationState.Phase
	switch phase {
	case synccommon.OperationRunning:
		return fmt.Sprintf(":warning: Syncing [%s](%s) to %s", app.Name, argocdApplicationURL, sourceRevision.Revision)
	case synccommon.OperationSucceeded:
		return fmt.Sprintf(":white_check_mark: Synced [%s](%s) to %s", app.Name, argocdApplicationURL, sourceRevision.Revision)
	case synccommon.OperationFailed, synccommon.OperationError:
		return fmt.Sprintf("## :x: Sync %s: [%s](%s)\nError while syncing to %s:\n%s",
			phase,
			app.Name,
			argocdApplicationURL,
			sourceRevision.Revision,
			generateSyncResultComment(app.Status.OperationState.SyncResult),
		)
	}
	return ""
}

func generateSyncResultComment(syncResult *argocdv1alpha1.SyncOperationResult) string {
	if syncResult == nil {
		return ""
	}
	var b strings.Builder
	for _, r := range syncResult.Resources {
		namespacedName := r.Namespace + "/" + r.Name
		switch r.Status {
		case synccommon.ResultCodeSyncFailed, synccommon.ResultCodePruneSkipped:
			b.WriteString(fmt.Sprintf("- %s `%s`: %s\n", r.Status, namespacedName, r.Message))
		}
	}
	return b.String()
}

func NewCommentOnOnHealthChanged(app argocdv1alpha1.Application, argocdURL string) *Comment {
	if app.Spec.Source == nil {
		return nil
	}
	repository := github.ParseRepositoryURL(app.Spec.Source.RepoURL)
	if repository == nil {
		return nil
	}
	revision := argocd.GetDeployedRevision(app)
	if revision == "" {
		return nil
	}
	body := generateCommentOnHealthChanged(app, argocdURL)
	if body == "" {
		return nil
	}
	return &Comment{
		GitHubRepository: *repository,
		Revision:         revision,
		Body:             body,
	}
}

func generateCommentOnHealthChanged(app argocdv1alpha1.Application, argocdURL string) string {
	revision := argocd.GetDeployedRevision(app)
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", argocdURL, app.Name)
	switch app.Status.Health.Status {
	case health.HealthStatusHealthy:
		return fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
			":white_check_mark:",
			app.Status.Health.Status,
			app.Name,
			argocdApplicationURL,
			revision,
		)
	case health.HealthStatusDegraded:
		return fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
			":x:",
			app.Status.Health.Status,
			app.Name,
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
		return fmt.Errorf("unable to create comment(s) on revision %s: %w", comment.Revision, err)
	}
	logger.Info("created comment(s)", "pulls", relatedPullNumbers)
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
