package notification

import (
	"testing"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/google/go-cmp/cmp"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getManifestGeneratePaths(t *testing.T) {
	t.Run("nil annotation", func(t *testing.T) {
		manifestGeneratePaths := getManifestGeneratePaths(argocdv1alpha1.Application{})
		if manifestGeneratePaths != nil {
			t.Errorf("manifestGeneratePaths wants nil but was %+v", manifestGeneratePaths)
		}
	})

	t.Run("empty annotation", func(t *testing.T) {
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

	t.Run("absolute path", func(t *testing.T) {
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
		want := []string{"components/app1"}
		if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
			t.Errorf("want != manifestGeneratePaths:\n%s", diff)
		}
	})

	t.Run("relative path of ascendant", func(t *testing.T) {
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
		want := []string{"applications/manifests1"}
		if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
			t.Errorf("want != manifestGeneratePaths:\n%s", diff)
		}
	})

	t.Run("relative path of period", func(t *testing.T) {
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
		want := []string{"applications/app1"}
		if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
			t.Errorf("want != manifestGeneratePaths:\n%s", diff)
		}
	})

	t.Run("multiple paths", func(t *testing.T) {
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
			"applications/app1",
			"applications/manifests1",
		}
		if diff := cmp.Diff(want, manifestGeneratePaths); diff != "" {
			t.Errorf("want != manifestGeneratePaths:\n%s", diff)
		}
	})
}
