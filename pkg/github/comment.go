package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v47/github"
)

type Comment struct {
	Repository Repository
	CommitSHA  string
	Body       string
}

func (c *client) CreateComment(ctx context.Context, r Repository, pulls []int, body string) error {
	var errs []string
	for _, pull := range pulls {
		_, _, err := c.rest.Issues.CreateComment(ctx, r.Owner, r.Name, pull,
			&github.IssueComment{Body: github.String(body)})
		if err != nil {
			errs = append(errs, fmt.Sprintf("pull request #%d: %s", pull, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("could not comment to pull request(s):\n%s", strings.Join(errs, "\n"))
	}
	return nil
}
