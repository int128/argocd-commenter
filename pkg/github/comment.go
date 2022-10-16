package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v47/github"
)

type Comment struct {
	Repository Repository
	CommitSHA  string
	Body       string
}

func (c *client) CreateComment(ctx context.Context, r Repository, pullNumber int, body string) error {
	_, _, err := c.rest.Issues.CreateComment(ctx, r.Owner, r.Name, pullNumber,
		&github.IssueComment{Body: github.String(body)})
	if err != nil {
		return fmt.Errorf("could not create a comment to the pull request #%d: %w", pullNumber, err)
	}
	return nil
}
