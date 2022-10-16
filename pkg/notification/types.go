package notification

import (
	"context"

	"github.com/int128/argocd-commenter/pkg/github"
)

type Client interface {
	CreateCommentOnPhaseChanged(context.Context, PhaseChangedEvent) error
	CreateCommentOnHealthChanged(context.Context, HealthChangedEvent) error
	CreateDeploymentStatusOnPhaseChanged(context.Context, PhaseChangedEvent, string) error
	CreateDeploymentStatusOnHealthChanged(context.Context, HealthChangedEvent, string) error
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

func trimDescription(s string) string {
	// The maximum description length is 140 characters.
	// https://docs.github.com/en/rest/reference/deployments#create-a-deployment-status
	if len(s) < 140 {
		return s
	}
	return s[0:139]
}
