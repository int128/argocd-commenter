package github

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/go-github/v72/github"
)

type Client interface {
	ListPullRequests(ctx context.Context, r Repository, revision string) ([]PullRequest, error)
	CreatePullRequestComment(ctx context.Context, r Repository, pullNumber int, body string) error
	CreateCommitComment(ctx context.Context, r Repository, sha, body string) error
	CreateDeploymentStatus(ctx context.Context, d Deployment, ds DeploymentStatus) error
	FindLatestDeploymentStatus(ctx context.Context, d Deployment) (*DeploymentStatus, error)
}

type Repository struct {
	Owner string
	Name  string
}

var (
	patternRepositoryHTTPS = regexp.MustCompile(`^https://github\.com/([^/]+?)/([^/]+?)(\.git)?$`)
	patternRepositorySSH   = regexp.MustCompile(`^git@github\.com:([^/]+?)/([^/]+?)(\.git)?$`)
)

func ParseRepositoryURL(s string) *Repository {
	if r := parseRepositoryHTTPS(s); r != nil {
		return r
	}
	if r := parseRepositorySSH(s); r != nil {
		return r
	}
	return nil
}

func parseRepositoryHTTPS(s string) *Repository {
	m := patternRepositoryHTTPS.FindStringSubmatch(s)
	if len(m) < 3 {
		return nil
	}
	return &Repository{Owner: m[1], Name: m[2]}
}

func parseRepositorySSH(s string) *Repository {
	m := patternRepositorySSH.FindStringSubmatch(s)
	if len(m) < 3 {
		return nil
	}
	return &Repository{Owner: m[1], Name: m[2]}
}

type PullRequest struct {
	Number int
	Files  []string
}

func IsNotFoundError(err error) bool {
	var gherr *github.ErrorResponse
	if errors.As(err, &gherr) {
		if gherr.Response != nil {
			return gherr.Response.StatusCode == 404
		}
	}
	return false
}
