package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

type CommitComment struct {
	Repository Repository
	CommitSHA  string
	Body       string
}

func CreateCommitComment(ctx context.Context, c CommitComment) error {
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")}))
	client := github.NewClient(tc)
	_, _, err := client.Repositories.CreateComment(ctx, c.Repository.Owner, c.Repository.Name, c.CommitSHA, &github.RepositoryComment{Body: &c.Body})
	if err != nil {
		return fmt.Errorf("error response from CreateComment: %w", err)
	}
	return nil
}
