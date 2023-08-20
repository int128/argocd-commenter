package argocd

import (
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetDeployedRevision returns the last synced revision
func GetDeployedRevision(a argocdv1alpha1.Application) string {
	if a.Status.OperationState == nil {
		return ""
	}
	if a.Status.OperationState.Operation.Sync == nil {
		return ""
	}
	return a.Status.OperationState.Operation.Sync.Revision
}

type SourceRevision struct {
	Source   argocdv1alpha1.ApplicationSource
	Revision string
}

func GetSourceRevisions(app argocdv1alpha1.Application) []SourceRevision {
	if app.Status.OperationState == nil {
		return nil
	}
	sources := app.Spec.GetSources()
	revisions := app.Status.OperationState.Operation.Sync.Revisions
	if revisions == nil {
		revisions = []string{app.Status.OperationState.Operation.Sync.Revision}
	}
	size := min(len(sources), len(revisions))

	sourceRevisions := make([]SourceRevision, size)
	for i := 0; i < size; i++ {
		sourceRevisions[i] = SourceRevision{
			Source:   sources[i],
			Revision: revisions[i],
		}
	}
	return sourceRevisions
}

// GetDeploymentURL returns the deployment URL in annotations
func GetDeploymentURL(a argocdv1alpha1.Application) string {
	if a.Annotations == nil {
		return ""
	}
	return a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
}

func GetOperationPhase(a argocdv1alpha1.Application) synccommon.OperationPhase {
	if a.Status.OperationState == nil {
		return ""
	}
	return a.Status.OperationState.Phase
}

// GetLastOperationAt returns OperationState.FinishedAt, OperationState.StartedAt or zero Time.
func GetLastOperationAt(a argocdv1alpha1.Application) metav1.Time {
	if a.Status.OperationState == nil {
		return metav1.Time{}
	}
	if a.Status.OperationState.FinishedAt != nil {
		return *a.Status.OperationState.FinishedAt
	}
	return a.Status.OperationState.StartedAt
}
