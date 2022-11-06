/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"go/build"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/pkg/github"
	"github.com/int128/argocd-commenter/pkg/notification"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	k8sClient  client.Client
	testEnv    *envtest.Environment
	ctx        context.Context
	cancel     context.CancelFunc
	githubMock GithubMock
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true),
		func(o *zap.Options) {
			o.TimeEncoder = zapcore.RFC3339NanoTimeEncoder
		}))
	ctx, cancel = context.WithCancel(context.TODO())

	By("find the CRD of Argo CD Application resource in Go module")
	crdPaths, err := filepath.Glob(filepath.Join(
		build.Default.GOPATH, "pkg", "mod",
		"github.com", "argoproj", "argo-cd", "v2@*", "manifests", "crds", "application-crd.yaml",
	))
	Expect(err).NotTo(HaveOccurred())
	Expect(crdPaths).NotTo(BeEmpty())

	By("bootstrapping test environment")
	crdPaths = append(crdPaths, filepath.Join("..", "config", "crd", "bases"))
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     crdPaths,
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = argocdv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = argocdcommenterv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	githubMockServer := httptest.NewServer(githubMock.NewHandler())
	ctx = context.WithValue(ctx, oauth2.HTTPClient, githubMockServer.Client())
	GinkgoT().Setenv("GITHUB_TOKEN", "dummy-github-token")
	GinkgoT().Setenv("GITHUB_ENTERPRISE_URL", githubMockServer.URL)
	ghc, err := github.NewClient(ctx)
	Expect(err).NotTo(HaveOccurred())
	nc := notification.NewClient(ghc)

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&ApplicationHealthReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// comment controllers
	err = (&ApplicationPhaseCommentReconciler{
		Client:       k8sManager.GetClient(),
		Scheme:       k8sManager.GetScheme(),
		Notification: nc,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())
	err = (&ApplicationHealthCommentReconciler{
		Client:       k8sManager.GetClient(),
		Scheme:       k8sManager.GetScheme(),
		Notification: nc,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// deployment controllers
	err = (&ApplicationPhaseDeploymentReconciler{
		Client:       k8sManager.GetClient(),
		Scheme:       k8sManager.GetScheme(),
		Notification: nc,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())
	err = (&ApplicationHealthDeploymentReconciler{
		Client:       k8sManager.GetClient(),
		Scheme:       k8sManager.GetScheme(),
		Notification: nc,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())
	err = (&ApplicationDeletionDeploymentReconciler{
		Client:       k8sManager.GetClient(),
		Scheme:       k8sManager.GetScheme(),
		Notification: nc,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	requeueIntervalWhenDeploymentNotFound = 1 * time.Second

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
