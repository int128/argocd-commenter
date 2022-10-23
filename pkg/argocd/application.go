package argocd

import (
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
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
