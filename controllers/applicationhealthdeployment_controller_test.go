package controllers

import (
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Application health deployment controller", func() {
	const timeout = time.Second * 3
	const interval = time.Millisecond * 250
	appKey := types.NamespacedName{Namespace: "default", Name: "app3"}

	Context("When an application is healthy", func() {
		It("Should notify a comment", func() {
			By("By creating an application")
			Expect(k8sClient.Create(ctx, &argocdv1alpha1.Application{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "Application",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      appKey.Name,
					Namespace: appKey.Namespace,
					Annotations: map[string]string{
						"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/argocd-commenter/deployments/1234567890",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Project: "default",
					Source: argocdv1alpha1.ApplicationSource{
						RepoURL:        "https://github.com/int128/argocd-commenter.git",
						Path:           "test",
						TargetRevision: "main",
					},
					Destination: argocdv1alpha1.ApplicationDestination{
						Server:    "https://kubernetes.default.svc",
						Namespace: "default",
					},
				},
			})).Should(Succeed())

			By("By updating the health status to progressing")
			Eventually(func(g Gomega) {
				var app argocdv1alpha1.Application
				g.Expect(k8sClient.Get(ctx, appKey, &app)).Should(Succeed())
				app.Status = argocdv1alpha1.ApplicationStatus{
					Health: argocdv1alpha1.HealthStatus{
						Status: health.HealthStatusProgressing,
					},
					OperationState: &argocdv1alpha1.OperationState{
						StartedAt: metav1.Now(),
						Operation: argocdv1alpha1.Operation{
							Sync: &argocdv1alpha1.SyncOperation{
								Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							},
						},
					},
				}
				g.Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			By("By updating the health status to healthy")
			Eventually(func(g Gomega) {
				var app argocdv1alpha1.Application
				g.Expect(k8sClient.Get(ctx, appKey, &app)).Should(Succeed())
				app.Status.Health.Status = health.HealthStatusHealthy
				g.Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func() int {
				return notificationMock.DeploymentStatuses.CountBy(appKey)
			}, timeout, interval).Should(Equal(1))
		})

		It("Should not notify a comment after healthy", func() {
			By("By updating the health status to progressing")
			Eventually(func(g Gomega) {
				var app argocdv1alpha1.Application
				g.Expect(k8sClient.Get(ctx, appKey, &app)).Should(Succeed())
				app.Status.Health.Status = health.HealthStatusHealthy
				g.Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			By("By updating the health status to healthy")
			Eventually(func(g Gomega) {
				var app argocdv1alpha1.Application
				g.Expect(k8sClient.Get(ctx, appKey, &app)).Should(Succeed())
				app.Status.Health.Status = health.HealthStatusHealthy
				g.Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Consistently(func() int {
				return notificationMock.DeploymentStatuses.CountBy(appKey)
			}, 1*time.Second, interval).Should(Equal(1))
		})
	})
})
