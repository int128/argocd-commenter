package github

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Repository struct {
	Owner string
	Name  string
}

func ParseRepositoryURL(urlstr string) (*Repository, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	path := u.Path
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimPrefix(path, "/")
	c := strings.SplitN(path, "/", 2)
	if len(c) != 2 {
		return nil, fmt.Errorf("invalid path %s", u.Path)
	}
	return &Repository{Owner: c[0], Name: c[1]}, nil
}

var pullRequestPattern = regexp.MustCompile(`https://github\.com/([\w-]+?)/([\w-]+?)/pull/(\d+)`)

type PullRequest struct {
	Repository
	Number int
}

func FindPullRequestURLs(s string) []PullRequest {
	matches := pullRequestPattern.FindAllStringSubmatch(s, -1)
	if matches == nil {
		return nil
	}
	var p []PullRequest
	for _, m := range matches {
		owner, name, number := m[1], m[2], m[3]
		n, err := strconv.Atoi(number)
		if err != nil {
			continue
		}
		p = append(p, PullRequest{Repository: Repository{Owner: owner, Name: name}, Number: n})
	}
	return p
}
