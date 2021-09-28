package notification

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/pkg/github"
)

type Client interface {
	NotifyHealth(ctx context.Context, a argocdv1alpha1.Application, argoCDURL string) error
	NotifyPhase(ctx context.Context, a argocdv1alpha1.Application, argoCDURL string) error
}

func NewClient(ghc github.Client) Client {
	return &client{ghc: ghc}
}

type client struct {
	ghc github.Client
}
