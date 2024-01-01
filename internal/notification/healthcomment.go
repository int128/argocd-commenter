package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

func (c client) CreateCommentsOnHealthChanged(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error {
	var errs []error
	sourceRevisions := argocd.GetSourceRevisions(app)
	for _, sourceRevision := range sourceRevisions {
		comment := generateCommentOnHealthChanged(app, argocdURL, sourceRevision)
		if comment == nil {
			continue
		}
		if err := c.createComment(ctx, *comment, app); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func generateCommentOnHealthChanged(app argocdv1alpha1.Application, argocdURL string, sourceRevision argocd.SourceRevision) *Comment {
	repository := github.ParseRepositoryURL(sourceRevision.Source.RepoURL)
	if repository == nil {
		return nil
	}
	body := generateCommentBodyOnHealthChanged(app, argocdURL, sourceRevision)
	if body == "" {
		return nil
	}
	return &Comment{
		GitHubRepository: *repository,
		Revision:         sourceRevision.Revision,
		Body:             body,
	}
}

func generateCommentBodyOnHealthChanged(app argocdv1alpha1.Application, argocdURL string, sourceRevision argocd.SourceRevision) string {
	argocdApplicationURL := fmt.Sprintf("%s/applications/%s", argocdURL, app.Name)
	switch app.Status.Health.Status {
	case health.HealthStatusHealthy:
		return fmt.Sprintf("## %s %s: [%s](%s)\nDeployed %s",
			":white_check_mark:",
			app.Status.Health.Status,
			app.Name,
			argocdApplicationURL,
			sourceRevision.Revision,
		)
	case health.HealthStatusDegraded:
		return fmt.Sprintf("## %s %s: [%s](%s)\nError while deploying %s:\n%s",
			":x:",
			app.Status.Health.Status,
			app.Name,
			argocdApplicationURL,
			sourceRevision.Revision,
			generateCommentResourcesHealth(app),
		)
	}
	return ""
}

func generateCommentResourcesHealth(app argocdv1alpha1.Application) string {
	var b strings.Builder
	for _, r := range app.Status.Resources {
		if r.Health == nil {
			continue
		}
		namespacedName := r.Namespace + "/" + r.Name
		switch r.Health.Status {
		case health.HealthStatusDegraded, health.HealthStatusMissing:
			b.WriteString(fmt.Sprintf("- %s `%s`: %s\n", r.Health.Status, namespacedName, r.Health.Message))
		}
	}
	return b.String()
}
