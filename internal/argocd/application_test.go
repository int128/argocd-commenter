package argocd

import (
	"testing"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
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
