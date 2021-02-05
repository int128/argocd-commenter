package commenter

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"

	"github.com/int128/argocd-commenter/pkg/github"
)

type ApplicationOperationState struct {
	Log logr.Logger
}

func (cmt *ApplicationOperationState) Do(ctx context.Context, application argocdv1alpha1.Application) error {
	cmt.Log.Info("start notification", "application", application)

	if application.Status.OperationState == nil {
		cmt.Log.Info("application.status.operationState is nil (should not reach here)", "application.status", application.Status)
		return nil
	}
	repository, err := github.ParseRepositoryURL(application.Spec.Source.RepoURL)
	if err != nil {
		cmt.Log.Info("skip non-GitHub URL", "error", err)
		return nil
	}

	commitComment := github.CommitComment{
		Repository: *repository,
		CommitSHA:  application.Status.Sync.Revision,
		Body: fmt.Sprintf("ArgoCD: %s: %s: %s",
			application.Name,
			application.Status.OperationState.Phase,
			application.Status.OperationState.Message,
		),
	}

	cmt.Log.Info("creating a commit comment", "commitComment", commitComment)
	if err := github.CreateCommitComment(ctx, commitComment); err != nil {
		return fmt.Errorf("could not add a comment: %w", err)
	}
	return nil
}
