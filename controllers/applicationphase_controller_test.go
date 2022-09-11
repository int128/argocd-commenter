package controllers

import (
	"context"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Application phase controller", func() {
	Context("When an application is created", func() {
		It("Should succeed", func() {
			ctx := context.TODO()

			By("By creating an application")
			app := &argocdv1alpha1.Application{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "Application",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app1",
					Namespace: "default",
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Project: "default",
					Source: argocdv1alpha1.ApplicationSource{
						RepoURL:        "https://github.com/int128/argocd-commenter.git",
						Path:           "test/app1",
						TargetRevision: "main",
					},
					Destination: argocdv1alpha1.ApplicationDestination{
						Server:    "https://kubernetes.default.svc",
						Namespace: "default",
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).Should(Succeed())
		})
	})
})
