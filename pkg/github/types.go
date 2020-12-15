package github

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v33/github"
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

func IsRetryableError(err error) bool {
	if errors.Is(err, &github.RateLimitError{}) {
		return true
	}

	var errorResponse *github.ErrorResponse
	if errors.As(err, &errorResponse) {
		sc := errorResponse.Response.StatusCode
		return sc >= 500 && sc <= 599
	}

	return false
}
