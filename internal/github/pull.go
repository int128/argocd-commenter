package github

import (
	"context"
	"fmt"
)

func (c *client) ListPullRequests(ctx context.Context, r Repository, revision string) ([]PullRequest, error) {
	ghPulls, _, err := c.rest.PullRequests.ListPullRequestsWithCommit(ctx, r.Owner, r.Name, revision, nil)
	if err != nil {
		return nil, fmt.Errorf("could not list pull requests with commit: %w", err)
	}
	var pulls []PullRequest
	for _, pr := range ghPulls {
		prFiles, _, err := c.rest.PullRequests.ListFiles(ctx, r.Owner, r.Name, pr.GetNumber(), nil)
		if err != nil {
			return nil, fmt.Errorf("could not list files of pull request #%d: %w", pr.GetNumber(), err)
		}
		var files []string
		for _, f := range prFiles {
			files = append(files, f.GetFilename())
		}
		pulls = append(pulls, PullRequest{Number: pr.GetNumber(), Files: files})
	}
	return pulls, nil
}
