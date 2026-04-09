package notification

import (
	"testing"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/internal/argocd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilteredSourceRevisionsForHealthComment(t *testing.T) {
	app := argocdv1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-app",
			Annotations: map[string]string{
				argocd.AnnotationExcludeRepoURLs: "https://github.com/org/repo-b",
			},
		},
		Spec: argocdv1alpha1.ApplicationSpec{
			Sources: argocdv1alpha1.ApplicationSources{
				{RepoURL: "https://github.com/org/repo-a", Path: "apps/a"},
				{RepoURL: "https://github.com/org/repo-b", Path: "apps/b"},
			},
		},
		Status: argocdv1alpha1.ApplicationStatus{
			Health: argocdv1alpha1.AppHealthStatus{
				Status: health.HealthStatusDegraded,
			},
			OperationState: &argocdv1alpha1.OperationState{
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revisions: []string{"aaa111", "bbb222"},
					},
				},
			},
			Resources: []argocdv1alpha1.ResourceStatus{
				{
					Namespace: "default",
					Name:      "my-deploy",
					Health: &argocdv1alpha1.HealthStatus{
						Status:  health.HealthStatusDegraded,
						Message: "deployment not available",
					},
				},
			},
		},
	}

	revisions := argocd.GetSourceRevisions(app)
	filter := argocd.NewRepoURLFilter(app)
	var count int
	for _, sr := range revisions {
		if filter.Allows(sr.Source.RepoURL) {
			if c := generateCommentOnHealthChanged(app, "https://argocd.example.com", sr); c != nil {
				count++
			}
		}
	}
	if count != 1 {
		t.Errorf("expected 1 comment (repo-b excluded), got %d", count)
	}
}
