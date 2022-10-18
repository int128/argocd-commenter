package argocd

import (
	"context"
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FindExternalURL returns the URL of Argo CD if available.
// See https://github.com/argoproj/argo-cd/blob/master/docs/operator-manual/argocd-cm.yaml
func FindExternalURL(ctx context.Context, c client.Client, namespace string) (string, error) {
	var cm v1.ConfigMap
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "argocd-cm"}, &cm)
	if err != nil {
		return "", fmt.Errorf("unable to get Argo CD ConfigMap: %w", err)
	}
	url, ok := cm.Data["url"]
	if !ok {
		return "", fmt.Errorf("url is not set in ConfigMap %s", cm.Name)
	}
	return url, nil
}
