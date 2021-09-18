package github

import (
	"fmt"
	"net/url"
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
