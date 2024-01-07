package controller

import (
	"context"
	"net/http"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
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
					RepoURL:        "https://github.com/test/phase-comment.git",
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
		It("Should notify a comment", func(ctx context.Context) {
			githubMock.AddHandlers(map[string]http.HandlerFunc{
				"GET /api/v3/repos/test/phase-comment/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa100/pulls": githubmock.ListPullRequestsWithCommit(100),
				"GET /api/v3/repos/test/phase-comment/pulls/100/files":                                        githubmock.ListFiles(),
				"POST /api/v3/repos/test/phase-comment/issues/100/comments":                                   githubMock.CreateComment(100),
			})

			By("Updating the application to running")
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
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return githubMock.Comments.CountBy(100) }).Should(Equal(1))

			By("Updating the application to succeeded")
			app.Status.OperationState.Phase = synccommon.OperationSucceeded
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return githubMock.Comments.CountBy(100) }).Should(Equal(2))
		}, SpecTimeout(3*time.Second))
	})

	Context("When an application sync operation is failed", func() {
		It("Should notify a comment", func(ctx context.Context) {
			githubMock.AddHandlers(map[string]http.HandlerFunc{
				"GET /api/v3/repos/test/phase-comment/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa101/pulls": githubmock.ListPullRequestsWithCommit(101),
				"GET /api/v3/repos/test/phase-comment/pulls/101/files":                                        githubmock.ListFiles(),
				"POST /api/v3/repos/test/phase-comment/issues/101/comments":                                   githubMock.CreateComment(101),
			})

			By("Updating the application to running")
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
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return githubMock.Comments.CountBy(101) }).Should(Equal(1))

			By("Updating the application to failed")
			app.Status.OperationState.Phase = synccommon.OperationFailed
			Expect(k8sClient.Update(ctx, &app)).Should(Succeed())
			Eventually(func() int { return githubMock.Comments.CountBy(101) }).Should(Equal(2))
		}, SpecTimeout(3*time.Second))
	})
})
