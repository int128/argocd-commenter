package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type options struct {
	appNames []string
	timeout  time.Duration
	expected ApplicationStatus
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	var o options
	flag.StringVar(&o.expected.Revision, "revision", "", "Expected revision")
	flag.StringVar(&o.expected.Sync, "sync", "Synced", "Expected sync status")
	flag.StringVar(&o.expected.Operation, "operation", "Succeeded", "Expected operation status")
	flag.StringVar(&o.expected.Health, "health", "Healthy", "Expected health status")
	flag.DurationVar(&o.timeout, "timeout", 1*time.Minute, "Timeout")
	flag.Parse()
	o.appNames = flag.Args()
	if err := run(context.Background(), o); err != nil {
		log.Fatalf("error: %s", err)
	}
}

func run(ctx context.Context, o options) error {
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), nil).ClientConfig()
	if err != nil {
		return fmt.Errorf("could not load the config: %w", err)
	}
	if err := argocdv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("could not add to scheme: %w", err)
	}
	k8sClient, err := client.New(cfg, client.Options{})
	if err != nil {
		return fmt.Errorf("could not create a Kubernetes client: %w", err)
	}
	log.Printf("Connected to Kubernetes cluster at %s", cfg.Host)

	for {
		ok, err := checkApplicationStatus(ctx, k8sClient, o)
		if err != nil {
			return fmt.Errorf("check: %w", err)
		}
		if ok {
			return nil
		}
		log.Printf("Retry after 5s")
		time.Sleep(5 * time.Second)
	}
}

func checkApplicationStatus(ctx context.Context, k8sClient client.Client, o options) (bool, error) {
	ok := true
	for _, appName := range o.appNames {
		key := types.NamespacedName{Namespace: "argocd", Name: appName}
		actualStatus, err := getApplicationStatus(ctx, k8sClient, key)
		if err != nil {
			return false, fmt.Errorf("could not get status of application %s: %w", key, err)
		}
		if diff := cmp.Diff(o.expected, actualStatus); diff != "" {
			ok = false
			log.Printf("Application %s status is not expected:\n%s", key, diff)
			continue
		}
		log.Printf("Application %s status is expected", key)
	}
	return ok, nil
}

type ApplicationStatus struct {
	Revision  string
	Sync      string
	Operation string
	Health    string
}

func getApplicationStatus(ctx context.Context, k8sClient client.Client, key types.NamespacedName) (ApplicationStatus, error) {
	var app argocdv1alpha1.Application
	if err := k8sClient.Get(ctx, key, &app); err != nil {
		return ApplicationStatus{}, fmt.Errorf("get: %w", err)
	}
	s := ApplicationStatus{
		Revision: app.Status.Sync.Revision,
		Sync:     string(app.Status.Sync.Status),
		Health:   string(app.Status.Health.Status),
	}
	if app.Status.OperationState != nil {
		s.Operation = string(app.Status.OperationState.Phase)
	}
	return s, nil
}
