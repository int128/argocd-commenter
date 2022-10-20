package notification

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/pkg/github"
)

type PhaseChangedEvent struct {
	Application argocdv1alpha1.Application
	ArgoCDURL   string
}

type HealthChangedEvent struct {
	Application argocdv1alpha1.Application
	ArgoCDURL   string
}

type DeletionEvent struct {
	Application argocdv1alpha1.Application
	ArgoCDURL   string
}

type Client interface {
	CreateCommentOnPhaseChanged(context.Context, PhaseChangedEvent) error
	CreateCommentOnHealthChanged(context.Context, HealthChangedEvent) error
	CreateDeploymentStatusOnPhaseChanged(context.Context, PhaseChangedEvent, string) error
	CreateDeploymentStatusOnHealthChanged(context.Context, HealthChangedEvent, string) error
	CreateDeploymentStatusOnDeletion(context.Context, DeletionEvent, string) error
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
