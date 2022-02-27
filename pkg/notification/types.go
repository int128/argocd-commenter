package notification

import (
	"context"

	"github.com/int128/argocd-commenter/pkg/github"
)

type Client interface {
	Comment(context.Context, Event) error
	Deployment(context.Context, Event) error
	CheckRun(context.Context, Event) error
}

func NewClient(ghc github.Client) Client {
	return &client{ghc: ghc}
}

type client struct {
	ghc github.Client
}
