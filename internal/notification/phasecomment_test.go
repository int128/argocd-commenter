package notification

import (
	"testing"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/internal/argocd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateCommentOnPhaseChanged(t *testing.T) {
	t.Run("generates comment for valid source", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{Name: "my-app"},
			Status: argocdv1alpha1.ApplicationStatus{
				OperationState: &argocdv1alpha1.OperationState{
					Phase: synccommon.OperationSucceeded,
				},
			},
		}
		sr := argocd.SourceRevision{
			Source:   argocdv1alpha1.ApplicationSource{RepoURL: "https://github.com/org/repo-a"},
			Revision: "abc123",
		}
		comment := generateCommentOnPhaseChanged(app, "https://argocd.example.com", sr)
		if comment == nil {
			t.Fatal("expected comment to be generated")
		}
		if comment.Body == "" {
			t.Error("expected non-empty body")
		}
	})

	t.Run("returns nil for non-github source", func(t *testing.T) {
		app := argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{Name: "my-app"},
			Status: argocdv1alpha1.ApplicationStatus{
				OperationState: &argocdv1alpha1.OperationState{
					Phase: synccommon.OperationSucceeded,
				},
			},
		}
		sr := argocd.SourceRevision{
			Source:   argocdv1alpha1.ApplicationSource{RepoURL: "568871113537.dkr.ecr.eu-west-1.amazonaws.com/helm-charts"},
			Revision: "2.30.0",
		}
		comment := generateCommentOnPhaseChanged(app, "https://argocd.example.com", sr)
		if comment != nil {
			t.Error("expected nil comment for non-github URL")
		}
	})
}

func TestFilteredSourceRevisionsForPhaseComment(t *testing.T) {
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
			OperationState: &argocdv1alpha1.OperationState{
				Phase: synccommon.OperationSucceeded,
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revisions: []string{"aaa111", "bbb222"},
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
			if c := generateCommentOnPhaseChanged(app, "https://argocd.example.com", sr); c != nil {
				count++
			}
		}
	}
	if count != 1 {
		t.Errorf("expected 1 comment (repo-b excluded), got %d", count)
	}
}
