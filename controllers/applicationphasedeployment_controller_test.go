package controllers

import (
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/google/go-github/v47/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Application phase controller", func() {
	const timeout = time.Second * 3
	const interval = time.Millisecond * 250
	var app argocdv1alpha1.Application

	BeforeEach(func() {
		By("By creating an application")
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

	Context("When an application is synced", func() {
		It("Should notify a deployment status", func() {
			githubMock.DeploymentStatuses.SetResponse(999100, []*github.DeploymentStatus{})

			By("By updating the operation state to running")
			patch := client.MergeFrom(app.DeepCopy())
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/manifests/deployments/999100",
			}
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
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999100)
			}, timeout, interval).Should(Equal(1))

			By("By updating the operation state to succeeded")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.OperationState.Phase = synccommon.OperationSucceeded
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999100)
			}, timeout, interval).Should(Equal(2))
		})
	})

	Context("When an application sync operation is failed", func() {
		It("Should notify a deployment status", func() {
			githubMock.DeploymentStatuses.SetResponse(999101, []*github.DeploymentStatus{})

			By("By updating the operation state to running")
			patch := client.MergeFrom(app.DeepCopy())
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/manifests/deployments/999101",
			}
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
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999101)
			}, timeout, interval).Should(Equal(1))

			By("By updating the operation state to failed")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.OperationState.Phase = synccommon.OperationFailed
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999101)
			}, timeout, interval).Should(Equal(2))
		})
	})

	Context("When an application is re-synced", func() {
		It("Should skip the notification", func() {
			githubMock.DeploymentStatuses.SetResponse(999102, []*github.DeploymentStatus{})

			By("By updating the operation state to succeeded")
			app.Annotations = map[string]string{
				"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/manifests/deployments/999102",
			}
			app.Status = argocdv1alpha1.ApplicationStatus{
				Health: argocdv1alpha1.HealthStatus{
					Status: health.HealthStatusProgressing,
				},
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

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999102)
			}, timeout, interval).Should(Equal(1))

			By("By updating the health status to healthy")
			app.Status.Health.Status = health.HealthStatusHealthy
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999102)
			}, timeout, interval).Should(Equal(2))

			By("By updating the operation state to running")
			app.Status.OperationState.Phase = synccommon.OperationRunning
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			Consistently(func() int {
				return githubMock.DeploymentStatuses.CountBy(999102)
			}, 100*time.Millisecond).Should(Equal(2))

			By("By updating the operation state to succeeded")
			app.Status.OperationState.Phase = synccommon.OperationSucceeded
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())

			Consistently(func() int {
				return githubMock.DeploymentStatuses.CountBy(999102)
			}, 100*time.Millisecond).Should(Equal(2))
		})
	})
})
