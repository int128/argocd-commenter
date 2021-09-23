package github

import "testing"

func TestParseRepositoryURL(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := ParseRepositoryURL("https://github.com/int128/sandbox")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("with .git", func(t *testing.T) {
		r := ParseRepositoryURL("https://github.com/int128/sandbox.git")
		if r == nil {
			t.Fatalf("repository was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); *r != want {
			t.Errorf("want %+v but was %+v", &want, r)
		}
	})

	t.Run("not repository", func(t *testing.T) {
		r := ParseRepositoryURL("https://github.com/int128/sandbox/commits")
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
