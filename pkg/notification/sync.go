package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) NotifySync(ctx context.Context, a argocdv1alpha1.Application) error {
	repository, err := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if err != nil {
		return nil
	}
	comment := github.Comment{
		Repository: *repository,
		CommitSHA:  a.Status.Sync.Revision,
		Body:       syncStatusCommentFor(a),
	}
	if err := c.ghc.AddComment(ctx, comment); err != nil {
		return fmt.Errorf("unable to add a comment: %w", err)
	}
	return nil
}

func syncStatusCommentFor(a argocdv1alpha1.Application) string {
	if a.Status.Sync.Status == argocdv1alpha1.SyncStatusCodeSynced {
		return fmt.Sprintf("## :white_check_mark: %s: %s\nSynced to %s",
			a.Status.Sync.Status,
			a.Name,
			a.Status.Sync.Revision)
	}
	return fmt.Sprintf("## :warning: %s: %s\nSyncing to %s",
		a.Status.Sync.Status,
		a.Name,
		a.Status.Sync.Revision)
}
