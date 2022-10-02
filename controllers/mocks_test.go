package controllers

import (
	"context"
	"sync"

	argocdcommenterv1 "github.com/int128/argocd-commenter/api/v1"
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

func (r *EventRecorder) call(key types.NamespacedName) int {
	r.m.Lock()
	defer r.m.Unlock()

	if r.counter == nil {
		r.counter = make(map[string]int)
	}
	r.counter[key.String()]++
	return r.counter[key.String()]
}

type NotificationMock struct {
	Comments              EventRecorder
	DeploymentStatuses    EventRecorder
	InactivateDeployments EventRecorder
}

func (n *NotificationMock) Comment(ctx context.Context, event notification.Event) error {
	logger := log.FromContext(ctx)
	key := types.NamespacedName{Namespace: event.Application.Namespace, Name: event.Application.Name}
	nth := n.Comments.call(key)
	logger.Info("called Comment", "nth", nth)
	return nil
}

func (n *NotificationMock) Deployment(ctx context.Context, event notification.Event) error {
	logger := log.FromContext(ctx)
	key := types.NamespacedName{Namespace: event.Application.Namespace, Name: event.Application.Name}
	nth := n.DeploymentStatuses.call(key)
	logger.Info("called Deployment", "nth", nth)
	return nil
}

func (n *NotificationMock) InactivateDeployment(ctx context.Context, appHealth argocdcommenterv1.ApplicationHealth) error {
	logger := log.FromContext(ctx)
	key := types.NamespacedName{Namespace: appHealth.Namespace, Name: appHealth.Name}
	nth := n.InactivateDeployments.call(key)
	logger.Info("called InactivateDeployment", "nth", nth)
	return nil
}

var _ notification.Client = &NotificationMock{}
