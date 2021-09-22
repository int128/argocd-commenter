package github

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/google/go-github/v39/github"
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
	Deployment  Deployment
	State       string
	Description string
}

func (c *client) CreateDeploymentStatus(ctx context.Context, ds DeploymentStatus) error {
	_, _, err := c.rest.Repositories.CreateDeploymentStatus(ctx,
		ds.Deployment.Repository.Owner, ds.Deployment.Repository.Name, ds.Deployment.Id,
		&github.DeploymentStatusRequest{
			State:       github.String(ds.State),
			Description: github.String(ds.Description),
		})
	if err != nil {
		return fmt.Errorf("GitHub API error: %w", err)
	}
	return nil
}
