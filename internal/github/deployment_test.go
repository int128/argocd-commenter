package github

import "testing"

func TestParseDeploymentURL(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		d := ParseDeploymentURL("https://api.github.com/repos/int128/sandbox/deployments/422988781")
		if d == nil {
			t.Fatalf("deployment was nil")
		}
		if want := (Repository{Owner: "int128", Name: "sandbox"}); d.Repository != want {
			t.Errorf("want %+v but was %+v", want, d.Repository)
		}
		if d.Id != 422988781 {
			t.Errorf("want %d but was %d", 422988781, d.Id)
		}
	})

	t.Run("not deployment", func(t *testing.T) {
		d := ParseDeploymentURL("https://api.github.com/repos/int128/sandbox")
		if d != nil {
			t.Errorf("want nil but was %+v", d)
		}
	})

	t.Run("empty", func(t *testing.T) {
		d := ParseDeploymentURL("")
		if d != nil {
			t.Errorf("want nil but was %+v", d)
		}
	})
}
