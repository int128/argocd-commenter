package github

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v47/github"
	"github.com/int128/oauth2-github-app"
	"golang.org/x/oauth2"
)

type client struct {
	rest *github.Client
}

func NewClient(ctx context.Context) (Client, error) {
	oauth2Client, err := newOAuth2Client(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create an OAuth2 client: %w", err)
	}
	ghc, err := newGitHubClient(oauth2Client)
	if err != nil {
		return nil, fmt.Errorf("could not create a GitHub client: %w", err)
	}
	return &client{rest: ghc}, nil
}

func newOAuth2Client(ctx context.Context) (*http.Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})), nil
	}
	appID, installationID, privateKey := os.Getenv("GITHUB_APP_ID"), os.Getenv("GITHUB_APP_INSTALLATION_ID"), os.Getenv("GITHUB_APP_PRIVATE_KEY")
	if appID != "" && installationID != "" && privateKey != "" {
		k, err := oauth2githubapp.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("invalid GITHUB_APP_PRIVATE_KEY: %w", err)
		}
		cfg := oauth2githubapp.Config{
			PrivateKey:     k,
			AppID:          appID,
			InstallationID: installationID,
		}
		return oauth2.NewClient(ctx, cfg.TokenSource(ctx)), nil
	}
	return nil, fmt.Errorf("you need to set either GITHUB_TOKEN or GITHUB_APP_ID")
}

func newGitHubClient(hc *http.Client) (*github.Client, error) {
	ghesURL := os.Getenv("GITHUB_ENTERPRISE_URL")
	if ghesURL != "" {
		ghc, err := github.NewEnterpriseClient(ghesURL, ghesURL, hc)
		if err != nil {
			return nil, fmt.Errorf("could not create a GitHub Enterprise client: %w", err)
		}
		return ghc, nil
	}
	return github.NewClient(hc), nil
}
