package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/watch"
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
	k8sClient, err := client.NewWithWatch(cfg, client.Options{})
	if err != nil {
		return fmt.Errorf("could not create a Kubernetes client: %w", err)
	}
	log.Printf("Connected to Kubernetes cluster at %s", cfg.Host)

	var fns []func() error
	for _, appName := range o.appNames {
		fns = append(fns, func() error {
			if err := watchApplicationStatus(ctx, k8sClient, appName, o.expected); err != nil {
				return fmt.Errorf("watchApplicationStatus: %w", err)
			}
			return nil
		})
	}
	return errors.AggregateGoroutines(fns...)
}

func watchApplicationStatus(ctx context.Context, c client.WithWatch, appName string, expected ApplicationStatus) error {
	var apps argocdv1alpha1.ApplicationList
	w, err := c.Watch(ctx, &apps, client.InNamespace("argocd"), client.MatchingFields{"metadata.name": appName})
	if err != nil {
		return fmt.Errorf("watch: %w", err)
	}
	defer w.Stop()

	log.Printf("Watching application %s", appName)
	for {
		select {
		case event := <-w.ResultChan():
			if event.Type == watch.Error {
				return fmt.Errorf("watch error: %+v", event.Object)
			}
			if event.Type == watch.Added || event.Type == watch.Modified {
				app, ok := event.Object.(*argocdv1alpha1.Application)
				if !ok {
					return fmt.Errorf("got unknown object %#v", event.Object)
				}
				actualStatus := newApplicationStatus(app)
				if diff := cmp.Diff(expected, actualStatus); diff != "" {
					log.Printf("Application %s status is not expected:\n%s", appName, diff)
					continue
				}
				log.Printf("Application %s status is expected", appName)
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type ApplicationStatus struct {
	Revision  string
	Sync      string
	Operation string
	Health    string
}

func newApplicationStatus(app *argocdv1alpha1.Application) ApplicationStatus {
	s := ApplicationStatus{
		Revision: app.Status.Sync.Revision,
		Sync:     string(app.Status.Sync.Status),
		Health:   string(app.Status.Health.Status),
	}
	if app.Status.OperationState != nil {
		s.Operation = string(app.Status.OperationState.Phase)
	}
	return s
}
