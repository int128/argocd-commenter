package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
)

type Comment struct {
	Repository Repository
	CommitSHA  string
	Body       string
}

func (c *client) AddComment(ctx context.Context, comment Comment) error {
	var q struct {
		Repository struct {
			Object struct {
				Commit struct {
					AssociatedPullRequests struct {
						Nodes []struct {
							ID     githubv4.ID
							Number int
						}
					} `graphql:"associatedPullRequests(first: 3)"`
				} `graphql:"... on Commit"`
			} `graphql:"object(oid: $commitSHA)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	v := map[string]interface{}{
		"owner":     githubv4.String(comment.Repository.Owner),
		"name":      githubv4.String(comment.Repository.Name),
		"commitSHA": githubv4.GitObjectID(comment.CommitSHA),
	}
	if err := c.graphql.Query(ctx, &q, v); err != nil {
		return fmt.Errorf("could not get commit %s: %w", comment.CommitSHA, err)
	}

	var errs []string
	for _, pr := range q.Repository.Object.Commit.AssociatedPullRequests.Nodes {
		if err := c.addComment(ctx, pr.ID, comment.Body); err != nil {
			errs = append(errs, fmt.Sprintf("pull request #%d: %s", pr.Number, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("could not comment to pull request(s):\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

func (c *client) addComment(ctx context.Context, id githubv4.ID, body string) error {
	var m struct {
		AddComment struct {
			Subject struct {
				ID githubv4.ID
			}
		} `graphql:"addComment(input: $input)"`
	}
	input := githubv4.AddCommentInput{
		SubjectID: id,
		Body:      githubv4.String(body),
	}
	if err := c.graphql.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("mutation error from GitHub API: %w", err)
	}
	return nil
}
