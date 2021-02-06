package commenter

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/go-logr/logr"

	"github.com/int128/argocd-commenter/pkg/github"
)

type ApplicationOperationState struct {
	Log logr.Logger
}

func (cmt *ApplicationOperationState) Do(ctx context.Context, application argocdv1alpha1.Application) error {
	repository, err := github.ParseRepositoryURL(application.Spec.Source.RepoURL)
	if err != nil {
		cmt.Log.Info("skip non-GitHub URL", "error", err)
		return nil
	}

	commitComment := github.CommitComment{
		Repository: *repository,
		CommitSHA:  application.Status.Sync.Revision,
		Body:       cmt.commentBody(application),
	}
	cmt.Log.Info("creating a commit comment", "commitComment", commitComment)
	if err := github.CreateCommitComment(ctx, commitComment); err != nil {
		return fmt.Errorf("could not add a comment: %w", err)
	}
	return nil
}

func (cmt *ApplicationOperationState) commentBody(application argocdv1alpha1.Application) string {
	var syncStatus string
	switch application.Status.Sync.Status {
	case argocdv1alpha1.SyncStatusCodeSynced:
		syncStatus = fmt.Sprintf(":white_check_mark: %s", application.Status.Sync.Status)
	default:
		syncStatus = fmt.Sprintf(":warning: %s", application.Status.Sync.Status)
	}

	var healthStatus string
	switch application.Status.Health.Status {
	case health.HealthStatusHealthy:
		healthStatus = fmt.Sprintf(":white_check_mark: %s", application.Status.Health.Status)
	default:
		healthStatus = fmt.Sprintf(":warning: %s", application.Status.Health.Status)
	}

	return fmt.Sprintf("%s %s (%s)", syncStatus, healthStatus, application.Name)
}
