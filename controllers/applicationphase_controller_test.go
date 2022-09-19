package controllers

import (
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Application phase controller", func() {
	const timeout = time.Second * 3
	const interval = time.Millisecond * 250
	appKey := types.NamespacedName{Namespace: "default", Name: "app1"}

	Context("When an application is synced", func() {
		It("Should notify a comment and deployment status", func() {
			By("By creating an application")
			Expect(k8sClient.Create(ctx, &argocdv1alpha1.Application{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "Application",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      appKey.Name,
					Namespace: appKey.Namespace,
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Project: "default",
					Source: argocdv1alpha1.ApplicationSource{
						RepoURL:        "https://github.com/int128/argocd-commenter.git",
						Path:           "test/app1",
						TargetRevision: "main",
					},
					Destination: argocdv1alpha1.ApplicationDestination{
						Server:    "https://kubernetes.default.svc",
						Namespace: "default",
					},
				},
			})).Should(Succeed())

			By("By updating the operation state to running")
			Eventually(func(g Gomega) {
				var app argocdv1alpha1.Application
				g.Expect(k8sClient.Get(ctx, appKey, &app)).Should(Succeed())
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
				g.Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func() int {
				return notificationMock.Comments.CountBy(appKey)
			}, timeout, interval).Should(Equal(1))

			By("By updating the operation state to succeeded")
			Eventually(func(g Gomega) {
				var app argocdv1alpha1.Application
				g.Expect(k8sClient.Get(ctx, appKey, &app)).Should(Succeed())
				app.Status.OperationState.Phase = synccommon.OperationSucceeded
				g.Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func() int {
				return notificationMock.Comments.CountBy(appKey)
			}, timeout, interval).Should(Equal(2))
		})
	})
})
