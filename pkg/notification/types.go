package notification

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/pkg/github"
)

type Event struct {
	PhaseIsChanged  bool
	HealthIsChanged bool
	Application     argocdv1alpha1.Application
	ArgoCDURL       string
}

type Client interface {
	Comment(context.Context, Event) error
	Deployment(context.Context, Event) error
}

func NewClient(ghc github.Client) Client {
	return &client{ghc: ghc}
}

type client struct {
	ghc github.Client
}
