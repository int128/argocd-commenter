package controller

import (
	"context"
	"net/http"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/int128/argocd-commenter/internal/controller/githubmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Comment", func() {
	var app argocdv1alpha1.Application
	var createComment githubmock.CreateComment

	BeforeEach(func(ctx context.Context) {
		By("Setting up a comment endpoint")
		createComment.Reset()
		githubServer.Route(map[string]http.Handler{
			"GET /api/v3/repos/owner/repo-comment/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101/pulls": githubmock.ListPullRequestsWithCommit(101),
			"GET /api/v3/repos/owner/repo-comment/pulls/101/files":                                        githubmock.ListPullRequestFiles(),
			"POST /api/v3/repos/owner/repo-comment/issues/101/comments":                                   &createComment,
		})

		By("Creating an application")
		app = argocdv1alpha1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Application",
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "fixture-phase-comment-",
				Namespace:    "default",
			},
			Spec: argocdv1alpha1.ApplicationSpec{
				Project: "default",
				Source: &argocdv1alpha1.ApplicationSource{
					RepoURL:        "https://github.com/owner/repo-comment.git",
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

	Context("When the application is synced", func() {
		BeforeEach(func(ctx context.Context) {
			By("Updating the application to running")
			startedAt := metav1.Now()
			app.Status.OperationState = &argocdv1alpha1.OperationState{
				Phase:     synccommon.OperationRunning,
				StartedAt: startedAt,
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101",
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return createComment.Count() }).Should(Equal(1))

			By("Updating the application to succeeded")
			finishedAt := metav1.Now()
			app.Status.OperationState = &argocdv1alpha1.OperationState{
				Phase:      synccommon.OperationSucceeded,
				StartedAt:  startedAt,
				FinishedAt: &finishedAt,
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101",
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return createComment.Count() }).Should(Equal(2))
		})

		Context("When the application is healthy", func() {
			It("Should create comments", func(ctx context.Context) {
				By("Updating the application to progressing")
				app.Status.Health = argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				}
				Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

				By("Updating the application to healthy")
				app.Status.Health = argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusHealthy,
				}
				Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
				Eventually(func() int { return createComment.Count() }).Should(Equal(3))
			}, SpecTimeout(3*time.Second))

			It("Should create healthy comment once", func(ctx context.Context) {
				By("Updating the application to progressing")
				app.Status.Health = argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				}
				Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

				By("Updating the application to healthy")
				app.Status.Health = argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusHealthy,
				}
				Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
				Eventually(func() int { return createComment.Count() }).Should(Equal(3))

				By("Updating the application to progressing")
				app.Status.Health = argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				}
				Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

				By("Updating the application to healthy")
				app.Status.Health = argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusHealthy,
				}
				Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
				Consistently(func() int { return createComment.Count() }, 100*time.Millisecond).Should(Equal(3))
			}, SpecTimeout(3*time.Second))
		})

		Context("When the application is degraded", func() {
			It("Should create comments", func(ctx context.Context) {
				By("Updating the application to progressing")
				app.Status.Health = argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				}
				Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

				By("Updating the application to degraded")
				app.Status.Health = argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusDegraded,
				}
				Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
				Eventually(func() int { return createComment.Count() }).Should(Equal(3))
			}, SpecTimeout(3*time.Second))
		})
	})

	Context("When the sync operation is failed", func() {
		It("Should create comments", func(ctx context.Context) {
			By("Updating the application to running")
			app.Status.OperationState = &argocdv1alpha1.OperationState{
				Phase:     synccommon.OperationRunning,
				StartedAt: metav1.Now(),
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101",
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return createComment.Count() }).Should(Equal(1))

			By("Updating the application to retrying")
			startedAt := metav1.Now()
			app.Status.OperationState = &argocdv1alpha1.OperationState{
				Phase:      synccommon.OperationRunning,
				StartedAt:  startedAt,
				RetryCount: 1,
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101",
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Consistently(func() int { return createComment.Count() }, 100*time.Millisecond).Should(Equal(1))

			By("Updating the application to failed")
			finishedAt := metav1.Now()
			app.Status.OperationState = &argocdv1alpha1.OperationState{
				Phase:      synccommon.OperationFailed,
				StartedAt:  startedAt,
				FinishedAt: &finishedAt,
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101",
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return createComment.Count() }).Should(Equal(2))
		}, SpecTimeout(3*time.Second))
	})
})
