package notification

import (
	"testing"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/int128/argocd-commenter/internal/argocd"
	"github.com/int128/argocd-commenter/internal/github"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_isPullRequestRelatedToEvent(t *testing.T) {
	t.Run("source path matches", func(t *testing.T) {
		pull := github.PullRequest{
			Files: []string{
				"applications/app1/deployment.yaml",
				"applications/app2/deployment.yaml",
			},
		}
		sourceRevision := argocd.SourceRevision{
			Source: argocdv1alpha1.ApplicationSource{
				Path: "applications/app2",
			},
		}
		got := isPullRequestRelatedToEvent(pull, sourceRevision, nil)
		const want = true
		if want != got {
			t.Errorf("isPullRequestRelatedToEvent wants %v but was %v", want, got)
		}
	})

	t.Run("manifest generate path matches", func(t *testing.T) {
		pull := github.PullRequest{
			Files: []string{
				"applications/app1/deployment.yaml",
				"applications/app2/deployment.yaml",
			},
		}
		sourceRevision := argocd.SourceRevision{
			Source: argocdv1alpha1.ApplicationSource{
				Path: "applications/app3",
			},
		}
		manifestGeneratePaths := []string{"/applications/app1"}
		got := isPullRequestRelatedToEvent(pull, sourceRevision, manifestGeneratePaths)
		const want = true
		if want != got {
			t.Errorf("isPullRequestRelatedToEvent wants %v but was %v", want, got)
		}
	})

	t.Run("no match", func(t *testing.T) {
		pull := github.PullRequest{
			Files: []string{
				"applications/app1/deployment.yaml",
				"applications/app2/deployment.yaml",
			},
		}
		sourceRevision := argocd.SourceRevision{
			Source: argocdv1alpha1.ApplicationSource{
				Path: "applications/app3",
			},
		}
		manifestGeneratePaths := []string{"/applications/app4"}
		got := isPullRequestRelatedToEvent(pull, sourceRevision, manifestGeneratePaths)
		const want = false
		if want != got {
			t.Errorf("isPullRequestRelatedToEvent wants %v but was %v", want, got)
		}
	})
}

func Test_getManifestGeneratePaths(t *testing.T) {
	t.Run("nil annotation", func(t *testing.T) {
		manifestGeneratePaths := getManifestGeneratePaths(argocdv1alpha1.Application{})
		if manifestGeneratePaths != nil {
			t.Errorf("manifestGeneratePaths wants nil but was %+v", manifestGeneratePaths)
		}
	})

	t.Run("empty annotation", func(t *testing.T) {
		t.Run("single source", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": "",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Source: &argocdv1alpha1.ApplicationSource{
						Path: "/applications/app1",
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			if manifestGeneratePaths != nil {
				t.Errorf("manifestGeneratePaths wants nil but was %+v", manifestGeneratePaths)
			}
		})
		t.Run("multiple sources", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": "",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Sources: argocdv1alpha1.ApplicationSources{
						{Path: "/applications/app1"},
						{Path: "/applications/app2"},
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			if manifestGeneratePaths != nil {
				t.Errorf("manifestGeneratePaths wants nil but was %+v", manifestGeneratePaths)
			}
		})
	})

	t.Run("absolute path", func(t *testing.T) {
		t.Run("single source", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": "/components/app1",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Source: &argocdv1alpha1.ApplicationSource{
						Path: "/applications/app1",
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			want := []string{"/components/app1"}
			if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
				t.Errorf("want != manifestGeneratePaths:\n%s", diff)
			}
		})
		t.Run("multiple sources", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": "/components/app1",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Sources: argocdv1alpha1.ApplicationSources{
						{Path: "/applications/app1"},
						{Path: "/applications/app2"},
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			want := []string{"/components/app1"}
			if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
				t.Errorf("want != manifestGeneratePaths:\n%s", diff)
			}
		})
	})

	t.Run("relative path of ascendant", func(t *testing.T) {
		t.Run("single source", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": "../manifests1",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Source: &argocdv1alpha1.ApplicationSource{
						Path: "/applications/app1",
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			want := []string{"/applications/manifests1"}
			if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
				t.Errorf("want != manifestGeneratePaths:\n%s", diff)
			}
		})
		t.Run("multiple sources", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": "../manifests1",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Sources: argocdv1alpha1.ApplicationSources{
						{Path: "/applications/app1"},
						{Path: "/applications/app2"},
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			want := []string{"/applications/manifests1"}
			if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
				t.Errorf("want != manifestGeneratePaths:\n%s", diff)
			}
		})
	})

	t.Run("relative path of period", func(t *testing.T) {
		t.Run("single source", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": ".",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Source: &argocdv1alpha1.ApplicationSource{
						Path: "/applications/app1",
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			want := []string{"/applications/app1"}
			if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
				t.Errorf("want != manifestGeneratePaths:\n%s", diff)
			}
		})
		t.Run("multiple sources", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": ".",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Sources: argocdv1alpha1.ApplicationSources{
						{Path: "/applications/app1"},
						{Path: "/applications/app2"},
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			want := []string{
				"/applications/app1",
				"/applications/app2",
			}
			if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
				t.Errorf("want != manifestGeneratePaths:\n%s", diff)
			}
		})
	})

	t.Run("multiple paths", func(t *testing.T) {
		t.Run("single source", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": ".;../manifests1",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Source: &argocdv1alpha1.ApplicationSource{
						Path: "/applications/app1",
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			want := []string{
				"/applications/app1",
				"/applications/manifests1",
			}
			if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
				t.Errorf("want != manifestGeneratePaths:\n%s", diff)
			}
		})
		t.Run("multiple sources", func(t *testing.T) {
			app := argocdv1alpha1.Application{
				ObjectMeta: v1meta.ObjectMeta{
					Annotations: map[string]string{
						"argocd.argoproj.io/manifest-generate-paths": ".;../manifests1",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Sources: argocdv1alpha1.ApplicationSources{
						{Path: "/applications/app1"},
						{Path: "/applications/app2"},
					},
				},
			}
			manifestGeneratePaths := getManifestGeneratePaths(app)
			want := []string{
				"/applications/app1",
				"/applications/app2",
				"/applications/manifests1",
			}
			if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
				t.Errorf("want != manifestGeneratePaths:\n%s", diff)
			}
		})
	})
}
