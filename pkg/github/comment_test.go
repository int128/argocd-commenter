package github

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shurcooL/githubv4"
)

func Test_aggregateAssociatedPullRequests(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		got := aggregateAssociatedPullRequests(
			Repository{Owner: "int128", Name: "argocd-commenter"},
			[]associatedPullRequestNode{},
			"Awesome commit message",
		)
		want := map[PullRequest]githubv4.ID{}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("mismatch (-got +want):\n%s", diff)
		}
	})

	t.Run("only in associatedPullRequestNodes", func(t *testing.T) {
		got := aggregateAssociatedPullRequests(
			Repository{Owner: "int128", Name: "argocd-commenter"},
			[]associatedPullRequestNode{
				{
					ID:     "PULL_REQUEST_ID_123",
					Number: 123,
				},
			},
			"Awesome commit message",
		)
		want := map[PullRequest]githubv4.ID{
			PullRequest{Repository: Repository{Owner: "int128", Name: "argocd-commenter"}, Number: 123}: "PULL_REQUEST_ID_123",
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("mismatch (-got +want):\n%s", diff)
		}
	})

	t.Run("only in message", func(t *testing.T) {
		got := aggregateAssociatedPullRequests(
			Repository{Owner: "int128", Name: "argocd-commenter"},
			[]associatedPullRequestNode{},
			"<a class=\"issue-link js-issue-link\" data-error-text=\"Failed to load title\" data-id=\"944916738\" data-permission-text=\"Title is private\" data-url=\"https://github.com/int128/sandbox/issues/2934\" data-hovercard-type=\"pull_request\" data-hovercard-url=\"/int128/sandbox/pull/2934/hovercard\" href=\"https://github.com/int128/sandbox/pull/2934\">int128/sandbox#2934</a>\n<a class=\"commit-link\" data-hovercard-type=\"commit\" data-hovercard-url=\"https://github.com/int128/sandbox/commit/f0e945677cfc4feb5a061e648f495ff684c90f8f/hovercard\" href=\"https://github.com/int128/sandbox/commit/f0e945677cfc4feb5a061e648f495ff684c90f8f\">int128/sandbox@<tt>f0e9456</tt></a>",
		)
		want := map[PullRequest]githubv4.ID{
			PullRequest{Repository: Repository{Owner: "int128", Name: "sandbox"}, Number: 2934}: nil,
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("mismatch (-got +want):\n%s", diff)
		}
	})

	t.Run("duplicated pull request", func(t *testing.T) {
		got := aggregateAssociatedPullRequests(
			Repository{Owner: "int128", Name: "argocd-commenter"},
			[]associatedPullRequestNode{
				{
					ID:     "PULL_REQUEST_ID_284",
					Number: 284,
				},
			},
			"Fixture for <a class=\"issue-link js-issue-link\" data-error-text=\"Failed to load title\" data-id=\"941246813\" data-permission-text=\"Title is private\" data-url=\"https://github.com/int128/argocd-commenter/issues/284\" data-hovercard-type=\"pull_request\" data-hovercard-url=\"/int128/argocd-commenter/pull/284/hovercard\" href=\"https://github.com/int128/argocd-commenter/pull/284\">#284</a> (test1)",
		)
		want := map[PullRequest]githubv4.ID{
			PullRequest{Repository: Repository{Owner: "int128", Name: "argocd-commenter"}, Number: 284}: "PULL_REQUEST_ID_284",
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("mismatch (-got +want):\n%s", diff)
		}
	})
}
