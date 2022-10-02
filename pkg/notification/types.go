package notification

import (
	"context"

	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	"github.com/int128/argocd-commenter/pkg/github"
)

type Client interface {
	Comment(context.Context, Event) error
	Deployment(context.Context, Event) error
	InactivateDeployment(context.Context, argocdcommenterv1.ApplicationHealth) error
}

func NewClient(ghc github.Client) Client {
	return &client{ghc: ghc}
}

type client struct {
	ghc github.Client
}
