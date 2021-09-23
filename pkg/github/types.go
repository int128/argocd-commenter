package github

import (
	"context"
	"regexp"
)

type Client interface {
	AddComment(ctx context.Context, comment Comment) error
	CreateDeploymentStatus(ctx context.Context, ds DeploymentStatus) error
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
