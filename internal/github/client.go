package github

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v76/github"
	"github.com/gregjones/httpcache"
	"github.com/int128/oauth2-github-app"
	"golang.org/x/oauth2"
)

type client struct {
	rest *github.Client
}

func NewClient(ctx context.Context) (Client, error) {
	transport := httpcache.NewMemoryCacheTransport()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, transport)
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
	var (
		token          = os.Getenv("GITHUB_TOKEN")
		appID          = os.Getenv("GITHUB_APP_ID")
		installationID = os.Getenv("GITHUB_APP_INSTALLATION_ID")
		privateKey     = os.Getenv("GITHUB_APP_PRIVATE_KEY")
		ghesURL        = os.Getenv("GITHUB_ENTERPRISE_URL")
	)
	if token != "" {
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})), nil
	}
	if appID == "" || installationID == "" || privateKey == "" {
		return nil, fmt.Errorf("you need to set either GITHUB_TOKEN or GitHub App configuration")
	}
	k, err := oauth2githubapp.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_APP_PRIVATE_KEY: %w", err)
	}
	cfg := oauth2githubapp.Config{
		PrivateKey:     k,
		AppID:          appID,
		InstallationID: installationID,
		BaseURL:        ghesURL,
	}
	return oauth2.NewClient(ctx, cfg.TokenSource(ctx)), nil
}

func newGitHubClient(hc *http.Client) (*github.Client, error) {
	ghesURL := os.Getenv("GITHUB_ENTERPRISE_URL")
	if ghesURL != "" {
		ghc, err := github.NewClient(hc).WithEnterpriseURLs(ghesURL, ghesURL)
		if err != nil {
			return nil, fmt.Errorf("could not create a GitHub Enterprise client: %w", err)
		}
		return ghc, nil
	}
	return github.NewClient(hc), nil
}
