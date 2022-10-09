package controllers

import (
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
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
		It("Should notify a comment and deployment status", func() {
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
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa100",
						},
					},
				},
			}
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.Comments.CountBy(100)
			}, timeout, interval).Should(Equal(1))
			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999100)
			}, timeout, interval).Should(Equal(1))

			By("By updating the operation state to succeeded")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.OperationState.Phase = synccommon.OperationSucceeded
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.Comments.CountBy(100)
			}, timeout, interval).Should(Equal(2))
			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999100)
			}, timeout, interval).Should(Equal(2))
		})
	})

	Context("When an application sync operation is failed", func() {
		It("Should notify a comment and deployment status", func() {
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
							Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101",
						},
					},
				},
			}
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.Comments.CountBy(101)
			}, timeout, interval).Should(Equal(1))
			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999101)
			}, timeout, interval).Should(Equal(1))

			By("By updating the operation state to failed")
			patch = client.MergeFrom(app.DeepCopy())
			app.Status.OperationState.Phase = synccommon.OperationFailed
			Expect(k8sClient.Patch(ctx, &app, patch)).Should(Succeed())

			Eventually(func() int {
				return githubMock.Comments.CountBy(101)
			}, timeout, interval).Should(Equal(2))
			Eventually(func() int {
				return githubMock.DeploymentStatuses.CountBy(999101)
			}, timeout, interval).Should(Equal(2))
		})
	})
})
