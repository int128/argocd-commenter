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

var _ = Describe("Application phase controller", func() {
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
					RepoURL:        "https://github.com/test/phase-deployment.git",
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

	Context("When an application is synced", func() {
		It("Should notify a deployment status", func(ctx context.Context) {
			var deploymentStatus githubmock.DeploymentStatus
			githubServer.AddHandlers(map[string]http.Handler{
				"GET /api/v3/repos/test/phase-deployment/deployments/999100/statuses":  deploymentStatus.ListEndpoint(),
				"POST /api/v3/repos/test/phase-deployment/deployments/999100/statuses": deploymentStatus.CreateEndpoint(),
			})

			By("Updating the deployment annotation")
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/test/phase-deployment/deployments/999100",
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to running")
			app.Status = argocdv1alpha1.ApplicationStatus{
				OperationState: &argocdv1alpha1.OperationState{
					Phase:     synccommon.OperationRunning,
					StartedAt: metav1.Now(),
					Operation: argocdv1alpha1.Operation{
						Sync: &argocdv1alpha1.SyncOperation{
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						},
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return deploymentStatus.CreateCount() }).Should(Equal(1))

			By("Updating the application to succeeded")
			app.Status.OperationState.Phase = synccommon.OperationSucceeded
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return deploymentStatus.CreateCount() }).Should(Equal(2))

			By("Updating the application to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return deploymentStatus.CreateCount() }).Should(Equal(3))

			By("Updating the application to running")
			app.Status.OperationState.Phase = synccommon.OperationRunning
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to succeeded")
			app.Status.OperationState.Phase = synccommon.OperationSucceeded
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Consistently(func() int { return deploymentStatus.CreateCount() }, "100ms").Should(Equal(3))
		}, SpecTimeout(3*time.Second))
	})

	Context("When an application sync operation is failed", func() {
		It("Should notify a deployment status", func(ctx context.Context) {
			var deploymentStatus githubmock.DeploymentStatus
			githubServer.AddHandlers(map[string]http.Handler{
				"GET /api/v3/repos/test/phase-deployment/deployments/999101/statuses":  deploymentStatus.ListEndpoint(),
				"POST /api/v3/repos/test/phase-deployment/deployments/999101/statuses": deploymentStatus.CreateEndpoint(),
			})

			By("Updating the deployment annotation")
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/test/phase-deployment/deployments/999101",
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to running")
			app.Status = argocdv1alpha1.ApplicationStatus{
				OperationState: &argocdv1alpha1.OperationState{
					Phase:     synccommon.OperationRunning,
					StartedAt: metav1.Now(),
					Operation: argocdv1alpha1.Operation{
						Sync: &argocdv1alpha1.SyncOperation{
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						},
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return deploymentStatus.CreateCount() }).Should(Equal(1))

			By("Updating the application to failed")
			app.Status.OperationState.Phase = synccommon.OperationFailed
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return deploymentStatus.CreateCount() }).Should(Equal(2))
		}, SpecTimeout(3*time.Second))
	})

	Context("When an application was synced before the deployment annotation is updated", func() {
		It("Should skip the notification", func(ctx context.Context) {
			var deploymentStatus githubmock.DeploymentStatus
			githubServer.AddHandlers(map[string]http.Handler{
				"GET /api/v3/repos/test/phase-deployment/deployments/999102/statuses":  deploymentStatus.ListEndpoint(),
				"POST /api/v3/repos/test/phase-deployment/deployments/999102/statuses": deploymentStatus.CreateEndpoint(),
			})

			By("Updating the deployment annotation")
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/test/phase-deployment/deployments/999999",
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the application to succeeded")
			app.Status = argocdv1alpha1.ApplicationStatus{
				OperationState: &argocdv1alpha1.OperationState{
					Phase:     synccommon.OperationSucceeded,
					StartedAt: metav1.Now(),
					Operation: argocdv1alpha1.Operation{
						Sync: &argocdv1alpha1.SyncOperation{
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						},
					},
				},
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			By("Updating the deployment annotation")
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/test/phase-deployment/deployments/999102",
			}
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			// this test depends on requeueIntervalWhenDeploymentNotFound and takes longer time
			Eventually(func() int { return deploymentStatus.CreateCount() }, 3*time.Second).Should(Equal(1))
		}, SpecTimeout(5*time.Second))
	})
})
