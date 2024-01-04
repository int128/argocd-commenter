package notification

import (
	"context"
	"errors"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
)

type Client interface {
	CreateCommentsOnPhaseChanged(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error
	CreateCommentsOnHealthChanged(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error
	CreateDeploymentStatusOnPhaseChanged(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error
	CreateDeploymentStatusOnHealthChanged(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error
	CreateDeploymentStatusOnDeletion(ctx context.Context, app argocdv1alpha1.Application, argocdURL string) error

	CheckIfDeploymentIsAlreadyHealthy(ctx context.Context, deploymentURL string) (bool, error)
}

func NewClient(ghc github.Client) Client {
	return &client{ghc: ghc}
}

func IsNotFoundError(err error) bool {
	return github.IsNotFoundError(err)
}

type Comment struct {
	GitHubRepository github.Repository
	SourceRevision   argocd.SourceRevision
	Body             string
}

type client struct {
	ghc github.Client
}

func (c client) createComment(ctx context.Context, comment Comment, app argocdv1alpha1.Application) error {
	logger := logr.FromContextOrDiscard(ctx).WithValues(
		"revision", comment.SourceRevision.Revision,
		"repository", comment.GitHubRepository,
	)
	pulls, err := c.ghc.ListPullRequests(ctx, comment.GitHubRepository, comment.SourceRevision.Revision)
	if err != nil {
		return fmt.Errorf("unable to list pull requests of revision %s: %w", comment.SourceRevision.Revision, err)
	}
	relatedPulls := filterPullRequestsRelatedToEvent(pulls, comment.SourceRevision, app)
	if len(relatedPulls) == 0 {
		logger.Info("no pull request related to the revision")
		return nil
	}

	var errs []error
	for _, pull := range relatedPulls {
		if err := c.ghc.CreateComment(ctx, comment.GitHubRepository, pull.Number, comment.Body); err != nil {
			errs = append(errs, err)
			continue
		}
		logger.Info("created a comment", "pullNumber", pull.Number)
	}
	if len(errs) > 0 {
		return fmt.Errorf("unable to create comment(s) on revision %s: %w", comment.SourceRevision.Revision, errors.Join(errs...))
	}
	return nil
}

type DeploymentStatus struct {
	GitHubDeployment       github.Deployment
	GitHubDeploymentStatus github.DeploymentStatus
}

func (c client) createDeploymentStatus(ctx context.Context, ds DeploymentStatus) error {
	logger := logr.FromContextOrDiscard(ctx).WithValues(
		"deployment", ds.GitHubDeployment,
		"state", ds.GitHubDeploymentStatus.State,
	)
	if err := c.ghc.CreateDeploymentStatus(ctx, ds.GitHubDeployment, ds.GitHubDeploymentStatus); err != nil {
		return fmt.Errorf("unable to create a deployment status of %s: %w", ds.GitHubDeploymentStatus.State, err)
	}
	logger.Info("created a deployment status")
	return nil
}

func (c client) CheckIfDeploymentIsAlreadyHealthy(ctx context.Context, deploymentURL string) (bool, error) {
	deployment := github.ParseDeploymentURL(deploymentURL)
	if deployment == nil {
		return false, nil
	}
	latestDeploymentStatus, err := c.ghc.FindLatestDeploymentStatus(ctx, *deployment)
	if err != nil {
		return false, fmt.Errorf("unable to find the latest deployment status: %w", err)
	}
	if latestDeploymentStatus == nil {
		return false, nil
	}
	return latestDeploymentStatus.State == "success", nil
}
