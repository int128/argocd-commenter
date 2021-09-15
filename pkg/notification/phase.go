package notification

import (
	"context"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) NotifyPhase(ctx context.Context, a argocdv1alpha1.Application) error {
	repository, err := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if err != nil {
		return nil
	}
	comment := github.Comment{
		Repository: *repository,
		CommitSHA:  a.Status.Sync.Revision,
		Body:       phaseCommentFor(a),
	}
	if err := c.ghc.AddComment(ctx, comment); err != nil {
		return fmt.Errorf("unable to add a comment: %w", err)
	}
	return nil
}

func phaseCommentFor(a argocdv1alpha1.Application) string {
	var resources strings.Builder
	if a.Status.OperationState.SyncResult != nil {
		for _, r := range a.Status.OperationState.SyncResult.Resources {
			namespacedName := r.Namespace + "/" + r.Name
			switch r.Status {
			case common.ResultCodeSyncFailed, common.ResultCodePruneSkipped:
				_, _ = fmt.Fprintf(&resources, "- %s `%s`: %s\n", r.Status, namespacedName, r.Message)
			}
		}
	}

	return fmt.Sprintf("## :x: Sync %s: %s\nError while syncing to %s\n%s",
		a.Status.OperationState.Phase,
		a.Name,
		a.Status.Sync.Revision,
		resources.String(),
	)
}
