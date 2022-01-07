package controllers

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func patchAnnotation(ctx context.Context, c client.Client, a argocdv1alpha1.Application, f func(map[string]string)) error {
	logger := log.FromContext(ctx)

	var patch unstructured.Unstructured
	patch.SetGroupVersionKind(a.GroupVersionKind())
	patch.SetNamespace(a.Namespace)
	patch.SetName(a.Name)
	annotations := a.DeepCopy().Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	f(annotations)
	patch.SetAnnotations(annotations)

	logger.Info("apply a patch", "annotations", annotations)
	return c.Patch(ctx, &patch, client.Apply, &client.PatchOptions{FieldManager: "argocd-commenter"})
}
