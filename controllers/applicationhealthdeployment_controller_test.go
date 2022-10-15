package controllers

import (
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Application health deployment controller", func() {
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
		It("Should notify a deployment status once", func() {
			By("By updating the health status to progressing")
			patch := client.MergeFrom(app.DeepCopy())
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/manifests/deployments/999300",
			}
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
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			By("By updating the health status to healthy")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999300)
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
				return githubMock.DeploymentStatuses.CountBy(999300)
			}, 100*time.Millisecond).Should(Equal(1))
		})
	})

	Context("When an application is degraded and then healthy", func() {
		It("Should notify a deployment status for degraded and healthy", func() {
			By("By updating the health status to progressing")
			patch := client.MergeFrom(app.DeepCopy())
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/manifests/deployments/999301",
			}
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
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			By("By updating the health status to degraded")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.Health.Status = health.HealthStatusDegraded
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999301)
			}, timeout, interval).Should(Equal(1))

			By("By updating the health status to healthy")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999301)
			}, timeout, interval).Should(Equal(2))
		})
	})

	Context("When an application is healthy but deployment is still old", func() {
		It("Should notify a deployment status when deployment is valid", func() {
			requeueIntervalWhenDeploymentNotFound = 1 * time.Second

			By("By updating the health status to progressing")
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/manifests/deployments/999999",
			}
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
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("By updating the health status to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("By updating the annotation")
			time.Sleep(1 * time.Second)
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/manifests/deployments/999302",
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999302)
			}, timeout, interval).Should(Equal(1))

			Eventually(func() string {
				var appHealth argocdcommenterv1.ApplicationHealth
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: app.Namespace, Name: app.Name}, &appHealth)).Should(Succeed())
				return appHealth.Status.LastHealthyDeploymentURL
			}, timeout, interval).Should(Equal("https://api.github.com/repos/int128/manifests/deployments/999302"))
		})
	})
})