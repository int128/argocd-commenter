package controller

import (
	"context"
	"net/http"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/int128/argocd-commenter/internal/controller/githubmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Application health comment controller", func() {
	var app argocdv1alpha1.Application

	BeforeEach(func(ctx context.Context) {
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
				Source: &argocdv1alpha1.ApplicationSource{
					RepoURL:        "https://github.com/test/health-comment.git",
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
		It("Should notify a comment once", func(ctx context.Context) {
			var comment githubmock.Comment
			githubServer.AddHandlers(map[string]http.Handler{
				"GET /api/v3/repos/test/health-comment/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101/pulls": githubmock.ListPullRequestsWithCommit(101),
				"GET /api/v3/repos/test/health-comment/pulls/101/files":                                        githubmock.ListFiles(),
				"POST /api/v3/repos/test/health-comment/issues/101/comments":                                   comment.CreateEndpoint(),
			})

			By("Updating the application to progressing")
			app.Status = argocdv1alpha1.ApplicationStatus{
				Health: argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				},
				OperationState: &argocdv1alpha1.OperationState{
					StartedAt: metav1.Now(),
					Operation: argocdv1alpha1.Operation{
						Sync: &argocdv1alpha1.SyncOperation{
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101",
						},
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return comment.CreateCount() }).Should(Equal(1))

			By("Updating the application to progressing")
			app.Status.Health.Status = health.HealthStatusProgressing
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Consistently(func() int { return comment.CreateCount() }, 100*time.Millisecond).Should(Equal(1))
		}, SpecTimeout(3*time.Second))
	})

	Context("When an application is degraded and then healthy", func() {
		It("Should notify a comment for degraded and healthy", func(ctx context.Context) {
			var comment githubmock.Comment
			githubServer.AddHandlers(map[string]http.Handler{
				"GET /api/v3/repos/test/health-comment/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa102/pulls": githubmock.ListPullRequestsWithCommit(102),
				"GET /api/v3/repos/test/health-comment/pulls/102/files":                                        githubmock.ListFiles(),
				"POST /api/v3/repos/test/health-comment/issues/102/comments":                                   comment.CreateEndpoint(),
			})

			By("Updating the application to progressing")
			app.Status = argocdv1alpha1.ApplicationStatus{
				Health: argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				},
				OperationState: &argocdv1alpha1.OperationState{
					StartedAt: metav1.Now(),
					Operation: argocdv1alpha1.Operation{
						Sync: &argocdv1alpha1.SyncOperation{
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa102",
						},
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to degraded")
			app.Status.Health.Status = health.HealthStatusDegraded
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return comment.CreateCount() }).Should(Equal(1))

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return comment.CreateCount() }).Should(Equal(2))
		}, SpecTimeout(3*time.Second))
	})
})
