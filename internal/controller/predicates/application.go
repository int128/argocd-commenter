package predicates

import (
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ApplicationUpdateComparer interface {
	Compare(applicationOld, applicationNew argocdv1alpha1.Application) bool
}

func ApplicationUpdate(c ApplicationUpdateComparer) predicate.Predicate {
	return &applicationUpdate{c}
}

type applicationUpdate struct {
	ApplicationUpdateComparer
}

func (applicationUpdate) Create(event.CreateEvent) bool {
	return false
}

func (applicationUpdate) Delete(event.DeleteEvent) bool {
	return false
}

func (p applicationUpdate) Update(e event.UpdateEvent) bool {
	applicationOld, ok := e.ObjectOld.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	applicationNew, ok := e.ObjectNew.(*argocdv1alpha1.Application)
	if !ok {
		return false
	}
	return p.Compare(*applicationOld, *applicationNew)
}

func (applicationUpdate) Generic(event.GenericEvent) bool {
	return false
}
