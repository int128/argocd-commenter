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

var _ = Describe("ApplicationHealth controller", func() {
	const timeout = time.Second * 3
	const interval = time.Millisecond * 250
	var app argocdv1alpha1.Application
	var appKey types.NamespacedName

	BeforeEach(func() {
		app = argocdv1alpha1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Application",
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "fixture-",
				Namespace:    "default",
				Annotations: map[string]string{
					"argocd-commenter.int128.github.io/deployment-url": "https://api.github.com/repos/int128/argocd-commenter/deployments/1234567890",
				},
				Finalizers: []string{
					"resources-finalizer.argocd.argoproj.io",
				},
			},
			Spec: argocdv1alpha1.ApplicationSpec{
				Project: "default",
				Source: argocdv1alpha1.ApplicationSource{
					RepoURL:        "https://github.com/int128/argocd-commenter.git",
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
		appKey = types.NamespacedName{Namespace: app.Namespace, Name: app.Name}
	})

	Context("When an application is deleting", func() {
		It("Should inactivate the deployment status", func() {
			By("By updating the health status to progressing")
			patch := client.MergeFrom(app.DeepCopy())
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
				return notificationMock.DeploymentStatuses.CountBy(appKey)
			}, timeout, interval).Should(Equal(1))

			By("By verifying the finalizer of ApplicationHealth")
			var appHealth argocdcommenterv1.ApplicationHealth
			Eventually(func() error {
				return k8sClient.Get(ctx, appKey, &appHealth)
			}, timeout, interval).Should(Succeed())
			Expect(appHealth.Finalizers).Should(ContainElements(myFinalizerName))

			By("By deleting the ApplicationHealth")
			Expect(k8sClient.Delete(ctx, &appHealth)).Should(Succeed())

			Eventually(func() int {
				return notificationMock.InactivateDeployments.CountBy(appKey)
			}, timeout, interval).Should(Equal(1))
		})
	})
})
