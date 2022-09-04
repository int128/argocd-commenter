package github

import "testing"

func TestParseRepositoryURL(t *testing.T) {
	t.Run("valid HTTPS", func(t *testing.T) {
		r := ParseRepositoryURL("https://github.com/int128/sandbox")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("HTTPS with .git", func(t *testing.T) {
		r := ParseRepositoryURL("https://github.com/int128/sandbox.git")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("HTTPS but not repository", func(t *testing.T) {
		r := ParseRepositoryURL("https://github.com/int128/sandbox/commits")
		if r != nil {
			t.Errorf("want nil but was %+v", r)
		}
	})

	// https://github.com/argoproj/argo-cd/blob/master/docs/user-guide/private-repositories.md
	t.Run("valid SSH", func(t *testing.T) {
		r := ParseRepositoryURL("git@github.com:argoproj/argocd-example-apps.git")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "argoproj", Name: "argocd-example-apps"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("SSH but not GitHub", func(t *testing.T) {
		r := ParseRepositoryURL("git@example.com:argoproj/argocd-example-apps.git")
		if r != nil {
			t.Errorf("want nil but was %+v", r)
		}
	})

	t.Run("empty", func(t *testing.T) {
		r := ParseRepositoryURL("")
		if r != nil {
			t.Errorf("want nil but was %+v", r)
		}
	})
}
