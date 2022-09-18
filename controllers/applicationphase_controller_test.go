package controllers

import (
	"context"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Application phase controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When an application is synced", func() {
		It("Should notify events", func() {
			ctx := context.TODO()
			app1Key := types.NamespacedName{Namespace: "default", Name: "app1"}

			By("By creating an application")
			Expect(k8sClient.Create(ctx, &argocdv1alpha1.Application{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "Application",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      app1Key.Name,
					Namespace: app1Key.Namespace,
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

			By("By getting the application")
			var app1 argocdv1alpha1.Application
			Eventually(func() error { return k8sClient.Get(ctx, app1Key, &app1) }, timeout, interval).Should(Succeed())

			By("By updating the operation state to running")
			app1.Status = argocdv1alpha1.ApplicationStatus{
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
			Expect(k8sClient.Update(ctx, &app1)).Should(Succeed())
			//TODO: assert mock

			By("By updating the operation state to succeeded")
			app1.Status.OperationState.Phase = synccommon.OperationSucceeded
			Expect(k8sClient.Update(ctx, &app1)).Should(Succeed())
			//TODO: assert mock
		})
	})
})
