package notification

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestEvent_GetManifestGeneratePaths(t *testing.T) {
	t.Run("nil annotation", func(t *testing.T) {
		var e Event
		manifestGeneratePaths := e.GetManifestGeneratePaths()
		if manifestGeneratePaths != nil {
			t.Errorf("manifestGeneratePaths wants nil but was %+v", manifestGeneratePaths)
		}
	})

	t.Run("empty annotation", func(t *testing.T) {
		var e Event
		e.Application.Spec.Source.Path = "/applications/app1"
		e.Application.Annotations = map[string]string{
			"argocd.argoproj.io/manifest-generate-paths": "",
		}
		manifestGeneratePaths := e.GetManifestGeneratePaths()
		if manifestGeneratePaths != nil {
			t.Errorf("manifestGeneratePaths wants nil but was %+v", manifestGeneratePaths)
		}
	})

	t.Run("absolute path", func(t *testing.T) {
		var e Event
		e.Application.Spec.Source.Path = "/applications/app1"
		e.Application.Annotations = map[string]string{
			"argocd.argoproj.io/manifest-generate-paths": "/components/app1",
		}
		manifestGeneratePaths := e.GetManifestGeneratePaths()
		want := []string{"components/app1"}
		if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
			t.Errorf("want != manifestGeneratePaths:\n%s", diff)
		}
	})

	t.Run("relative path of ascendant", func(t *testing.T) {
		var e Event
		e.Application.Spec.Source.Path = "/applications/app1"
		e.Application.Annotations = map[string]string{
			"argocd.argoproj.io/manifest-generate-paths": "../manifests1",
		}
		manifestGeneratePaths := e.GetManifestGeneratePaths()
		want := []string{"applications/manifests1"}
		if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
			t.Errorf("want != manifestGeneratePaths:\n%s", diff)
		}
	})

	t.Run("relative path of period", func(t *testing.T) {
		var e Event
		e.Application.Spec.Source.Path = "/applications/app1"
		e.Application.Annotations = map[string]string{
			"argocd.argoproj.io/manifest-generate-paths": ".",
		}
		manifestGeneratePaths := e.GetManifestGeneratePaths()
		want := []string{"applications/app1"}
		if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
			t.Errorf("want != manifestGeneratePaths:\n%s", diff)
		}
	})

	t.Run("multiple paths", func(t *testing.T) {
		var e Event
		e.Application.Spec.Source.Path = "/applications/app1"
		e.Application.Annotations = map[string]string{
			"argocd.argoproj.io/manifest-generate-paths": ".;../manifests1",
		}
		manifestGeneratePaths := e.GetManifestGeneratePaths()
		want := []string{
			"applications/app1",
			"applications/manifests1",
		}
		if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
			t.Errorf("want != manifestGeneratePaths:\n%s", diff)
		}
	})
}
