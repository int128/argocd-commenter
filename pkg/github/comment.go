package github

import (
	"context"
	"fmt"
	"os"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type CommitComment struct {
	Repository Repository
	CommitSHA  string
	Body       string
}

func CreateCommitComment(ctx context.Context, c CommitComment) error {
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")}))
	client := githubv4.NewClient(tc)

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
		"owner":     githubv4.String(c.Repository.Owner),
		"name":      githubv4.String(c.Repository.Name),
		"commitSHA": githubv4.GitObjectID(c.CommitSHA),
	}
	if err := client.Query(ctx, &q, v); err != nil {
		return fmt.Errorf("query error from GitHub API: %w", err)
	}
	if len(q.Repository.Object.Commit.AssociatedPullRequests.Nodes) == 0 {
		return fmt.Errorf("could not find a pull request associated to commit %s", c.CommitSHA)
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
		Body:      githubv4.String(c.Body),
	}
	if err := client.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("could not add a comment to pull request #%d: %w", associatedPullRequest.Number, err)
	}
	return nil
}
