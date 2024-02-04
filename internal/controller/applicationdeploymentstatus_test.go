package controller

import (
	"context"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/google/go-github/v58/github"
	"github.com/int128/argocd-commenter/internal/controller/githubmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Deployment status", func() {
	var app argocdv1alpha1.Application
	var listDeploymentStatus *githubmock.ListDeploymentStatus
	var createDeploymentStatus *githubmock.CreateDeploymentStatus

	BeforeEach(func(ctx context.Context) {
		By("Setting up a deployment status endpoint")
		listDeploymentStatus = &githubmock.ListDeploymentStatus{Response: []*github.DeploymentStatus{}}
		createDeploymentStatus = &githubmock.CreateDeploymentStatus{}
		githubServer.Handle(
			"GET /api/v3/repos/owner/repo-deployment/deployments/101/statuses",
			listDeploymentStatus,
		)
		githubServer.Handle(
			"POST /api/v3/repos/owner/repo-deployment/deployments/101/statuses",
			createDeploymentStatus,
		)

		By("Creating an application")
		app = argocdv1alpha1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Application",
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "fixture-deployment-status-phase-",
				Namespace:    "default",
				Annotations: map[string]string{
					"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/owner/repo-deployment/deployments/101",
				},
			},
			Spec: argocdv1alpha1.ApplicationSpec{
				Project: "default",
				Source: &argocdv1alpha1.ApplicationSource{
					RepoURL:        "https://github.com/owner/repo-deployment.git",
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
			Eventually(func() int { return createDeploymentStatus.Count() }).Should(Equal(1))

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
			Eventually(func() int { return createDeploymentStatus.Count() }).Should(Equal(2))
		})

		Context("When the application is healthy", func() {
			It("Should create deployment statuses", func(ctx context.Context) {
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
				Eventually(func() int { return createDeploymentStatus.Count() }).Should(Equal(3))
			}, SpecTimeout(3*time.Second))

			It("Should not create any deployment status after healthy", func(ctx context.Context) {
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
				Eventually(func() int { return createDeploymentStatus.Count() }).Should(Equal(3))

				// The controller depends on the deployment status to deduplicate the health status.
				listDeploymentStatus.Response = []*github.DeploymentStatus{
					{State: github.String("success")},
				}

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
				Consistently(func() int { return createDeploymentStatus.Count() }, "100ms").Should(Equal(3))

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
				Consistently(func() int { return createDeploymentStatus.Count() }, "100ms").Should(Equal(3))
			}, SpecTimeout(3*time.Second))
		})
	})

	Context("When the sync operation is failed", func() {
		It("Should create deployment statuses", func(ctx context.Context) {
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
			Eventually(func() int { return createDeploymentStatus.Count() }).Should(Equal(1))

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
			Consistently(func() int { return createDeploymentStatus.Count() }, 100*time.Millisecond).Should(Equal(1))

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
			Eventually(func() int { return createDeploymentStatus.Count() }).Should(Equal(2))
		}, SpecTimeout(3*time.Second))
	})
})

var _ = Describe("Deployment status", func() {
	BeforeEach(func() {
		requeueToEvaluateHealthStatusAfterSyncOperation = 0
	})

	Context("When an application was synced before the deployment annotation is updated", func() {
		var app argocdv1alpha1.Application

		BeforeEach(func(ctx context.Context) {
			By("Creating an application")
			app = argocdv1alpha1.Application{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "Application",
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fixture-deployment-status-phase-",
					Namespace:    "default",
					Annotations: map[string]string{
						"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/owner/repo-deployment/deployments/999",
					},
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Project: "default",
					Source: &argocdv1alpha1.ApplicationSource{
						RepoURL:        "https://github.com/owner/repo-deployment.git",
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

		It("Should finally create a deployment status", func(ctx context.Context) {
			By("Setting up a deployment status endpoint")
			listDeploymentStatus := &githubmock.ListDeploymentStatus{Response: []*github.DeploymentStatus{}}
			createDeploymentStatus := &githubmock.CreateDeploymentStatus{}
			githubServer.Handle(
				"GET /api/v3/repos/owner/repo-deployment/deployments/101/statuses",
				listDeploymentStatus,
			)
			githubServer.Handle(
				"POST /api/v3/repos/owner/repo-deployment/deployments/101/statuses",
				createDeploymentStatus,
			)

			By("Updating the application to running")
			startedAt := metav1.Now()
			app.Status.OperationState = &argocdv1alpha1.OperationState{
				Phase:     synccommon.OperationRunning,
				StartedAt: startedAt,
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Consistently(func() int { return createDeploymentStatus.Count() }, 100*time.Millisecond).Should(Equal(0))

			By("Updating the deployment annotation")
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/owner/repo-deployment/deployments/101",
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return createDeploymentStatus.Count() },
				// This depends on requeueIntervalWhenDeploymentNotFound and takes longer time
				2*requeueIntervalWhenDeploymentNotFound,
			).Should(Equal(1))

			By("Updating the application to succeeded")
			finishedAt := metav1.Now()
			app.Status.OperationState = &argocdv1alpha1.OperationState{
				Phase:      synccommon.OperationSucceeded,
				StartedAt:  startedAt,
				FinishedAt: &finishedAt,
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return createDeploymentStatus.Count() }, 3*time.Second).Should(Equal(2))
		}, SpecTimeout(5*time.Second))

		It("Should retry to create a deployment status until timeout", func(ctx context.Context) {
			By("Updating the application to running")
			app.Status.OperationState = &argocdv1alpha1.OperationState{
				Phase:     synccommon.OperationRunning,
				StartedAt: metav1.NewTime(metav1.Now().Add(-requeueTimeoutWhenDeploymentNotFound)),
				Operation: argocdv1alpha1.Operation{
					Sync: &argocdv1alpha1.SyncOperation{
						Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Finding the retry event")
			Eventually(func(g Gomega) {
				var eventList corev1.EventList
				g.Expect(k8sClient.List(ctx, &eventList, crclient.MatchingFields{
					"involvedObject.name": app.Name,
					"reason":              "DeploymentNotFoundRetryTimeout",
				})).Should(Succeed())
				g.Expect(eventList.Items).Should(HaveLen(1))
			})
		})
	})
})
