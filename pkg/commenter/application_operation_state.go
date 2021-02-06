package commenter

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/pkg/github"
)

type ApplicationOperationState struct {
	Log logr.Logger
}

func (cmt *ApplicationOperationState) Do(ctx context.Context, application argocdv1alpha1.Application) error {
	if application.Status.OperationState == nil {
		cmt.Log.Info("skip nil operationState (never reach here)", "status", application.Status)
		return nil
	}
	repository, err := github.ParseRepositoryURL(application.Spec.Source.RepoURL)
	if err != nil {
		cmt.Log.Info("skip non-GitHub URL", "error", err)
		return nil
	}

	comment := github.CommitComment{
		Repository: *repository,
		CommitSHA:  application.Status.Sync.Revision,
		Body:       cmt.commentBody(application),
	}
	cmt.Log.Info("adding a comment", "comment", comment)
	if err := github.CreateCommitComment(ctx, comment); err != nil {
		return fmt.Errorf("could not add a comment: %w", err)
	}
	return nil
}

func (cmt *ApplicationOperationState) commentBody(application argocdv1alpha1.Application) string {
	var syncStatus string
	switch application.Status.Sync.Status {
	case argocdv1alpha1.SyncStatusCodeSynced:
		syncStatus = ":white_check_mark: Synced"
	default:
		syncStatus = fmt.Sprintf(":warning: %s", application.Status.Sync.Status)
	}

	var operationStatePhase string
	switch application.Status.OperationState.Phase {
	case common.OperationSucceeded:
		operationStatePhase = ":white_check_mark: Succeeded"
	case common.OperationRunning:
		operationStatePhase = ":gear: Syncing"
	default:
		operationStatePhase = fmt.Sprintf(":warning: %s", application.Status.OperationState.Phase)
	}

	return fmt.Sprintf("%s %s **%s** -> `/%s` @ %s",
		syncStatus,
		operationStatePhase,
		application.Name,
		application.Status.Sync.ComparedTo.Source.Path,
		application.Status.Sync.Revision,
	)
}
