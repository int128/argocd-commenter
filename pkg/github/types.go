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
