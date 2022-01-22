package github

import (
	"context"
	"regexp"
)

type Client interface {
	ListPullRequests(ctx context.Context, r Repository, revision string) ([]PullRequest, error)
	CreateComment(ctx context.Context, r Repository, pulls []int, body string) error
	CreateDeploymentStatus(ctx context.Context, d Deployment, ds DeploymentStatus) error
}

type Repository struct {
	Owner string
	Name  string
}

var patternRepositoryURL = regexp.MustCompile(`^https://github\.com/([^/]+?)/([^/]+?)(\.git)?$`)

func ParseRepositoryURL(s string) *Repository {
	m := patternRepositoryURL.FindStringSubmatch(s)
	if len(m) < 3 {
		return nil
	}
	return &Repository{Owner: m[1], Name: m[2]}
}

type PullRequest struct {
	Number int
	Files  []string
}
