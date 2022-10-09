package controllers

import (
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Application health change controller", func() {
	const timeout = time.Second * 3
	const interval = time.Millisecond * 250
	var app argocdv1alpha1.Application

	BeforeEach(func() {
		app = argocdv1alpha1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Application",
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "fixture-",
				Namespace:    "default",
			},
			Spec: argocdv1alpha1.ApplicationSpec{
				Project: "default",
				Source: argocdv1alpha1.ApplicationSource{
					RepoURL:        "https://github.com/int128/manifests.git",
					Path:           "test",
					TargetRevision: "main",
				},
				Destination: argocdv1alpha1.ApplicationDestination{
					Server:    "https://kubernetes.default.svc",
					Namespace: "default",
				},
			},
		}
		Expect(k8sClient.Create(ctx, &app)).Should(Succeed())
	})

	Context("When an application is healthy", func() {
		It("Should notify a comment once", func() {
			By("By updating the health status to progressing")
			patch := client.MergeFrom(app.DeepCopy())
			app.Status = argocdv1alpha1.ApplicationStatus{
				Health: argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				},
				OperationState: &argocdv1alpha1.OperationState{
					StartedAt: metav1.Now(),
					Operation: argocdv1alpha1.Operation{
						Sync: &argocdv1alpha1.SyncOperation{
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa200",
						},
					},
				},
			}
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			By("By updating the health status to healthy")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.Comments.CountBy(200)
			}, timeout, interval).Should(Equal(1))

			By("By updating the health status to progressing")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.Health.Status = health.HealthStatusProgressing
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			By("By updating the health status to healthy")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Consistently(func() int {
				return githubMock.Comments.CountBy(200)
			}, 100*time.Millisecond).Should(Equal(1))
		})
	})

	Context("When an application is degraded and then healthy", func() {
		It("Should notify a comment for degraded and healthy", func() {
			By("By updating the health status to progressing")
			patch := client.MergeFrom(app.DeepCopy())
			app.Status = argocdv1alpha1.ApplicationStatus{
				Health: argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				},
				OperationState: &argocdv1alpha1.OperationState{
					StartedAt: metav1.Now(),
					Operation: argocdv1alpha1.Operation{
						Sync: &argocdv1alpha1.SyncOperation{
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa201",
						},
					},
				},
			}
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			By("By updating the health status to degraded")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.Health.Status = health.HealthStatusDegraded
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.Comments.CountBy(201)
			}, timeout, interval).Should(Equal(1))

			By("By updating the health status to healthy")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.Comments.CountBy(201)
			}, timeout, interval).Should(Equal(2))
		})
	})
})
