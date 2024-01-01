package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

var SyncOperationPhasesForComment = []synccommon.OperationPhase{
	synccommon.OperationRunning,
	synccommon.OperationSucceeded,
	synccommon.OperationFailed,
	synccommon.OperationError,
}

func (c client) CreateCommentsOnPhaseChanged(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error {
	var errs []error
	sourceRevisions := argocd.GetSourceRevisions(app)
	for _, sourceRevision := range sourceRevisions {
		comment := generateCommentOnPhaseChanged(app, argocdURL, sourceRevision)
		if comment == nil {
			continue
		}
		if err := c.createComment(ctx, *comment, app); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
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
			generateCommentResourcesOnPhaseChanged(app.Status.OperationState.SyncResult),
		)
	}
	return ""
}

func generateCommentResourcesOnPhaseChanged(syncResult *argocdv1alpha1.SyncOperationResult) string {
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
