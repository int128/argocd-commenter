package controllers

import (
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/google/go-github/v47/github"
	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	})

	Context("When an application is healthy", func() {
		It("Should notify a deployment status once", func() {
			githubMock.DeploymentStatuses.SetResponse(999300, []*github.DeploymentStatus{})

			By("Creating the GitHubDeployment")
			ghd := argocdcommenterv1.GitHubDeployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argocdcommenter.int128.github.io/v1",
					Kind:       "GitHubDeployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: app.Namespace,
					Name:      app.Name,
				},
				Spec: argocdcommenterv1.GitHubDeploymentSpec{
					DeploymentURL: "https://api.github.com/repos/int128/manifests/deployments/999300",
				},
			}
			Expect(k8sClient.Create(ctx, &ghd)).Should(Succeed())

			By("Updating the application to degraded")
			app.Status.Health.Status = health.HealthStatusDegraded
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return githubMock.DeploymentStatuses.CountBy(999300) }, timeout, interval).Should(Equal(1))

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return githubMock.DeploymentStatuses.CountBy(999300) }, timeout, interval).Should(Equal(2))

			By("Updating the application to progressing")
			app.Status.Health.Status = health.HealthStatusProgressing
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to healthy, again")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Consistently(func() int { return githubMock.DeploymentStatuses.CountBy(999300) }, "100ms").Should(Equal(2))
		})
	})

	Context("When the deployment annotation is updated and then the application becomes healthy", func() {
		It("Should notify a deployment status", func() {
			githubMock.DeploymentStatuses.SetResponse(999301, []*github.DeploymentStatus{})

			By("Creating the GitHubDeployment")
			ghd := argocdcommenterv1.GitHubDeployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argocdcommenter.int128.github.io/v1",
					Kind:       "GitHubDeployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: app.Namespace,
					Name:      app.Name,
				},
				Spec: argocdcommenterv1.GitHubDeploymentSpec{
					DeploymentURL: "https://api.github.com/repos/int128/manifests/deployments/999999",
				},
			}
			Expect(k8sClient.Create(ctx, &ghd)).Should(Succeed())

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the deployment URL")
			ghd.Spec.DeploymentURL = "https://api.github.com/repos/int128/manifests/deployments/999301"
			Expect(k8sClient.Update(ctx, &ghd)).Should(Succeed())
			Consistently(func() int { return githubMock.DeploymentStatuses.CountBy(999301) }, "100ms").Should(BeZero())

			By("Updating the application to progressing")
			app.Status.Health.Status = health.HealthStatusProgressing
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return githubMock.DeploymentStatuses.CountBy(999301) }, timeout, interval).Should(Equal(1))
		})
	})

	Context("When an application became healthy before the deployment annotation is updated", func() {
		It("Should notify a deployment status when the deployment annotation is valid", func() {
			githubMock.DeploymentStatuses.SetResponse(999302, []*github.DeploymentStatus{})

			By("Creating the GitHubDeployment")
			ghd := argocdcommenterv1.GitHubDeployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argocdcommenter.int128.github.io/v1",
					Kind:       "GitHubDeployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: app.Namespace,
					Name:      app.Name,
				},
				Spec: argocdcommenterv1.GitHubDeploymentSpec{
					DeploymentURL: "https://api.github.com/repos/int128/manifests/deployments/999999",
				},
			}
			Expect(k8sClient.Create(ctx, &ghd)).Should(Succeed())

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Consistently(func() int { return githubMock.DeploymentStatuses.CountBy(999302) }, "100ms").Should(BeZero())

			By("Updating the deployment URL")
			ghd.Spec.DeploymentURL = "https://api.github.com/repos/int128/manifests/deployments/999302"
			Expect(k8sClient.Update(ctx, &ghd)).Should(Succeed())
			Eventually(func() int { return githubMock.DeploymentStatuses.CountBy(999302) }, timeout, interval).Should(Equal(1))

			By("Deleting the old deployment")
			githubMock.DeploymentStatuses.SetResponse(999302, nil)
			By("Creating a new deployment")
			githubMock.DeploymentStatuses.SetResponse(999303, []*github.DeploymentStatus{})

			By("Updating the application to progressing")
			app.Status.Health.Status = health.HealthStatusProgressing
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Consistently(func() int { return githubMock.DeploymentStatuses.CountBy(999303) }, "100ms").Should(BeZero())

			By("Updating the deployment URL")
			ghd.Spec.DeploymentURL = "https://api.github.com/repos/int128/manifests/deployments/999303"
			Expect(k8sClient.Update(ctx, &ghd)).Should(Succeed())
			Eventually(func() int { return githubMock.DeploymentStatuses.CountBy(999303) }, timeout, interval).Should(Equal(1))
			Expect(githubMock.DeploymentStatuses.CountBy(999302)).Should(Equal(1))
		})
	})
})
