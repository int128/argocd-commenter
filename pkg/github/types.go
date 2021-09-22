package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type Client interface {
	AddComment(ctx context.Context, comment Comment) error
	CreateDeploymentStatus(ctx context.Context, ds DeploymentStatus) error
}

type Repository struct {
	Owner string
	Name  string
}

func ParseRepositoryURL(urlstr string) (*Repository, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	path := u.Path
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimPrefix(path, "/")
	c := strings.SplitN(path, "/", 2)
	if len(c) != 2 {
		return nil, fmt.Errorf("invalid path %s", u.Path)
	}
	return &Repository{Owner: c[0], Name: c[1]}, nil
}
