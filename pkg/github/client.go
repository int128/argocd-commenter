package github

import (
	"context"
	"fmt"
	"os"

	"github.com/int128/oauth2-github-app/app"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Client interface {
	AddComment(ctx context.Context, comment Comment) error
}

type client struct {
	graphql *githubv4.Client
}

func NewClient(ctx context.Context) (*client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		return newClientWithPersonalAccessToken(ctx, token), nil
	}
	appID, installationID, privateKey := os.Getenv("GITHUB_APP_ID"), os.Getenv("GITHUB_APP_INSTALLATION_ID"), os.Getenv("GITHUB_APP_PRIVATE_KEY")
	if appID != "" && installationID != "" && privateKey != "" {
		return newClientForGitHubApp(ctx, appID, installationID, privateKey)
	}
	return nil, fmt.Errorf("you need to set either GITHUB_TOKEN or GITHUB_APP_ID")
}

func newClientWithPersonalAccessToken(ctx context.Context, token string) *client {
	c := githubv4.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})))
	return &client{graphql: c}
}

func newClientForGitHubApp(ctx context.Context, appID, installationID, privateKey string) (*client, error) {
	k, err := app.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_APP_PRIVATE_KEY: %w", err)
	}
	cfg := app.Config{
		PrivateKey:     k,
		AppID:          appID,
		InstallationID: installationID,
	}
	c := githubv4.NewClient(oauth2.NewClient(ctx, cfg.TokenSource(ctx)))
	return &client{graphql: c}, nil
}
