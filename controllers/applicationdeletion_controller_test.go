package controllers

import (
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Application health controller with deployment", func() {
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
		It("Should notify a deployment status once", func() {
			By("By deleting the application")
			Expect(k8sClient.Delete(ctx, &app)).Should(Succeed())

			Eventually(func() int {
				return notificationMock.DeploymentStatuses.CountBy(appKey)
			}, timeout, interval).Should(Equal(1))
		})
	})
})
