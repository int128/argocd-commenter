package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/pkg/github"
)

func (c client) NotifyHealth(ctx context.Context, a argocdv1alpha1.Application) error {
	repository, err := github.ParseRepositoryURL(a.Spec.Source.RepoURL)
	if err != nil {
		return nil
	}
	comment := github.Comment{
		Repository: *repository,
		CommitSHA:  a.Status.Sync.Revision,
		Body:       healthStatusCommentFor(a),
	}
	if err := c.ghc.AddComment(ctx, comment); err != nil {
		return fmt.Errorf("unable to add a comment: %w", err)
	}
	return nil
}

func healthStatusCommentFor(a argocdv1alpha1.Application) string {
	if a.Status.Health.Status == health.HealthStatusHealthy {
		return fmt.Sprintf(":white_check_mark: %s: %s",
			a.Status.Health.Status,
			a.Name)
	}
	return fmt.Sprintf(":warning: %s: %s",
		a.Status.Sync.Status,
		a.Name)
}
