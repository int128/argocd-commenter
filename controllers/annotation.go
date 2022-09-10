package controllers

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func patchAnnotation(ctx context.Context, c client.Client, app *argocdv1alpha1.Application, f func(map[string]string)) error {
	logger := log.FromContext(ctx)

	patch := client.MergeFrom(app.DeepCopy())
	if app.Annotations == nil {
		app.Annotations = make(map[string]string)
	}
	f(app.Annotations)
	logger.Info("apply a patch", "patch", patch)
	return c.Patch(ctx, app, patch)
}
