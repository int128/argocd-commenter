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

package controller

import (
	"context"
	"fmt"
	"go/build"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/int128/argocd-commenter/internal/controller/githubmock"
	"github.com/int128/argocd-commenter/internal/github"
	"github.com/int128/argocd-commenter/internal/notification"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	k8sClient    client.Client
	githubServer githubmock.Server
)

var _ = BeforeEach(func() {
	requeueIntervalWhenDeploymentNotFound = 1 * time.Second

	requeueTimeToEvaluateHealthStatusAfterSyncOperation = 0
})

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true),
		func(o *zap.Options) {
			o.TimeEncoder = zapcore.RFC3339NanoTimeEncoder
		}))

	By("Finding the Argo CD Application CRD")
	crdPaths, err := filepath.Glob(filepath.Join(
		build.Default.GOPATH, "pkg", "mod",
		"github.com", "argoproj", "argo-cd", "v2@*", "manifests", "crds", "application-crd.yaml",
	))
	Expect(err).NotTo(HaveOccurred())
	Expect(crdPaths).NotTo(BeEmpty())

	By("Bootstrapping test environment")
	crdPaths = append(crdPaths, filepath.Join("..", "..", "config", "crd", "bases"))
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     crdPaths,
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	ctx, cancel := context.WithCancel(context.TODO())
	DeferCleanup(func() {
		cancel()
		By("Tearing down the test environment")
		Expect(testEnv.Stop()).Should(Succeed())
	})

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = argocdv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = argocdcommenterv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("Creating argocd-cm")
	Expect(k8sClient.Create(ctx, &corev1.ConfigMap{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "argocd-cm",
			Namespace: "default",
		},
		// https://argo-cd.readthedocs.io/en/stable/operator-manual/argocd-cm-yaml/
		Data: map[string]string{
			"url": "https://argocd.example.com",
		},
	})).Should(Succeed())

	By("Setting up the GitHub mock server")
	githubMockServer := httptest.NewServer(&githubServer)
	DeferCleanup(func() {
		By("Shutting down the GitHub mock server")
		githubMockServer.Close()
	})
	GinkgoT().Setenv("GITHUB_TOKEN", "dummy-github-token")
	GinkgoT().Setenv("GITHUB_ENTERPRISE_URL", githubMockServer.URL)
	ghc, err := github.NewClient(ctx)
	Expect(err).NotTo(HaveOccurred())
	nc := notification.NewClient(ghc)

	By("Setting up the controller manager")
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

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

	go func() {
		defer GinkgoRecover()
		By("Starting the controller manager")
		err := k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})
