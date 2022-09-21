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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type options struct {
	appNames []string
	timeout  time.Duration
	want     ApplicationStatus
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	var o options
	flag.StringVar(&o.want.Revision, "revision", "", "Want revision")
	flag.StringVar(&o.want.Sync, "sync", "Synced", "Want sync status")
	flag.StringVar(&o.want.Operation, "operation", "Succeeded", "Want operation status")
	flag.StringVar(&o.want.Health, "health", "Healthy", "Want health status")
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

	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("could not get Kubernetes config: %w", err)
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
	for i := range o.appNames {
		appName := o.appNames[i]
		fns = append(fns, func() error {
			if err := watchApplicationStatus(ctx, k8sClient, appName, o.want); err != nil {
				return fmt.Errorf("watchApplicationStatus: %w", err)
			}
			return nil
		})
	}
	return errors.AggregateGoroutines(fns...)
}

func watchApplicationStatus(ctx context.Context, c client.WithWatch, appName string, want ApplicationStatus) error {
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
				return fmt.Errorf("watch error: %#v", event.Object)
			}
			if event.Type == watch.Added || event.Type == watch.Modified {
				app, ok := event.Object.(*argocdv1alpha1.Application)
				if !ok {
					return fmt.Errorf("got unknown object %#v", event.Object)
				}
				got := newApplicationStatus(app)
				if diff := cmp.Diff(want, got); diff != "" {
					log.Printf("Application %s status mismatch (-want +got):\n%s", appName, diff)
					continue
				}
				log.Printf("Application %s is expected status", appName)
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
