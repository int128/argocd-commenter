package notification

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/pkg/github"
)

type Client interface {
	CreateCommentOnPhaseChanged(context.Context, PhaseChangedEvent) error
	CreateCommentOnHealthChanged(context.Context, HealthChangedEvent) error
	CreateDeploymentStatusOnPhaseChanged(context.Context, PhaseChangedEvent) error
	CreateDeploymentStatusOnHealthChanged(context.Context, HealthChangedEvent) error
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

func GetDeploymentURL(a argocdv1alpha1.Application) string {
	return a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
}

func trimDescription(s string) string {
	// The maximum description length is 140 characters.
	// https://docs.github.com/en/rest/reference/deployments#create-a-deployment-status
	if len(s) < 140 {
		return s
	}
	return s[0:139]
}
