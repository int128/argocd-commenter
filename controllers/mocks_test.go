package controllers

import (
	"context"
	"sync"

	"github.com/int128/argocd-commenter/pkg/notification"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type EventRecorder struct {
	m       sync.Mutex
	counter map[string]int
}

func (r *EventRecorder) CountBy(key types.NamespacedName) int {
	r.m.Lock()
	defer r.m.Unlock()
	return r.counter[key.String()]
}

func (r *EventRecorder) call(event notification.Event) int {
	r.m.Lock()
	defer r.m.Unlock()

	if r.counter == nil {
		r.counter = make(map[string]int)
	}
	key := types.NamespacedName{Namespace: event.Application.Namespace, Name: event.Application.Name}
	r.counter[key.String()]++
	return r.counter[key.String()]
}

type NotificationMock struct {
	Comments           EventRecorder
	DeploymentStatuses EventRecorder
}

func (n *NotificationMock) Comment(ctx context.Context, event notification.Event) error {
	logger := log.FromContext(ctx)
	nth := n.Comments.call(event)
	logger.Info("called Comment", "nth", nth)
	return nil
}

func (n *NotificationMock) Deployment(ctx context.Context, event notification.Event) error {
	logger := log.FromContext(ctx)
	nth := n.DeploymentStatuses.call(event)
	logger.Info("called Deployment", "nth", nth)
	return nil
}
