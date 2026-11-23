package argocd

import (
	"testing"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetApplicationExternalURL(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		externalURL := GetApplicationExternalURL(argocdv1alpha1.Application{})
		if externalURL != "" {
			t.Errorf("externalURL wants empty but got %s", externalURL)
		}
	})
	t.Run("Single URL", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			Status: argocdv1alpha1.ApplicationStatus{
				Summary: argocdv1alpha1.ApplicationSummary{
					ExternalURLs: []string{"https://example.com"},
				},
			},
		}
		externalURL := GetApplicationExternalURL(app)
		if want := "https://example.com"; externalURL != want {
			t.Errorf("externalURL wants %s but got %s", want, externalURL)
		}
	})
	t.Run("Multiple URLs", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			Status: argocdv1alpha1.ApplicationStatus{
				Summary: argocdv1alpha1.ApplicationSummary{
					ExternalURLs: []string{"https://example.com", "https://example.org"},
				},
			},
		}
		externalURL := GetApplicationExternalURL(app)
		if want := "https://example.com"; externalURL != want {
			t.Errorf("externalURL wants %s but got %s", want, externalURL)
		}
	})
	t.Run("Single URL with pipe", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			Status: argocdv1alpha1.ApplicationStatus{
				Summary: argocdv1alpha1.ApplicationSummary{
					ExternalURLs: []string{"Argo CD|https://example.com/argocd"},
				},
			},
		}
		externalURL := GetApplicationExternalURL(app)
		if want := "https://example.com/argocd"; externalURL != want {
			t.Errorf("externalURL wants %s but got %s", want, externalURL)
		}
	})
}

func TestRepoURLFilter(t *testing.T) {
	t.Run("neither annotation set allows all", func(t *testing.T) {
		app := argocdv1alpha1.Application{}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/repo-a") {
			t.Error("expected Allows to return true")
		}
	})

	t.Run("nil annotations allows all", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{Annotations: nil},
		}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/anything") {
			t.Error("expected Allows to return true")
		}
	})

	t.Run("include only allows listed URLs", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationIncludeRepoURLs: "https://github.com/org/repo-a",
				},
			},
		}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/repo-a") {
			t.Error("expected repo-a to be allowed")
		}
		if f.Allows("https://github.com/org/repo-b") {
			t.Error("expected repo-b to be denied")
		}
	})

	t.Run("exclude blocks listed URLs", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationExcludeRepoURLs: "https://github.com/org/repo-a",
				},
			},
		}
		f := NewRepoURLFilter(app)
		if f.Allows("https://github.com/org/repo-a") {
			t.Error("expected repo-a to be denied")
		}
		if !f.Allows("https://github.com/org/repo-b") {
			t.Error("expected repo-b to be allowed")
		}
	})

	t.Run("include takes priority over exclude", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationIncludeRepoURLs: "https://github.com/org/repo-a",
					AnnotationExcludeRepoURLs: "https://github.com/org/repo-a",
				},
			},
		}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/repo-a") {
			t.Error("expected repo-a to be allowed (include takes priority)")
		}
		if f.Allows("https://github.com/org/repo-b") {
			t.Error("expected repo-b to be denied (not in include list)")
		}
	})

	t.Run("multiple semicolon-separated URLs", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationIncludeRepoURLs: "https://github.com/org/repo-a;https://github.com/org/repo-b",
				},
			},
		}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/repo-a") {
			t.Error("expected repo-a to be allowed")
		}
		if !f.Allows("https://github.com/org/repo-b") {
			t.Error("expected repo-b to be allowed")
		}
		if f.Allows("https://github.com/org/repo-c") {
			t.Error("expected repo-c to be denied")
		}
	})

	t.Run("trailing .git in annotation matches URL without .git", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationIncludeRepoURLs: "https://github.com/org/repo-a.git",
				},
			},
		}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/repo-a") {
			t.Error("expected URL without .git to match annotation with .git")
		}
	})

	t.Run("URL with .git matches annotation without .git", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationIncludeRepoURLs: "https://github.com/org/repo-a",
				},
			},
		}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/repo-a.git") {
			t.Error("expected URL with .git to match annotation without .git")
		}
	})

	t.Run("empty annotation value treated as not set", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationIncludeRepoURLs: "",
				},
			},
		}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/anything") {
			t.Error("expected empty include annotation to allow all")
		}
	})

	t.Run("whitespace around URLs is trimmed", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationIncludeRepoURLs: " https://github.com/org/repo-a ; https://github.com/org/repo-b ",
				},
			},
		}
		f := NewRepoURLFilter(app)
		if !f.Allows("https://github.com/org/repo-a") {
			t.Error("expected repo-a to be allowed after trimming")
		}
		if !f.Allows("https://github.com/org/repo-b") {
			t.Error("expected repo-b to be allowed after trimming")
		}
	})
}
