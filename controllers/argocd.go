package controllers

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// findArgoCDURL returns the URL of Argo CD if available.
// See https://github.com/argoproj/argo-cd/blob/master/docs/operator-manual/argocd-cm.yaml
func findArgoCDURL(ctx context.Context, c client.Client, namespace string) (string, error) {
	var cm corev1.ConfigMap
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "argocd-cm"}, &cm)
	if err != nil {
		return "", err
	}
	return cm.Data["url"], nil
}

func getCurrentDeployedRevision(a argocdv1alpha1.Application) string {
	if a.Status.OperationState == nil {
		return ""
	}
	if a.Status.OperationState.Operation.Sync == nil {
		return ""
	}
	return a.Status.OperationState.Operation.Sync.Revision
}

func getDeploymentURL(a argocdv1alpha1.Application) string {
	return a.Annotations["argocd-commenter.int128.github.io/deployment-url"]
}
