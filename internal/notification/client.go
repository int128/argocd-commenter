package notification

import (
	"context"
	"fmt"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/int128/argocd-commenter/internal/github"
	"k8s.io/apimachinery/pkg/util/errors"
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
	Revision         string
	Body             string
}

type client struct {
	ghc github.Client
}

func (c client) createComment(ctx context.Context, comment Comment, app argocdv1alpha1.Application) error {
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
	if err := c.createPullRequestComment(ctx, comment.GitHubRepository, relatedPullNumbers, comment.Body); err != nil {
		return fmt.Errorf("unable to create comment(s) on revision %s: %w", comment.Revision, err)
	}
	logger.Info("created comment(s)", "pulls", relatedPullNumbers)
	return nil
}

func (c client) createPullRequestComment(ctx context.Context, repository github.Repository, pullNumbers []int, body string) error {
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
