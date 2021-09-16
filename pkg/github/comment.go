package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v39/github"
)

type Comment struct {
	Repository Repository
	CommitSHA  string
	Body       string
}

func (c *client) AddComment(ctx context.Context, comment Comment) error {
	pulls, _, err := c.rest.PullRequests.ListPullRequestsWithCommit(ctx,
		comment.Repository.Owner, comment.Repository.Name, comment.CommitSHA, nil)
	if err != nil {
		return fmt.Errorf("could not list pull requests with commit: %w", err)
	}

	var errs []string
	for _, pull := range pulls {
		_, _, err := c.rest.PullRequests.CreateComment(ctx,
			comment.Repository.Owner, comment.Repository.Name, pull.GetNumber(),
			&github.PullRequestComment{
				Body: github.String(comment.Body),
			})
		if err != nil {
			errs = append(errs, fmt.Sprintf("pull request #%d: %s", pull.GetNumber(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("could not comment to pull request(s):\n%s", strings.Join(errs, "\n"))
	}
	return nil
}
