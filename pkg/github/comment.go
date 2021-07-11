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
					} `graphql:"associatedPullRequests(first: 3)"`
					MessageHeadlineHTML string
					MessageBodyHTML     string
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

	pullRequests := make(map[PullRequest]githubv4.ID)
	for _, n := range q.Repository.Object.Commit.AssociatedPullRequests.Nodes {
		pullRequests[PullRequest{Repository: comment.Repository, Number: n.Number}] = n.ID
	}
	messageHTML := q.Repository.Object.Commit.MessageHeadlineHTML + "\n" + q.Repository.Object.Commit.MessageBodyHTML
	for _, p := range FindPullRequestURLs(messageHTML) {
		if _, ok := pullRequests[p]; !ok {
			pullRequests[p] = nil
		}
	}

	for pullRequest, pullRequestID := range pullRequests {
		if pullRequestID == nil {
			q, err := c.getPullRequest(ctx, pullRequest)
			if err != nil {
				return fmt.Errorf("could not get pull request %+v: %w", pullRequest, err)
			}
			pullRequestID = q.Repository.PullRequest.ID
		}
		if err := c.addComment(ctx, pullRequestID, comment.Body); err != nil {
			return fmt.Errorf("could not comment to pull request #%d: %w", pullRequest.Number, err)
		}
	}
	return nil
}

type queryPullRequest struct {
	Repository struct {
		PullRequest struct {
			ID githubv4.ID
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func (c *client) getPullRequest(ctx context.Context, pullRequest PullRequest) (*queryPullRequest, error) {
	var q queryPullRequest
	v := map[string]interface{}{
		"owner":  githubv4.String(pullRequest.Owner),
		"name":   githubv4.String(pullRequest.Name),
		"number": githubv4.Int(pullRequest.Number),
	}
	if err := c.graphql.Query(ctx, &q, v); err != nil {
		return nil, fmt.Errorf("query error from GitHub API: %w", err)
	}
	return &q, nil
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
		return fmt.Errorf("could not add a comment: %w", err)
	}
	return nil
}
