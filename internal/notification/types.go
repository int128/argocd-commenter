package notification

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/internal/github"
)

type Client interface {
	CreateComment(ctx context.Context, comment Comment, app argocdv1alpha1.Application) error
	CreateDeployment(ctx context.Context, ds DeploymentStatus) error
	CheckIfDeploymentIsAlreadyHealthy(ctx context.Context, deploymentURL string) (bool, error)
}

func NewClient(ghc github.Client) Client {
	return &client{ghc: ghc}
}

type client struct {
	ghc github.Client
}

func IsNotFoundError(err error) bool {
	return github.IsNotFoundError(err)
}
