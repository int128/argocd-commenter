package github

import "testing"

func TestParseRepositoryURL(t *testing.T) {
	t.Run("valid HTTPS of GitHub", func(t *testing.T) {
		r := ParseRepositoryURL("https://github.com/int128/sandbox")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("valid HTTPS of GitHub with .git", func(t *testing.T) {
		r := ParseRepositoryURL("https://github.com/int128/sandbox.git")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("valid HTTPS of GitHub Enterprise Server", func(t *testing.T) {
		r := ParseRepositoryURL("https://ghes.example.com/int128/sandbox")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("valid HTTPS of GitHub Enterprise Server with .git", func(t *testing.T) {
		r := ParseRepositoryURL("https://ghes.example.com/int128/sandbox.git")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("invalid HTTPS", func(t *testing.T) {
		r := ParseRepositoryURL("https://example.com")
		if r != nil {
			t.Errorf("want nil but was %+v", r)
		}
	})

	// https://github.com/argoproj/argo-cd/blob/master/docs/user-guide/private-repositories.md
	t.Run("valid SSH of GitHub", func(t *testing.T) {
		r := ParseRepositoryURL("git@github.com:argoproj/argocd-example-apps.git")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "argoproj", Name: "argocd-example-apps"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("valid SSH of GitHub Enterprise Server", func(t *testing.T) {
		r := ParseRepositoryURL("git@ghes.example.com:argoproj/argocd-example-apps.git")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "argoproj", Name: "argocd-example-apps"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("empty", func(t *testing.T) {
		r := ParseRepositoryURL("")
		if r != nil {
			t.Errorf("want nil but was %+v", r)
		}
	})
}
