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
					MessageHeadlineHTML string `graphql:"messageHeadlineHTML"`
					MessageBodyHTML     string `graphql:"messageBodyHTML"`
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

	associatedPullRequests := aggregateAssociatedPullRequests(
		comment.Repository,
		q.Repository.Object.Commit.AssociatedPullRequests.Nodes,
		q.Repository.Object.Commit.MessageHeadlineHTML+"\n"+q.Repository.Object.Commit.MessageBodyHTML,
	)
	for pr, prID := range associatedPullRequests {
		if prID == nil {
			q, err := c.getPullRequest(ctx, pr.Repository, pr.Number)
			if err != nil {
				return fmt.Errorf("could not get pull request %+v: %w", pr, err)
			}
			prID = q.Repository.PullRequest.ID
		}
		if err := c.addComment(ctx, prID, comment.Body); err != nil {
			return fmt.Errorf("could not comment to pull request #%d: %w", pr.Number, err)
		}
	}
	return nil
}

type associatedPullRequestNode = struct {
	ID     githubv4.ID
	Number int
}

func aggregateAssociatedPullRequests(repository Repository, associatedPullRequestNodes []associatedPullRequestNode, messageHTML string) map[PullRequest]githubv4.ID {
	m := make(map[PullRequest]githubv4.ID)
	for _, n := range associatedPullRequestNodes {
		m[PullRequest{Repository: repository, Number: n.Number}] = n.ID
	}
	for _, pr := range FindPullRequestURLs(messageHTML) {
		if _, ok := m[pr]; ok {
			continue // dedupe pull requests
		}
		m[pr] = nil // lazy resolving
	}
	return m
}

type queryPullRequest struct {
	Repository struct {
		PullRequest struct {
			ID githubv4.ID
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func (c *client) getPullRequest(ctx context.Context, repository Repository, number int) (*queryPullRequest, error) {
	var q queryPullRequest
	v := map[string]interface{}{
		"owner":  githubv4.String(repository.Owner),
		"name":   githubv4.String(repository.Name),
		"number": githubv4.Int(number),
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
		return fmt.Errorf("mutation error from GitHub API: %w", err)
	}
	return nil
}
