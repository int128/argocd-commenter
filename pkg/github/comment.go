package github

import (
	"context"
	"fmt"

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
					} `graphql:"associatedPullRequests(first: 1)"`
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
		return fmt.Errorf("query error from GitHub API: %w", err)
	}
	if len(q.Repository.Object.Commit.AssociatedPullRequests.Nodes) == 0 {
		return fmt.Errorf("could not find a pull request associated to commit %s", comment.CommitSHA)
	}
	associatedPullRequest := q.Repository.Object.Commit.AssociatedPullRequests.Nodes[0]

	var m struct {
		AddComment struct {
			Subject struct {
				ID githubv4.ID
			}
		} `graphql:"addComment(input: $input)"`
	}
	input := githubv4.AddCommentInput{
		SubjectID: associatedPullRequest.ID,
		Body:      githubv4.String(comment.Body),
	}
	if err := c.graphql.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("could not add a comment to pull request #%d: %w", associatedPullRequest.Number, err)
	}
	return nil
}
