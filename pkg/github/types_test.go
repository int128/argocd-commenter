package github

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindPullRequestURLs(t *testing.T) {
	t.Run("", func(t *testing.T) {
		got := FindPullRequestURLs("Fixture for <a class=\"issue-link js-issue-link\" data-error-text=\"Failed to load title\" data-id=\"941246813\" data-permission-text=\"Title is private\" data-url=\"https://github.com/int128/argocd-commenter/issues/284\" data-hovercard-type=\"pull_request\" data-hovercard-url=\"/int128/argocd-commenter/pull/284/hovercard\" href=\"https://github.com/int128/argocd-commenter/pull/284\">#284</a> (test1)")
		want := []PullRequest{
			{
				Repository: Repository{Owner: "int128", Name: "argocd-commenter"},
				Number:     284,
			},
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("mismatch (-got +want):\n%s", diff)
		}
	})
}
