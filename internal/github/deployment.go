package github

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/google/go-github/v71/github"
)

type Deployment struct {
	Repository Repository
	Id         int64
}

var patternDeploymentURL = regexp.MustCompile(`^https://api\.github\.com/repos/(.+?)/(.+?)/deployments/(\d+)$`)

// ParseDeploymentURL parses the URL.
// For example, https://api.github.com/repos/int128/sandbox/deployments/422988781
func ParseDeploymentURL(s string) *Deployment {
	m := patternDeploymentURL.FindStringSubmatch(s)
	if len(m) != 4 {
		return nil
	}
	id, err := strconv.ParseInt(m[3], 10, 64)
	if err != nil {
		return nil
	}
	return &Deployment{
		Repository: Repository{Owner: m[1], Name: m[2]},
		Id:         int64(id),
	}
}

type DeploymentStatus struct {
	State          string
	Description    string
	LogURL         string
	EnvironmentURL string
}

func (c *client) CreateDeploymentStatus(ctx context.Context, d Deployment, ds DeploymentStatus) error {
	r := github.DeploymentStatusRequest{
		State:       github.Ptr(ds.State),
		Description: github.Ptr(ds.Description),
	}
	if ds.LogURL != "" {
		r.LogURL = github.Ptr(ds.LogURL)
	}
	if ds.EnvironmentURL != "" {
		r.EnvironmentURL = github.Ptr(ds.EnvironmentURL)
	}
	_, _, err := c.rest.Repositories.CreateDeploymentStatus(ctx, d.Repository.Owner, d.Repository.Name, d.Id, &r)
	if err != nil {
		return fmt.Errorf("GitHub API error: %w", err)
	}
	return nil
}

func (c *client) FindLatestDeploymentStatus(ctx context.Context, d Deployment) (*DeploymentStatus, error) {
	r, _, err := c.rest.Repositories.ListDeploymentStatuses(ctx, d.Repository.Owner, d.Repository.Name, d.Id, nil)
	if err != nil {
		return nil, fmt.Errorf("GitHub API error: %w", err)
	}
	if len(r) == 0 {
		return nil, nil
	}
	ds := r[0]
	return &DeploymentStatus{
		State:          ds.GetState(),
		Description:    ds.GetDescription(),
		LogURL:         ds.GetLogURL(),
		EnvironmentURL: ds.GetEnvironmentURL(),
	}, nil
}
