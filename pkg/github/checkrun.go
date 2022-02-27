package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/v39/github"
	"regexp"
)

type Commit struct {
	Repository Repository
	SHA        string
}

var patternCommitURL = regexp.MustCompile(`^https://api\.github\.com/repos/(.+?)/(.+?)/commits/([0-9a-f]+)$`)

func ParseCommitURL(s string) *Commit {
	m := patternCommitURL.FindStringSubmatch(s)
	if len(m) != 4 {
		return nil
	}
	return &Commit{
		Repository: Repository{Owner: m[1], Name: m[2]},
		SHA:        m[3],
	}
}

type CheckRun struct {
	Name       string
	Status     string
	Conclusion string
	Title      string
	Summary    string
}

func (c *client) CreateCheckRun(ctx context.Context, commit Commit, cr CheckRun) error {
	o := github.CreateCheckRunOptions{
		HeadSHA: commit.SHA,
		Name:    cr.Name,
		Status:  github.String(cr.Status),
		Output: &github.CheckRunOutput{
			Title:   github.String(cr.Title),
			Summary: github.String(cr.Summary),
		},
	}
	if cr.Conclusion != "" {
		o.Conclusion = github.String(cr.Conclusion)
	}
	_, _, err := c.rest.Checks.CreateCheckRun(ctx, commit.Repository.Owner, commit.Repository.Name, o)
	if err != nil {
		return fmt.Errorf("GitHub API error: %w", err)
	}
	return nil
}
